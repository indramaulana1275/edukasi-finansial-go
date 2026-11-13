package services

import (
	"strings"
	"testing"
)

// Semua prompt AI yang tampil ke user harus memakai satuan Rupiah yang sama:
// nilai di atas 1 Miliar ditulis "Miliar", bukan "Juta".
func TestIDRFormatInstructionIsInEveryPrompt(t *testing.T) {
	prompts := map[string]string{
		"summary":        systemInstruction("Moderat", "id"),
		"chat-quick":     quickChatSystemInstruction(ChatContext{Language: "id", Mode: ChatModeQuick}),
		"asset-analysis": assetAnalysisInstruction(),
		"news":           newsInstruction("id"),
		"calendar":       calendarInstruction("id"),
	}

	for name, prompt := range prompts {
		if !strings.Contains(prompt, "JANGAN gunakan format Juta untuk angka Miliar") {
			t.Fatalf("%s prompt tidak memuat aturan format IDR", name)
		}
		if !strings.Contains(prompt, "IDR 1,46 Miliar") {
			t.Fatalf("%s prompt tidak memuat contoh pembulatan Miliar", name)
		}
	}
}

// Angka persentase 24 jam tidak boleh dikarang: aturan ini harus ikut di prompt
// ringkasan Dasbor dan kedua mode chat, karena di situlah model menyebut naik/turun.
func TestPriceChangeInstructionIsInSummaryAndChatPrompts(t *testing.T) {
	prompts := map[string]string{
		"summary":    systemInstruction("Moderat", "id"),
		"chat":       chatSystemInstruction(ChatContext{Profile: "Moderat", Language: "id", Mode: ChatModeMedium}),
		"chat-quick": quickChatSystemInstruction(ChatContext{Language: "id", Mode: ChatModeQuick}),
	}

	for name, prompt := range prompts {
		if !strings.Contains(prompt, "DILARANG MENGARANG ANGKA") {
			t.Fatalf("%s prompt tidak memuat larangan mengarang angka", name)
		}
		if !strings.Contains(prompt, `"change_24h"`) {
			t.Fatalf("%s prompt tidak merujuk field change_24h", name)
		}
		if !strings.Contains(prompt, "BUKAN untuk mengganti angka persentase harga") {
			t.Fatalf("%s prompt tidak membatasi peran berita", name)
		}
	}
}

func TestFormatMarketChangeLinesUsesExactValue(t *testing.T) {
	market := map[string]interface{}{
		"ihsg": map[string]interface{}{"price": 5143.245, "change_percent": -0.11},
		"gold": map[string]interface{}{"price_usd_oz": 4020.8, "change_percent": 0.61},
		"usd":  map[string]interface{}{"idr_rate": 18115.0, "change_percent": 0.89},
	}

	got := formatMarketChangeLines(market)
	for _, want := range []string{
		"- IHSG: change_24h = -0.11%",
		"- Emas (XAU): change_24h = +0.61%",
		"- USD/IDR: change_24h = +0.89%",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("baris perubahan tidak memuat %q: %q", want, got)
		}
	}
}

func TestUserPromptCarriesChange24h(t *testing.T) {
	market := map[string]interface{}{
		"ihsg": map[string]interface{}{"price": 5143.245, "change_percent": -0.11},
	}

	prompt := userPrompt(market, nil, "Berikan ringkasan pasar hari ini", "id")
	if !strings.Contains(prompt, "change_24h = -0.11%") {
		t.Fatalf("user prompt tidak membawa angka change_24h: %q", prompt)
	}
}

func TestFormatChatPricesLabelsChange24h(t *testing.T) {
	got := formatChatPrices([]ChatPrice{{Asset: "BTC", Price: 80285, Unit: "USD", Change: -0.11}})

	want := "- BTC: 80285 USD, change_24h: -0.11%"
	if got != want {
		t.Fatalf("baris harga = %q, ingin %q", got, want)
	}
}
