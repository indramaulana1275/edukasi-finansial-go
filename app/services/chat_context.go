package services

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// Konteks yang dikirim frontend bersama satu pertanyaan chat. Batasnya kecil
// dengan sengaja: seluruh isinya masuk ke system prompt, jadi teks yang panjang
// hanya menghabiskan token (dan kuota Gemini) tanpa menambah informasi.
const (
	// Cukup untuk seluruh daftar aset aplikasi: 10 koin Crypto, 10 kurs, 20
	// saham, 30 komoditas, plus tiga kartu Dasbor. Sebelumnya 12, yang membuat
	// FINANIKA hanya melihat Bitcoin, Emas, dan USD/IDR.
	chatContextMaxPrices         = 90
	chatContextMaxNews           = 5
	chatContextMaxPlatforms      = 40
	chatContextTextLimit         = 140
	chatContextPlatformDescLimit = 160

	// Label aset yang dikirim ke prompt boleh lebih panjang daripada label
	// kiriman klien karena isinya berasal dari tabel aplikasi sendiri.
	chatAssetLabelLimit = 60
)

// ChatPrice adalah satu baris harga yang sedang tampil di layar pengguna,
// mis. {Asset: "BTC", Price: 81100.5, Unit: "USD", Change: 2.1}.
type ChatPrice struct {
	Asset  string  `json:"asset"`
	Price  float64 `json:"price"`
	Unit   string  `json:"unit"`
	Change float64 `json:"change"`
}

// ChatNews adalah satu judul dari Insight Global yang ikut sebagai konteks.
type ChatNews struct {
	Title    string `json:"title"`
	Category string `json:"category"`
}

// ChatTurn adalah satu giliran percakapan sebelumnya. Riwayat ini dikirim ulang
// ke model supaya pertanyaan lanjutan ("kalau biayanya?", "kenapa begitu?")
// tetap nyambung dengan jawaban sebelumnya.
type ChatTurn struct {
	Role    string
	Content string
}

// ChatHistoryTurns membatasi giliran yang dikirim ulang: enam giliran berarti
// tiga pasang tanya-jawab terakhir, cukup untuk merujuk "platform itu" atau
// "aset tadi" tanpa menggandakan biaya token setiap pertanyaan.
const (
	ChatHistoryTurns     = 6
	chatHistoryTextLimit = 600
)

// cleanChatHistory membuang giliran yang tidak dikenal, merapikan spasinya, dan
// memotong daftarnya dari yang paling baru. Tanpa pembersihan ini, jawaban AI
// yang panjang bisa memenuhi prompt dan menggeser pertanyaan yang sedang dijawab.
func cleanChatHistory(turns []ChatTurn) []ChatTurn {
	if len(turns) > ChatHistoryTurns {
		turns = turns[len(turns)-ChatHistoryTurns:]
	}

	cleaned := make([]ChatTurn, 0, len(turns))
	for _, turn := range turns {
		if turn.Role != models.ChatRoleUser && turn.Role != models.ChatRoleAssistant {
			continue
		}

		content := cleanChatText(turn.Content, chatHistoryTextLimit)
		if content == "" {
			continue
		}

		cleaned = append(cleaned, ChatTurn{Role: turn.Role, Content: content})
	}

	return cleaned
}

// ChatPlatform adalah satu platform jual-beli di halaman Rekomendasi Platform.
// Daftarnya hanya ada di view (PLATFORMS), jadi bagian inilah yang benar-benar
// baru bagi server — harga, profil, dan berita sudah dimiliki server sendiri.
type ChatPlatform struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ChatMode menentukan kedalaman jawaban chat. Nilai dikirim frontend bersama
// pertanyaan dan dinormalisasi di server, sehingga nilai asing selalu jatuh ke
// mode Medium (perilaku lama).
type ChatMode string

const (
	ChatModeQuick    ChatMode = "cepat"
	ChatModeMedium   ChatMode = "medium"
	ChatModeAnalysis ChatMode = "analisis"
)

// NormaliseChatMode memetakan nilai kiriman klien ke salah satu dari tiga mode.
// Alias bahasa Inggris diterima supaya API tetap ramah untuk pemanggil langsung.
func NormaliseChatMode(mode string) ChatMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "cepat", "quick", "fast":
		return ChatModeQuick
	case "analisis", "analysis", "deep":
		return ChatModeAnalysis
	default:
		return ChatModeMedium
	}
}

