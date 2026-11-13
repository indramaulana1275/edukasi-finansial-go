package services

import (
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

const (
	// Daftar berita disimpan satu jam: judul berita ekonomi tidak berubah tiap
	// menit, jadi satu permintaan halaman tidak perlu memanggil Google News lagi.
	insightCacheTTL = time.Hour
	// Daftar cadangan hanya sebentar — begitu internetnya kembali, halaman tidak
	// menunggu satu jam untuk berita sungguhan.
	insightFallbackCacheTTL = 5 * time.Minute

	// Berapa banyak kartu berita di halaman Insight Global. Nilainya dipakai dua
	// jalur sekaligus: membatasi hasil gabungan feed RSS, dan menentukan panjang
	// daftar contoh di bawah — jadi dua daftar itu selalu sepanjang ini.
	insightMaxItems = 32

	// Feed yang menghasilkan lebih sedikit dari ini dianggap sedang limit, lalu
	// daftar contoh yang dipakai supaya halaman tetap berisi minimal 30 berita.
	insightMinItems = 30

	// Tiap kategori diambil lebih banyak dari jatahnya, karena satu judul bisa
	// muncul di dua pencarian dan yang kembar dibuang setelah digabung.
	insightFeedPerTopicSlack = 2

	// Penanda dari mana daftarnya datang; dikirim apa adanya ke frontend supaya
	// halaman bisa jujur soal sumbernya.
	insightSourceFeed     = "google_news"
	insightSourceFallback = "contoh"

	// Feed RSS Google News: gratis dan tanpa API key (GNews/NewsAPI keduanya
	// butuh key). Satu pencarian per kategori membuat label berita langsung
	// cocok dengan filter Saham/SBSN/Logam Mulia di halaman.
	insightFeedBaseURL = "https://news.google.com/rss/search"
)

// insightFeedTopic mengikat satu pencarian Google News ke kategori yang dipakai
// filter halaman. Kategori ditentukan oleh topik pencarian, bukan ditebak dari
// judul, supaya setiap berita pasti masuk salah satu filter.
type insightFeedTopic struct {
	Category string
	Query    string
}

var insightFeedTopics = []insightFeedTopic{
	{Category: "Saham", Query: "\"Saham Indonesia\" OR IHSG"},
	{Category: "SBSN", Query: "\"SBSN\" OR \"Sukuk\""},
	{Category: "Logam Mulia", Query: "\"Harga Emas\""},
}

// insightFeedTopicsEN memakai pencarian berbahasa Inggris saat halaman dibuka
// dalam mode EN, sehingga judul yang tampil juga bahasa Inggris. Kategori tetap
// sama supaya filter halaman tidak perlu daftar terjemahan sendiri.
var insightFeedTopicsEN = []insightFeedTopic{
	{Category: "Saham", Query: "\"Indonesia stocks\" OR IHSG OR IDX"},
	{Category: "SBSN", Query: "\"SBSN\" OR \"Sukuk\""},
	{Category: "Logam Mulia", Query: "\"gold price\""},
}

// insightTopicsFor memilih daftar pencarian sesuai bahasa halaman.
func insightTopicsFor(language string) []insightFeedTopic {
	if normaliseChatLanguage(language) == "en" {
		return insightFeedTopicsEN
	}

	return insightFeedTopics
}

// InsightItem adalah satu berita ekonomi global di halaman Insight Global.
type InsightItem struct {
	ID       string
	Title    string
	Category string
	Date     string
	Summary  string
}

// insightList adalah isi halaman Insight Global yang di-cache: beritanya plus
// sumber yang benar-benar melayaninya — sama seperti historySeries di market,
// sumbernya ikut dibawa bersama datanya supaya tidak ada tebakan.
type insightList struct {
	Items  []InsightItem
	Source string
}

type InsightService struct{}

func NewInsightService() *InsightService {
	return &InsightService{}
}

// List mengembalikan berita yang sedang dipakai halaman Insight Global beserta
// sumbernya ("google_news" atau "contoh"). Daftar disimpan per bahasa supaya
// mode EN tidak memakai judul hasil pencarian Indonesia.
func (s *InsightService) List(language string) ([]InsightItem, string) {
	list := s.cached(language)

	return list.Items, list.Source
}

// Find mencari satu berita di daftar yang sedang dipakai — id lebih dulu, lalu
// judulnya. Karena yang dicari selalu ada di daftar halaman, endpoint penjelasan
// AI tidak bisa dipakai untuk membahas teks sembarangan.
func (s *InsightService) Find(id, title, language string) (InsightItem, bool) {
	id = strings.TrimSpace(id)
	title = strings.ToLower(strings.TrimSpace(title))

	for _, item := range s.cached(language).Items {
		if id != "" && item.ID == id {
			return item, true
		}

		if title != "" && strings.ToLower(item.Title) == title {
			return item, true
		}
	}

	return InsightItem{}, false
}

// cached membaca daftar berita dari cache, atau menyusunnya sekali kalau belum
// ada. Daftar cadangan diberi masa simpan pendek supaya kegagalan feed sesaat
// tidak mengunci halaman pada berita contoh selama satu jam.
func (s *InsightService) cached(language string) insightList {
	cacheKey := "insight_news_v2_" + normaliseChatLanguage(language)

	if cached := facades.Cache().Get(cacheKey); cached != nil {
		if list, ok := cached.(insightList); ok {
			return list
		}
	}

	list := fetchInsightFeed(language)

	ttl := insightCacheTTL
	if list.Source == insightSourceFallback {
		ttl = insightFallbackCacheTTL
	}

	if err := facades.Cache().Put(cacheKey, list, ttl); err != nil {
		facades.Log().Warningf("gagal menyimpan cache berita: %v", err)
	}

	return list
}

// fetchInsightFeed mengambil berita dari feed RSS; kalau tidak ada satu pun yang
// berhasil, daftar contoh yang dipakai supaya halaman Insight Global tidak pernah
// kosong. Bisa dimatikan lewat INSIGHT_FEED=false untuk demo tanpa internet.
func fetchInsightFeed(language string) insightList {
	if !facades.Config().GetBool("insight.feed_enabled", true) {
		return insightList{Items: contohInsightItems(), Source: insightSourceFallback}
	}

	topics := insightTopicsFor(language)

	type result struct {
		items []InsightItem
		err   error
	}

	// Setiap kategori diambil lebih banyak dari jatahnya karena judul kembar baru
	// dibuang setelah semua hasil digabung, lalu daftar dipotong ke insightMaxItems.
	perFeed := insightPerCategory() * insightFeedPerTopicSlack

	results := make([]result, len(topics))
	var waitGroup sync.WaitGroup

	for index, topic := range topics {
		waitGroup.Add(1)

		go func(slot int, feed insightFeedTopic) {
			defer waitGroup.Done()

			items, err := fetchNewsFeed(insightFeedURL(feed.Query, language), perFeed, feed.Category)
			results[slot] = result{items: items, err: err}
		}(index, topic)
	}

	waitGroup.Wait()

	groups := make([][]InsightItem, 0, len(results))
	for _, res := range results {
		if res.err != nil {
			facades.Log().Warningf("feed berita gagal dipakai: %v", res.err)
			continue
		}

		groups = append(groups, res.items)
	}

	items := mergeInsightItems(groups...)
	if len(items) < insightMinItems || !insightCoversEveryCategory(items) {
		return insightList{Items: contohInsightItems(), Source: insightSourceFallback}
	}

	return insightList{Items: items, Source: insightSourceFeed}
}

// insightPerCategory membagi jatah kartu secara merata ke semua kategori feed,
// supaya filter tidak pernah kehabisan berita hanya karena satu topik mendominasi.
func insightPerCategory() int {
	if len(insightFeedTopics) == 0 {
		return insightMaxItems
	}

	return (insightMaxItems + len(insightFeedTopics) - 1) / len(insightFeedTopics)
}

// insightCoversEveryCategory memastikan satu feed kategori yang gagal tidak
// meninggalkan tombol filter tanpa isi; daftar contoh dipakai bila ada yang bolong.
func insightCoversEveryCategory(items []InsightItem) bool {
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		seen[item.Category] = true
	}

	for _, topic := range insightFeedTopics {
		if !seen[topic.Category] {
			return false
		}
	}

	return true
}

