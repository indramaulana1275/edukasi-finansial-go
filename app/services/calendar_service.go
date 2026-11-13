package services

import "strings"

// EconomicEvent adalah satu agenda ekonomi global di halaman Kalender Ekonomi.
//
// Daftarnya masih statis: tanggal, jam rilis, dan dampaknya ditulis tangan di
// economicCalendarSeeds. Halaman hanya memformat data ini, jadi membuka halaman
// tidak memanggil upstream mana pun dan tidak memakai kuota AI.
type EconomicEvent struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Time    string `json:"time"`
	Name    string `json:"name"`
	Country string `json:"country"`
	// Impact adalah label risiko volatilitas: "Tinggi", "Sedang", atau "Rendah".
	Impact string `json:"impact"`
	// Assets menyebut aset yang paling terasa: Emas, SBSN, Saham, USD.
	Assets []string `json:"assets"`
	// Note adalah satu kalimat sebab-akibat yang tampil di kartu.
	Note string `json:"note"`

	// Padanan bahasa Inggris disimpan terpisah dan tidak ikut JSON. List/Find
	// memilih pasangan yang sesuai bahasa permintaan, jadi klien hanya menerima
	// satu versi teks di field di atas.
	NameEN    string   `json:"-"`
	CountryEN string   `json:"-"`
	ImpactEN  string   `json:"-"`
	AssetsEN  []string `json:"-"`
	NoteEN    string   `json:"-"`
}

type CalendarService struct{}

func NewCalendarService() *CalendarService {
	return &CalendarService{}
}

// localized mengembalikan salinan event dalam bahasa yang diminta. Bahasa yang
// tidak dikenal diperlakukan sebagai Indonesia, sama seperti chat.
func (e EconomicEvent) localized(language string) EconomicEvent {
	if normaliseChatLanguage(language) != "en" {
		return e
	}

	if e.NameEN != "" {
		e.Name = e.NameEN
	}
	if e.CountryEN != "" {
		e.Country = e.CountryEN
	}
	if e.ImpactEN != "" {
		e.Impact = e.ImpactEN
	}
	if len(e.AssetsEN) > 0 {
		e.Assets = append([]string(nil), e.AssetsEN...)
	}
	if e.NoteEN != "" {
		e.Note = e.NoteEN
	}

	return e
}

// List mengembalikan daftar agenda apa adanya, sudah urut tanggal dan jam.
func (s *CalendarService) List(language string) []EconomicEvent {
	events := make([]EconomicEvent, len(economicCalendarSeeds))
	for index, event := range economicCalendarSeeds {
		events[index] = event.localized(language)
	}

	return events
}

// Find mencari satu agenda berdasarkan id. Endpoint penjelasan AI memakai ini
// supaya model hanya membahas agenda yang memang tampil di halaman, bukan teks
// bebas yang dikirim klien.
func (s *CalendarService) Find(id, language string) (EconomicEvent, bool) {
	id = strings.TrimSpace(id)

	for _, event := range economicCalendarSeeds {
		if event.ID == id {
			return event.localized(language), true
		}
	}

	return EconomicEvent{}, false
}