// ChatContext adalah potret data aplikasi yang dikirim frontend bersama
// pertanyaan chat. Semua bagian opsional: chat harus tetap jalan kalau frontend
// lama belum mengirim konteks, dan server mengisinya sendiri.
type ChatContext struct {
	Profile   string         `json:"profile"`
	Language  string         `json:"language"`
	Mode      ChatMode       `json:"-"`
	Prices    []ChatPrice    `json:"prices"`
	News      []ChatNews     `json:"news"`
	Platforms []ChatPlatform `json:"platforms"`
}

// chatLanguage adalah satu bahasa jawaban yang didukung tombol bahasa di
// topbar. Closing adalah padanan baris penutup "Saran:" di bahasa itu, supaya
// jawaban berbahasa Inggris tidak diakhiri label Indonesia.
type chatLanguage struct {
	Name    string
	Closing string
}

var chatLanguages = map[string]chatLanguage{
	"id": {Name: "Bahasa Indonesia", Closing: "Saran:"},
	"en": {Name: "English", Closing: "Suggestion:"},
}

// normaliseChatLanguage mengembalikan kode bahasa yang dikenal, atau "id" untuk
// apa pun yang tidak dikenal — termasuk permintaan lama tanpa field language.
func normaliseChatLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if _, ok := chatLanguages[lang]; ok {
		return lang
	}

	return "id"
}

// chatLanguageInstruction menegaskan bahasa jawaban untuk kedua pilihan bahasa.
// Perintahnya eksplisit supaya model tidak jatuh kembali ke bahasa persona saat
// pengguna memilih English.
func chatLanguageInstruction(lang string) string {
	code := normaliseChatLanguage(lang)
	language := chatLanguages[code]

	label := "bahasa Indonesia (ID)"
	if code == "en" {
		label = "bahasa Inggris (EN)"
	}

	return fmt.Sprintf("Jawab menggunakan %s sesuai pilihan user saat ini. Tulis seluruh jawaban dalam bahasa itu, termasuk baris penutup \"%s\".\n", label, language.Closing)
}

// PrepareChatContext membersihkan konteks kiriman frontend lalu melengkapinya
// dengan data milik server.
//
// Konteks dari klien diperlakukan sebagai data tampilan, bukan kebenaran: setiap
// teks dipotong dan dirapikan (lihat cleanChatText), dan bagian yang kosong atau
// tidak masuk akal diisi dari marketData / berita yang sudah diambil controller.
// Profil risiko tetap ditentukan pemanggil dari tabel risk_profiles, karena hanya
// server yang tahu hasil kuis terakhir user.
func PrepareChatContext(sent *ChatContext, profilRisiko string, marketData map[string]interface{}, assetPrices []ChatPrice, berita []InsightItem) ChatContext {
	chat := ChatContext{Profile: normaliseChatProfile(profilRisiko), Language: "id", Mode: ChatModeMedium}

	var sentPrices []ChatPrice
	if sent != nil {
		chat.Language = normaliseChatLanguage(sent.Language)
		sentPrices = cleanChatPrices(sent.Prices)
		chat.News = cleanChatNews(sent.News)
		chat.Platforms = cleanChatPlatforms(sent.Platforms)
	}

	// Harga milik server selalu jadi pangkal: tiga kartu Dasbor plus seluruh
	// daftar Crypto, Kurs, Saham, dan Komoditas dari cache/API internal. Baris
	// kiriman frontend hanya menambah aset yang belum dicakup server, sehingga
	// angka di prompt tidak bisa diganti dari browser.
	serverPrices := mergeChatPrices(chatPricesFromMarket(marketData), assetPrices)
	chat.Prices = mergeChatPrices(serverPrices, sentPrices)

	// Berita milik server juga jadi pangkal: daftar Insight Global yang sedang
	// berlaku sudah di-cache di server, jadi judul kiriman browser (yang bisa
	// sudah basi atau tidak nyambung dengan daftar yang tampil) hanya dipakai
	// kalau server sama sekali tidak punya berita.
	if news := chatNewsFromInsight(berita); len(news) > 0 {
		chat.News = news
	}

	return chat
}