// insightFeedURL merangkai URL pencarian Google News untuk satu topik. Parameter
// hl/gl/ceid ikut bahasa halaman supaya feed EN benar-benar berbahasa Inggris.
func insightFeedURL(query, language string) string {
	if normaliseChatLanguage(language) == "en" {
		return insightFeedBaseURL + "?q=" + url.QueryEscape(query) + "&hl=en&gl=US&ceid=US:en"
	}

	return insightFeedBaseURL + "?q=" + url.QueryEscape(query) + "&hl=id&gl=ID&ceid=ID:id"
}

// rssFeed hanya mengambil bagian yang dipakai dari RSS Google News; description
// dan link-nya sengaja diabaikan karena isinya cuma tautan, bukan ringkasan.
// Source ikut dibaca karena nama medianya kadang sendiri memuat " - "
// (mis. "Bisnis.com - Market"), sehingga judulnya tidak bisa dipotong dengan
// menebak dari tanda hubung saja.
type rssFeed struct {
	Channel struct {
		Items []struct {
			Title   string `xml:"title"`
			PubDate string `xml:"pubDate"`
			Source  string `xml:"source"`
		} `xml:"item"`
	} `xml:"channel"`
}

// fetchNewsFeed membaca satu feed dan menyaringnya jadi daftar berita siap pakai.
// Kategori diambil dari topik pencarian, bukan ditebak dari judul, karena feed
// Google News tidak membawa label topik yang bisa dipercaya.
func fetchNewsFeed(feedURL string, limit int, category string) ([]InsightItem, error) {
	response, err := facades.Http().
		FlushHeaders().
		WithHeader("User-Agent", marketUserAgent).
		Get(feedURL)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", feedURL, err)
	}

	if response.Failed() {
		return nil, fmt.Errorf("request to %s returned status %d", feedURL, response.Status())
	}

	body, err := response.Body()
	if err != nil {
		return nil, fmt.Errorf("could not read response from %s: %w", feedURL, err)
	}

	var feed rssFeed
	if err := xml.Unmarshal([]byte(body), &feed); err != nil {
		return nil, fmt.Errorf("could not decode feed from %s: %w", feedURL, err)
	}

	items := make([]InsightItem, 0, limit)
	for _, entry := range feed.Channel.Items {
		if len(items) >= limit {
			break
		}

		title, media := splitNewsTitle(entry.Title, entry.Source)
		if title == "" {
			continue
		}

		items = append(items, InsightItem{
			ID:       newsID(title),
			Title:    title,
			Category: category,
			Date:     newsDate(entry.PubDate),
			Summary:  newsSummary(media),
		})
	}

	return items, nil
}