// economicCalendarSeeds sengaja diurutkan dari agenda terdekat. Menambah event
// cukup menambah satu entri; id dipakai sebagai kunci cache AI dan id elemen
// HTML, jadi harus unik dan tanpa spasi.
var economicCalendarSeeds = []EconomicEvent{
	{
		ID:        "fed-fomc-november",
		Date:      "2026-11-04",
		Time:      "02:00 WIB",
		Name:      "Keputusan Suku Bunga The Fed (FOMC)",
		NameEN:    "The Fed Interest Rate Decision (FOMC)",
		Country:   "Amerika Serikat",
		CountryEN: "United States",
		Impact:    "Tinggi",
		ImpactEN:  "High",
		Assets:    []string{"Emas", "SBSN", "Saham"},
		AssetsEN:  []string{"Gold", "SBSN", "Stocks"},
		Note:      "Suku bunga acuan menentukan biaya dana global; kenaikan menekan aset berisiko, penurunan mengangkatnya.",
		NoteEN:    "The benchmark interest rate sets the global cost of funds; a hike weighs on risk assets, while a cut lifts them.",
	},
	{
		ID:        "us-nfp-november",
		Date:      "2026-11-06",
		Time:      "20:30 WIB",
		Name:      "Data Tenaga Kerja AS (Non-Farm Payrolls)",
		NameEN:    "US Labour Data (Non-Farm Payrolls)",
		Country:   "Amerika Serikat",
		CountryEN: "United States",
		Impact:    "Tinggi",
		ImpactEN:  "High",
		Assets:    []string{"Emas", "SBSN", "Saham", "USD"},
		AssetsEN:  []string{"Gold", "SBSN", "Stocks", "USD"},
		Note:      "Pasar tenaga kerja kuat menahan harapan pemangkasan suku bunga, lemah justru sebaliknya.",
		NoteEN:    "A strong labour market dampens hopes of rate cuts; a weak one does the opposite.",
	},
	{
		ID:        "us-cpi-oktober",
		Date:      "2026-11-12",
		Time:      "20:30 WIB",
		Name:      "Rilis Inflasi Konsumen AS (CPI) Oktober",
		NameEN:    "US Consumer Inflation Release (CPI) October",
		Country:   "Amerika Serikat",
		CountryEN: "United States",
		Impact:    "Tinggi",
		ImpactEN:  "High",
		Assets:    []string{"Emas", "SBSN", "Saham", "USD"},
		AssetsEN:  []string{"Gold", "SBSN", "Stocks", "USD"},
		Note:      "Inflasi lebih tinggi dari perkiraan menunda pemangkasan suku bunga dan biasanya menguatkan dolar.",
		NoteEN:    "Inflation above expectations delays rate cuts and usually strengthens the dollar.",
	},
	{
		ID:        "bi-rate-november",
		Date:      "2026-11-19",
		Time:      "14:00 WIB",
		Name:      "Rapat Dewan Gubernur Bank Indonesia",
		NameEN:    "Bank Indonesia Board of Governors Meeting",
		Country:   "Indonesia",
		CountryEN: "Indonesia",
		Impact:    "Sedang",
		ImpactEN:  "Medium",
		Assets:    []string{"Saham", "USD"},
		AssetsEN:  []string{"Stocks", "USD"},
		Note:      "Keputusan BI menjaga stabilitas rupiah; suku bunga lebih tinggi cenderung menahan IHSG jangka pendek.",
		NoteEN:    "BI's decision protects rupiah stability; higher rates tend to hold back the IDX Composite in the short term.",
	},
	{
		ID:        "ecb-rate-desember",
		Date:      "2026-12-10",
		Time:      "20:15 WIB",
		Name:      "Keputusan Suku Bunga ECB",
		NameEN:    "ECB Interest Rate Decision",
		Country:   "Uni Eropa",
		CountryEN: "European Union",
		Impact:    "Sedang",
		ImpactEN:  "Medium",
		Assets:    []string{"Emas", "Saham", "USD"},
		AssetsEN:  []string{"Gold", "Stocks", "USD"},
		Note:      "Kebijakan ECB menggerakkan euro dan dolar; euro melemah biasanya menahan harga emas dalam dolar.",
		NoteEN:    "ECB policy moves the euro and the dollar; a weaker euro usually caps gold prices in dollars.",
	},
	{
		ID:        "us-cpi-november",
		Date:      "2026-12-11",
		Time:      "20:30 WIB",
		Name:      "Rilis Inflasi Konsumen AS (CPI) November",
		NameEN:    "US Consumer Inflation Release (CPI) November",
		Country:   "Amerika Serikat",
		CountryEN: "United States",
		Impact:    "Tinggi",
		ImpactEN:  "High",
		Assets:    []string{"Emas", "SBSN", "Saham", "USD"},
		AssetsEN:  []string{"Gold", "SBSN", "Stocks", "USD"},
		Note:      "Angka inflasi adalah masukan utama arah suku bunga, sehingga memicu lonjakan volatilitas lintas aset.",
		NoteEN:    "Inflation figures are a key input for the rate path, so they trigger volatility across asset classes.",
	},
	{
		ID:        "fed-fomc-desember",
		Date:      "2026-12-16",
		Time:      "02:00 WIB",
		Name:      "Keputusan Suku Bunga The Fed (FOMC) Desember",
		NameEN:    "The Fed Interest Rate Decision (FOMC) December",
		Country:   "Amerika Serikat",
		CountryEN: "United States",
		Impact:    "Tinggi",
		ImpactEN:  "High",
		Assets:    []string{"Emas", "SBSN", "Saham"},
		AssetsEN:  []string{"Gold", "SBSN", "Stocks"},
		Note:      "Selain keputusan bunga, proyeksi suku bunga (dot plot) sering mengubah ekspektasi pasar setahun ke depan.",
		NoteEN:    "Beyond the rate decision, the rate projections (dot plot) often reshape market expectations for the year ahead.",
	},
	{
		ID:        "opec-meeting-desember",
		Date:      "2026-12-20",
		Time:      "21:00 WIB",
		Name:      "Pertemuan OPEC+ Kuartal IV",
		NameEN:    "OPEC+ Q4 Meeting",
		Country:   "Organisasi Negara Pengekspor Minyak",
		CountryEN: "Organization of the Petroleum Exporting Countries",
		Impact:    "Sedang",
		ImpactEN:  "Medium",
		Assets:    []string{"Saham", "Emas"},
		AssetsEN:  []string{"Stocks", "Gold"},
		Note:      "Keputusan kuota produksi memengaruhi harga minyak, biaya energi emiten, dan permintaan aset lindung nilai.",
		NoteEN:    "Production quota decisions affect oil prices, issuers' energy costs, and demand for hedging assets.",
	},
}