// mergeChatPrices menambahkan harga kiriman frontend yang belum ada di daftar
// server, dicocokkan lewat kata pertama label aset ("BTC", "Emas", "USD/IDR").
func mergeChatPrices(fromServer, sent []ChatPrice) []ChatPrice {
	merged := make([]ChatPrice, 0, len(fromServer)+len(sent))
	known := make(map[string]bool, len(fromServer))

	for _, price := range fromServer {
		if len(merged) == chatContextMaxPrices {
			return merged
		}

		known[chatPriceKey(price.Asset)] = true
		merged = append(merged, price)
	}

	for _, price := range sent {
		if len(merged) == chatContextMaxPrices {
			break
		}
		if known[chatPriceKey(price.Asset)] {
			continue
		}

		merged = append(merged, price)
	}

	return merged
}

// chatPriceKey memakai kata pertama label sebagai kunci, karena label dari
// klien bebas: "BTC" dan "BTC (Bitcoin)" harus dianggap aset yang sama.
func chatPriceKey(asset string) string {
	fields := strings.Fields(asset)
	if len(fields) == 0 {
		return ""
	}

	return strings.ToLower(fields[0])
}

// normaliseChatProfile menerima hanya tiga tipe risiko yang dikenal. Nilai lain
// (mis. teks bebas dari klien) jatuh ke default aplikasi.
func normaliseChatProfile(profilRisiko string) string {
	switch strings.TrimSpace(profilRisiko) {
	case models.RisikoKonservatif:
		return models.RisikoKonservatif
	case models.RisikoAgresif:
		return models.RisikoAgresif
	default:
		return models.RisikoModerat
	}
}

// cleanChatPrices membuang baris yang tidak bisa dipakai model: harga nol,
// negatif, atau NaN, serta label kosong.
func cleanChatPrices(prices []ChatPrice) []ChatPrice {
	cleaned := make([]ChatPrice, 0, len(prices))

	for _, price := range prices {
		asset := cleanChatLabel(price.Asset, 40)
		if asset == "" || !finiteChatNumber(price.Price) || price.Price <= 0 {
			continue
		}

		change := price.Change
		if !finiteChatNumber(change) {
			change = 0
		}

		cleaned = append(cleaned, ChatPrice{
			Asset:  asset,
			Price:  price.Price,
			Unit:   cleanChatLabel(price.Unit, 20),
			Change: change,
		})

		if len(cleaned) == chatContextMaxPrices {
			break
		}
	}

	return cleaned
}

func cleanChatNews(news []ChatNews) []ChatNews {
	cleaned := make([]ChatNews, 0, len(news))

	for _, item := range news {
		title := cleanChatText(item.Title, chatContextTextLimit)
		if title == "" {
			continue
		}

		cleaned = append(cleaned, ChatNews{
			Title:    title,
			Category: cleanChatText(item.Category, 30),
		})

		if len(cleaned) == chatContextMaxNews {
			break
		}
	}

	return cleaned
}

func cleanChatPlatforms(platforms []ChatPlatform) []ChatPlatform {
	cleaned := make([]ChatPlatform, 0, len(platforms))
	seen := make(map[string]bool, len(platforms))

	for _, platform := range platforms {
		name := cleanChatLabel(platform.Name, 60)
		if name == "" || seen[strings.ToLower(name)] {
			continue
		}

		seen[strings.ToLower(name)] = true
		cleaned = append(cleaned, ChatPlatform{
			Name:        name,
			Category:    cleanChatText(platform.Category, 20),
			Type:        cleanChatText(platform.Type, 20),
			Description: cleanChatText(platform.Description, chatContextPlatformDescLimit),
		})

		if len(cleaned) == chatContextMaxPlatforms {
			break
		}
	}

	return cleaned
}

// cleanChatText merapatkan spasi dan baris baru lalu memotong panjangnya. Baris
// baru sengaja dihilangkan: konteks ini disisipkan ke system prompt, jadi teks
// yang bisa memecah struktur prompt tidak boleh lolos.
func cleanChatText(value string, limit int) string {
	joined := strings.Join(strings.Fields(value), " ")
	runes := []rune(joined)
	if len(runes) > limit {
		return string(runes[:limit])
	}

	return joined
}

// cleanChatLabel membersihkan label yang tampil sebagai nama aset atau platform:
// hanya huruf, angka, spasi, dan tanda baca lazim yang dipertahankan, supaya
// teks kiriman klien tidak bisa menyelipkan kalimat perintah ke system prompt.
func cleanChatLabel(value string, limit int) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsSpace(r):
			return r
		case strings.ContainsRune("()/.,%-+&:", r):
			return r
		default:
			return -1
		}
	}, value)

	return cleanChatText(cleaned, limit)
}

func finiteChatNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// chatPricesFromMarket merakit tiga kartu Dasbor menjadi konteks harga, dipakai
// saat frontend tidak mengirim harga (mis. chat dibuka sebelum /api/market selesai).
func chatPricesFromMarket(marketData map[string]interface{}) []ChatPrice {
	prices := make([]ChatPrice, 0, 3)

	if bitcoin, ok := toMap(marketData["bitcoin"]); ok {
		if price := toFloat64(bitcoin["price_usd"]); price > 0 {
			prices = append(prices, ChatPrice{
				Asset:  "BTC",
				Price:  price,
				Unit:   "USD",
				Change: toFloat64(bitcoin["change_24h"]),
			})
		}
	}

	if gold, ok := toMap(marketData["gold"]); ok {
		if price := toFloat64(gold["price_usd_oz"]); price > 0 {
			prices = append(prices, ChatPrice{
				Asset:  "Emas",
				Price:  price,
				Unit:   "USD/oz",
				Change: toFloat64(gold["change_percent"]),
			})
		}
	}

	if usd, ok := toMap(marketData["usd"]); ok {
		if rate := toFloat64(usd["idr_rate"]); rate > 0 {
			prices = append(prices, ChatPrice{
				Asset:  "USD/IDR",
				Price:  rate,
				Unit:   "IDR",
				Change: toFloat64(usd["change_percent"]),
			})
		}
	}

	return prices
}

// ChatPricesFromAssets merangkum seluruh daftar aset yang tampil di aplikasi —
// sepuluh koin Crypto, sepuluh kurs terhadap rupiah, dua puluh saham, dan tiga
// puluh komoditas — menjadi baris harga untuk prompt AI. Dengan ini FINANIKA
// tidak lagi hanya mengenal Bitcoin, Emas, dan USD/IDR: setiap aset yang bisa
// dibuka pengguna di halaman lain ikut punya harga acuan.
//
// Label dimulai dengan simbol unik (BTC, USD/IDR, BBCA.JK) supaya
// mergeChatPrices mengenali baris yang sama dengan kartu Dasbor. Nama panjang
// tetap disertakan agar pengguna bisa bertanya dengan nama aset.
func ChatPricesFromAssets(crypto, forex, saham, komoditas []map[string]any) []ChatPrice {
	prices := make([]ChatPrice, 0, len(crypto)+len(forex)+len(saham)+len(komoditas))

	for _, coin := range crypto {
		price := toFloat64(coin["price_usd"])
		label := chatAssetLabel(coin["symbol"], coin["name"])
		if price <= 0 || label == "" {
			continue
		}

		prices = append(prices, ChatPrice{
			Asset:  label,
			Price:  price,
			Unit:   "USD",
			Change: toFloat64(coin["change_24h"]),
		})
	}

	for _, rate := range forex {
		price := toFloat64(rate["rate"])
		label := chatAssetLabel(rate["pair"], rate["name"])
		if price <= 0 || label == "" {
			continue
		}

		prices = append(prices, ChatPrice{
			Asset:  label,
			Price:  price,
			Unit:   "IDR",
			Change: toFloat64(rate["change_percent"]),
		})
	}

	for _, stock := range saham {
		price := toFloat64(stock["price"])
		label := chatAssetLabel(stock["symbol"], stock["name"])
		if price <= 0 || label == "" {
			continue
		}

		prices = append(prices, ChatPrice{
			Asset:  label,
			Price:  price,
			Unit:   chatAssetUnit(toStr(stock["currency"]), ""),
			Change: toFloat64(stock["change_percent"]),
		})
	}

	for _, commodity := range komoditas {
		price := toFloat64(commodity["price"])
		label := chatAssetLabel(commodity["name"], commodity["symbol"])
		if price <= 0 || label == "" {
			continue
		}

		prices = append(prices, ChatPrice{
			Asset:  label,
			Price:  price,
			Unit:   chatAssetUnit(toStr(commodity["currency"]), toStr(commodity["unit"])),
			Change: toFloat64(commodity["change_percent"]),
		})
	}

	return prices
}

// chatAssetLabel menggabungkan simbol dan nama aset menjadi satu label yang
// aman dikirim ke prompt. Simbol ditaruh di depan untuk crypto, kurs, dan saham
// supaya kuncinya unik; nama tetap mudah dicari model.
func chatAssetLabel(primary, secondary any) string {
	first := cleanChatLabel(toStr(primary), chatAssetLabelLimit)
	second := cleanChatLabel(toStr(secondary), chatAssetLabelLimit)

	switch {
	case first == "":
		return second
	case second == "":
		return first
	}

	return cleanChatLabel(first+" ("+second+")", chatAssetLabelLimit)
}