// mergeInsightItems menggabungkan hasil beberapa feed: judul kembar dibuang
// supaya satu berita tidak muncul sebagai dua kartu. Tiap kategori diberi jatah
// yang sama lebih dulu, lalu sisa slot diisi kandidat terbaru dari kategori mana
// pun, dan hasil akhirnya diurutkan dari tanggal terbaru.
func mergeInsightItems(groups ...[]InsightItem) []InsightItem {
	quota := insightPerCategory()
	selected := make([]InsightItem, 0, insightMaxItems)
	overflow := make([]InsightItem, 0)
	seen := make(map[string]bool)
	counts := make(map[string]int)

	for _, group := range groups {
		for _, item := range sortInsightItems(group) {
			key := strings.ToLower(item.Title)
			if seen[key] {
				continue
			}

			seen[key] = true
			if counts[item.Category] < quota {
				counts[item.Category]++
				selected = append(selected, item)
				continue
			}

			overflow = append(overflow, item)
		}
	}

	selected = sortInsightItems(selected)
	for _, item := range sortInsightItems(overflow) {
		if len(selected) >= insightMaxItems {
			break
		}

		selected = append(selected, item)
	}

	selected = sortInsightItems(selected)
	if len(selected) > insightMaxItems {
		selected = selected[:insightMaxItems]
	}

	return selected
}