// chatAssetUnit menulis satuan harga aset. Komoditas membawa satuan sendiri
// ("per troy ounce", "per kg"), sedangkan saham cukup mata uangnya.
func chatAssetUnit(currency, unit string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	unit = strings.TrimSpace(unit)

	switch {
	case currency == "" && unit == "":
		return "USD"
	case currency == "":
		return cleanChatLabel(unit, 20)
	case unit == "":
		return currency
	default:
		return cleanChatLabel(currency+" "+unit, 20)
	}
}

func chatNewsFromInsight(berita []InsightItem) []ChatNews {
	limit := len(berita)
	if limit > chatContextMaxNews {
		limit = chatContextMaxNews
	}

	news := make([]ChatNews, 0, limit)
	for _, item := range berita[:limit] {
		news = append(news, ChatNews{
			Title:    cleanChatText(item.Title, chatContextTextLimit),
			Category: cleanChatText(item.Category, 30),
		})
	}

	return news
}

// formatChatPrices menulis harga sebagai baris teks, bukan JSON, karena model
// hanya perlu membacanya dan JSON menambah tanda baca tanpa informasi baru.
func formatChatPrices(prices []ChatPrice) string {
	if len(prices) == 0 {
		return "(harga pasar sedang tidak tersedia)"
	}

	var builder strings.Builder
	for _, price := range prices {
		unit := price.Unit
		if unit == "" {
			unit = "USD"
		}

		fmt.Fprintf(&builder, "- %s: %s %s, change_24h: %s\n",
			price.Asset, formatChatNumber(price.Price), unit, formatChatChange(price.Change))
	}

	return strings.TrimRight(builder.String(), "\n")
}

func formatChatNews(news []ChatNews) string {
	if len(news) == 0 {
		return "(belum ada berita terbaru)"
	}

	var builder strings.Builder
	for index, item := range news {
		if item.Category != "" {
			fmt.Fprintf(&builder, "%d. [%s] %s\n", index+1, item.Category, item.Title)
			continue
		}

		fmt.Fprintf(&builder, "%d. %s\n", index+1, item.Title)
	}

	return strings.TrimRight(builder.String(), "\n")
}

// formatChatPlatforms mengelompokkan platform per kategori supaya model tahu
// mana yang untuk kripto, saham, dan komoditas tanpa membaca satu per satu.
// Deskripsi hanya ikut kalau frontend mengirimkannya — yaitu untuk platform yang
// benar-benar disebut di pertanyaan — sehingga daftar 36 platform tetap ringkas.
func formatChatPlatforms(platforms []ChatPlatform) string {
	if len(platforms) == 0 {
		return "(daftar platform tidak tersedia)"
	}

	labels := map[string]string{
		"crypto":    "Kripto",
		"saham":     "Saham",
		"komoditas": "Komoditas",
	}

	var builder strings.Builder
	for _, platform := range platforms {
		category := labels[strings.ToLower(platform.Category)]
		if category == "" {
			category = platform.Category
		}

		meta := make([]string, 0, 2)
		if category != "" {
			meta = append(meta, category)
		}
		if platform.Type != "" {
			meta = append(meta, platform.Type)
		}

		line := "- " + platform.Name
		if len(meta) > 0 {
			line += " (" + strings.Join(meta, ", ") + ")"
		}
		if platform.Description != "" {
			line += ": " + platform.Description
		}

		builder.WriteString(line + "\n")
	}

	return strings.TrimRight(builder.String(), "\n")
}

// formatChatNumber memilih jumlah desimal mengikuti besar nilainya, sama seperti
// usdPrice di frontend: rupiah dan dolar penuh tidak perlu dua desimal.
func formatChatNumber(value float64) string {
	switch {
	case value >= 1000:
		return strconv.FormatFloat(value, 'f', 0, 64)
	case value >= 1:
		return strconv.FormatFloat(value, 'f', 2, 64)
	default:
		return strconv.FormatFloat(value, 'f', 6, 64)
	}
}

func formatChatChange(value float64) string {
	if value == 0 {
		return "0.00%"
	}

	if value > 0 {
		return "+" + strconv.FormatFloat(value, 'f', 2, 64) + "%"
	}

	return strconv.FormatFloat(value, 'f', 2, 64) + "%"
}