func sortInsightItems(items []InsightItem) []InsightItem {
	sorted := make([]InsightItem, len(items))
	copy(sorted, items)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Date > sorted[j].Date
	})

	return sorted
}

// splitNewsTitle memisahkan judul dari nama medianya. Google News menulisnya
// sebagai "Judul - Media", dan nama media itu tidak perlu ikut dikirim ke Gemini.
//
// Media dari elemen <source> dipakai lebih dulu bila judulnya memang diakhiri
// nama itu, karena beberapa media memakai tanda hubung di namanya sendiri
// ("Bisnis.com - Market") sehingga pemotongan dari belakang menyisakan potongan
// nama media di judul. Kalau tidak cocok, cara lama yang dipakai: potong di
// pemisah " - " terakhir.
func splitNewsTitle(raw, source string) (string, string) {
	raw = strings.TrimSpace(raw)
	source = strings.TrimSpace(source)

	if source != "" {
		if trimmed, ok := strings.CutSuffix(raw, " - "+source); ok {
			return strings.TrimSpace(trimmed), source
		}
	}

	index := strings.LastIndex(raw, " - ")
	if index <= 0 {
		return raw, ""
	}

	return strings.TrimSpace(raw[:index]), strings.TrimSpace(raw[index+3:])
}

// newsID memberi tiap berita penanda yang aman dipakai sebagai id elemen HTML.
// Judulnya sendiri tidak bisa dipakai karena mengandung spasi dan tanda baca.
func newsID(title string) string {
	hash := fnv.New32a()
	hash.Write([]byte(title))

	return fmt.Sprintf("gn-%08x", hash.Sum32())
}

// newsDate menormalkan pubDate feed (RFC1123) jadi YYYY-MM-DD, format yang sama
// dengan daftar contoh, supaya frontend dan pengurutan tanggal memakai satu
// bentuk saja. Tanggal yang gagal dibaca memakai tanggal hari ini.
func newsDate(raw string) string {
	if parsed, err := time.Parse(time.RFC1123, strings.TrimSpace(raw)); err == nil {
		return parsed.Format("2006-01-02")
	}

	return time.Now().Format("2006-01-02")
}

// newsSummary menyusun ringkasan sendiri karena description dari feed cuma
// berisi tautan dan nama media.
func newsSummary(media string) string {
	if strings.TrimSpace(media) == "" {
		return "Laporan berita ekonomi terkini."
	}

	return "Laporan " + media + "."
}

// contohInsightItems adalah daftar bawaan yang dipakai kalau feed tidak bisa
// dihubungi atau dimatikan, jadi halaman Insight Global tidak pernah kosong.
// Tanggalnya ikut ditulis tetap supaya urutannya tetap masuk akal.
func contohInsightItems() []InsightItem {
	items := make([]InsightItem, len(contohInsightNews))
	copy(items, contohInsightNews)

	return sortInsightItems(items)
}

var contohInsightNews = []InsightItem{
	{
		ID:       "ihsg-menguat-perbankan",
		Title:    "IHSG Ditutup Menguat Ditopang Saham Perbankan",
		Category: "Saham",
		Date:     "2026-11-06",
		Summary:  "Indeks Harga Saham Gabungan ditutup di zona hijau setelah saham bank besar menopang indeks. Volume perdagangan meningkat seiring membaiknya sentimen pasar domestik.",
	},
	{
		ID:       "dana-asing-saham",
		Title:    "Aliran Dana Asing Masuk ke Pasar Saham Indonesia",
		Category: "Saham",
		Date:     "2026-11-05",
		Summary:  "Investor asing mencatat pembelian bersih di pasar reguler. Masuknya dana asing membantu memperkuat rupiah dan memberi ruang penguatan pada saham berkapitalisasi besar.",
	},
	{
		ID:       "saham-energi-memimpin",
		Title:    "Saham Energi Memimpin Penguatan IHSG",
		Category: "Saham",
		Date:     "2026-11-04",
		Summary:  "Sektor energi memimpin kenaikan indeks setelah harga komoditas global stabil. Investor memilih emiten dengan pendapatan yang terkait langsung dengan harga energi.",
	},
	{
		ID:       "laba-bank-melampaui",
		Title:    "Laba Emiten Perbankan Melampaui Perkiraan",
		Category: "Saham",
		Date:     "2026-11-03",
		Summary:  "Sejumlah bank besar melaporkan laba di atas konsensus analis. Kualitas kredit yang membaik dan pertumbuhan kredit mendorong optimisme terhadap sektor perbankan.",
	},
	{
		ID:       "saham-konsumer-bertahan",
		Title:    "Saham Konsumer Bertahan saat Pasar Berfluktuasi",
		Category: "Saham",
		Date:     "2026-11-02",
		Summary:  "Saham sektor konsumer menunjukkan ketahanan ketika indeks bergerak volatil. Permintaan domestik yang stabil menjadi penopang kinerja emiten di sektor ini.",
	},
	{
		ID:       "bursa-asia-ihsg",
		Title:    "Bursa Asia Menguat, IHSG Ikut Terangkat",
		Category: "Saham",
		Date:     "2026-10-30",
		Summary:  "Penguatan bursa regional menular ke IHSG. Sentimen positif datang dari harapan pelonggaran kebijakan moneter dan stabilnya data ekonomi utama.",
	},
	{
		ID:       "infrastruktur-minat-investor",
		Title:    "Sektor Infrastruktur Menarik Minat Investor",
		Category: "Saham",
		Date:     "2026-10-28",
		Summary:  "Saham infrastruktur kembali dilirik setelah proyek strategis nasional berjalan sesuai jadwal. Arus kas yang lebih pasti menjadi daya tarik bagi investor jangka panjang.",
	},
	{
		ID:       "ihsg-uji-resistance",
		Title:    "IHSG Menguji Level Resistance Setelah Rebond",
		Category: "Saham",
		Date:     "2026-10-26",
		Summary:  "Indeks mendekati area resistance penting setelah rebound dari support. Pelaku pasar mencermati volume transaksi untuk melihat apakah penguatan berlanjut.",
	},
	{
		ID:       "teknologi-global-domestik",
		Title:    "Saham Teknologi Global Menguat, Sentimen Domestik Membaik",
		Category: "Saham",
		Date:     "2026-10-23",
		Summary:  "Penguatan saham teknologi di pasar global meningkatkan selera risiko. Sentimen itu ikut membantu saham teknologi domestik dan emiten terkait ekonomi digital.",
	},
	{
		ID:       "asing-beli-bersih",
		Title:    "Investor Asing Catat Beli Bersih di Pasar Reguler",
		Category: "Saham",
		Date:     "2026-10-21",
		Summary:  "Data perdagangan menunjukkan investor asing kembali melakukan pembelian bersih. Saham perbankan dan konsumer menjadi tujuan utama aliran dana tersebut.",
	},
	{
		ID:       "sbsn-ritel-baru",
		Title:    "Pemerintah Terbitkan SBSN Ritel Seri Baru",
		Category: "SBSN",
		Date:     "2026-11-06",
		Summary:  "Pemerintah membuka masa penawaran SBSN ritel seri baru. Instrumen ini menawarkan imbalan tetap dan dapat dibeli dengan nominal terjangkau oleh investor individu.",
	},
	{
		ID:       "sukuk-ritel-lampaui-target",
		Title:    "Permintaan Sukuk Ritel Melebihi Target Penjualan",
		Category: "SBSN",
		Date:     "2026-11-05",
		Summary:  "Minat investor terhadap sukuk ritel melampaui target awal. Imbalan yang kompetitif dan risiko gagal bayar yang rendah menjadi daya tarik utama.",
	},
	{
		ID:       "imbal-sbsn-stabil",
		Title:    "Imbal Hasil SBSN Stabil di Tengah Ketidakpastian Global",
		Category: "SBSN",
		Date:     "2026-11-03",
		Summary:  "Imbal hasil SBSN bergerak stabil meski pasar global berfluktuasi. Permintaan dari investor domestik membantu menahan gejolak harga di pasar sekunder.",
	},
	{
		ID:       "sukuk-pendapatan-tetap",
		Title:    "Sukuk Negara Jadi Alternatif Investasi Pendapatan Tetap",
		Category: "SBSN",
		Date:     "2026-10-31",
		Summary:  "Sukuk negara menjadi pilihan bagi investor yang menginginkan pendapatan tetap. Pembayaran imbalan berkala membuat arus kas lebih mudah direncanakan.",
	},
	{
		ID:       "lelang-sbsn-serapan",
		Title:    "Pemerintah Lelang SBSN, Serapan Sesuai Rencana",
		Category: "SBSN",
		Date:     "2026-10-29",
		Summary:  "Lelang SBSN memperoleh serapan yang sesuai dengan target pembiayaan. Partisipasi investor institusi tetap kuat di tengah kondisi pasar yang menantang.",
	},
	{
		ID:       "sukuk-hijau",
		Title:    "Sukuk Hijau Indonesia Dilirik Investor Asing",
		Category: "SBSN",
		Date:     "2026-10-27",
		Summary:  "Sukuk hijau Indonesia menarik perhatian investor asing yang mencari proyek berkelanjutan. Kerangka pembiayaan hijau dinilai memperluas basis investor negara.",
	},
	{
		ID:       "kupon-sbsn-ritel",
		Title:    "Kupon SBSN Ritel Menarik Minat Investor Ritel",
		Category: "SBSN",
		Date:     "2026-10-24",
		Summary:  "Kupon kompetitif membuat SBSN ritel diminati investor individu. Kemudahan pembelian melalui platform distribusi resmi turut mendorong partisipasi.",
	},
	{
		ID:       "obligasi-negara-stabil",
		Title:    "Pasar Obligasi Negara Tetap Stabil saat Rupiah Berfluktuasi",
		Category: "SBSN",
		Date:     "2026-10-22",
		Summary:  "Pasar obligasi negara bertahan stabil ketika rupiah bergerak. Investor domestik menjadi penopang utama permintaan di pasar sekunder.",
	},
	{
		ID:       "sbsn-konservatif",
		Title:    "SBSN Jadi Pilihan Konservatif di Tengah Suku Bunga Tinggi",
		Category: "SBSN",
		Date:     "2026-10-20",
		Summary:  "Investor konservatif memilih SBSN untuk menjaga nilai modal sambil memperoleh imbalan tetap. Risiko pasar instrumen ini relatif lebih rendah dibanding saham.",
	},
	{
		ID:       "penerbitan-sukuk-terjaga",
		Title:    "Pemerintah Jaga Penerbitan Sukuk Sesuai Kebutuhan Pembiayaan",
		Category: "SBSN",
		Date:     "2026-10-18",
		Summary:  "Penerbitan sukuk negara disesuaikan dengan kebutuhan pembiayaan dan kondisi pasar. Strategi ini membantu menjaga biaya utang tetap terkendali.",
	},
	{
		ID:       "emas-dolar-melemah",
		Title:    "Harga Emas Dunia Menguat Terdorong Pelemahan Dolar",
		Category: "Logam Mulia",
		Date:     "2026-11-06",
		Summary:  "Harga emas dunia naik setelah dolar AS melemah. Emas yang dihargai dalam dolar menjadi lebih murah bagi pemegang mata uang lain sehingga permintaan meningkat.",
	},
	{
		ID:       "antam-level-baru",
		Title:    "Emas Antam Naik ke Level Baru",
		Category: "Logam Mulia",
		Date:     "2026-11-05",
		Summary:  "Harga emas Antam mencatat kenaikan seiring penguatan harga emas global. Investor ritel tetap mencermati spread antara harga beli dan harga buyback.",
	},
	{
		ID:       "bank-sentral-cadangan-emas",
		Title:    "Bank Sentral Dunia Terus Menambah Cadangan Emas",
		Category: "Logam Mulia",
		Date:     "2026-11-04",
		Summary:  "Sejumlah bank sentral menambah cadangan emas untuk memperkuat ketahanan neraca. Permintaan institusi menjadi salah satu penopang harga emas jangka panjang.",
	},
	{
		ID:       "emas-ketidakpastian-global",
		Title:    "Harga Emas Berpotensi Diuntungkan Ketidakpastian Global",
		Category: "Logam Mulia",
		Date:     "2026-11-02",
		Summary:  "Ketidakpastian ekonomi dan geopolitik mendorong investor mencari aset lindung nilai. Emas kembali menjadi pilihan utama ketika selera risiko menurun.",
	},
	{
		ID:       "permintaan-emas-fisik",
		Title:    "Permintaan Emas Fisik Meningkat Menjelang Akhir Tahun",
		Category: "Logam Mulia",
		Date:     "2026-10-30",
		Summary:  "Permintaan emas batangan meningkat menjelang akhir tahun. Pembeli ritel memanfaatkan koreksi harga untuk menambah porsi emas sebagai tabungan jangka panjang.",
	},
	{
		ID:       "perak-mengikuti-emas",
		Title:    "Harga Perak Mengikuti Penguatan Emas",
		Category: "Logam Mulia",
		Date:     "2026-10-28",
		Summary:  "Harga perak bergerak mengikuti penguatan emas. Kedua logam mulia ini sering dipakai investor sebagai pelengkap portofolio lindung nilai.",
	},
	{
		ID:       "emas-lindung-inflasi",
		Title:    "Emas Jadi Aset Lindung Nilai saat Inflasi Tinggi",
		Category: "Logam Mulia",
		Date:     "2026-10-25",
		Summary:  "Emas diminati ketika inflasi tinggi karena dianggap mampu menjaga daya beli. Namun harganya tetap bisa berfluktuasi dalam jangka pendek.",
	},
	{
		ID:       "emas-koreksi-dolar",
		Title:    "Harga Emas Terkoreksi Seiring Penguatan Dolar AS",
		Category: "Logam Mulia",
		Date:     "2026-10-23",
		Summary:  "Harga emas terkoreksi ketika dolar AS menguat. Koreksi ini menjadi kesempatan bagi sebagian investor untuk membeli secara bertahap.",
	},
	{
		ID:       "ritel-tambah-emas",
		Title:    "Investor Ritel Mulai Menambah Porsi Emas",
		Category: "Logam Mulia",
		Date:     "2026-10-21",
		Summary:  "Investor ritel menambah porsi emas sebagai bagian dari diversifikasi. Porsi yang wajar membantu mengurangi dampak volatilitas aset lain di portofolio.",
	},
	{
		ID:       "spread-emas-batangan",
		Title:    "Spread Harga Emas Batangan Perlu Dicermati",
		Category: "Logam Mulia",
		Date:     "2026-10-19",
		Summary:  "Selisih harga beli dan buyback emas batangan memengaruhi keuntungan investor. Memahami spread membantu memilih bentuk emas yang sesuai tujuan investasi.",
	},
	{
		ID:       "saham-properti-suku-bunga",
		Title:    "Saham Properti Bangkit Seiring Harapan Penurunan Suku Bunga",
		Category: "Saham",
		Date:     "2026-10-20",
		Summary:  "Saham properti menguat ketika pasar berharap suku bunga turun. Biaya pendanaan yang lebih rendah berpotensi meningkatkan permintaan properti dan margin pengembang.",
	},
	{
		ID:       "emas-digital-fisik",
		Title:    "Emas Digital dan Fisik Sama-Sama Dicari Investor",
		Category: "Logam Mulia",
		Date:     "2026-10-18",
		Summary:  "Investor memilih emas digital untuk kemudahan transaksi dan emas fisik untuk kepemilikan langsung. Keduanya perlu disesuaikan dengan tujuan dan biaya masing-masing.",
	},
}
