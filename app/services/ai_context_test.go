package services

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatNewsDigestKeepsFiveNewest(t *testing.T) {
	items := make([]InsightItem, 0, 8)
	for i := 1; i <= 8; i++ {
		items = append(items, InsightItem{
			Title:    fmt.Sprintf("Berita %d", i),
			Category: "Moneter",
		})
	}

	digest := formatNewsDigest(items)

	for i := 1; i <= aiNewsDigestLimit; i++ {
		want := fmt.Sprintf("%d. [Moneter] Berita %d", i, i)
		if !strings.Contains(digest, want) {
			t.Fatalf("digest tidak memuat berita %d: %q", i, digest)
		}
	}

	if strings.Contains(digest, "Berita 6") {
		t.Fatalf("digest mengirim berita melebihi batas %d: %q", aiNewsDigestLimit, digest)
	}
}

func TestFormatChatPlatformsIncludesDescription(t *testing.T) {
	got := formatChatPlatforms([]ChatPlatform{{
		Name:        "Indodax",
		Category:    "crypto",
		Type:        "Lokal",
		Description: "Bursa kripto lokal tertua di Indonesia.",
	}})

	want := "- Indodax (Kripto, Lokal): Bursa kripto lokal tertua di Indonesia."
	if got != want {
		t.Fatalf("baris platform = %q, ingin %q", got, want)
	}
}

func TestChatPricesFromAssetsCoversEveryListing(t *testing.T) {
	crypto := []map[string]any{{
		"symbol": "BTC", "name": "Bitcoin", "price_usd": 81000.0, "change_24h": 1.2,
	}}
	forex := []map[string]any{{
		"pair": "USD/IDR", "name": "Dolar Amerika Serikat", "rate": 16300.0, "change_percent": 0.1,
	}}
	saham := []map[string]any{{
		"symbol": "BBCA.JK", "name": "Bank Central Asia (BBCA)",
		"price": 6200.0, "change_percent": -0.5, "currency": "IDR",
	}}
	komoditas := []map[string]any{{
		"symbol": "CHILI", "name": "Cabai Merah", "price": 45000.0,
		"change_percent": 2.0, "currency": "IDR", "unit": "per kg",
	}}

	prices := ChatPricesFromAssets(crypto, forex, saham, komoditas)
	if len(prices) != 4 {
		t.Fatalf("jumlah harga = %d, ingin 4 (satu per daftar)", len(prices))
	}

	byKey := make(map[string]ChatPrice, len(prices))
	for _, price := range prices {
		byKey[chatPriceKey(price.Asset)] = price
	}

	if got := byKey["btc"]; got.Price != 81000 || got.Unit != "USD" {
		t.Fatalf("baris crypto = %+v, ingin 81000 USD", got)
	}
	if got := byKey["usd/idr"]; got.Price != 16300 || got.Unit != "IDR" {
		t.Fatalf("baris kurs = %+v, ingin 16300 IDR", got)
	}
	if got := byKey["bbca.jk"]; got.Price != 6200 || got.Unit != "IDR" {
		t.Fatalf("baris saham = %+v, ingin 6200 IDR", got)
	}
	if got := byKey["cabai"]; got.Price != 45000 || got.Unit != "IDR per kg" {
		t.Fatalf("baris komoditas = %+v, ingin 45000 IDR per kg", got)
	}
}

func TestPrepareChatContextKeepsEveryServerAsset(t *testing.T) {
	market := map[string]interface{}{
		"bitcoin": map[string]interface{}{"price_usd": 81000.0, "change_24h": 1.0},
	}
	assets := []ChatPrice{
		{Asset: "BTC (Bitcoin)", Price: 1.0, Unit: "USD"},
		{Asset: "BBCA.JK (Bank Central Asia (BBCA))", Price: 6200.0, Unit: "IDR"},
		{Asset: "Cabai Merah (CHILI)", Price: 45000.0, Unit: "IDR per kg"},
	}

	chat := PrepareChatContext(nil, "Moderat", market, assets, nil)
	if len(chat.Prices) != 3 {
		t.Fatalf("jumlah harga konteks = %d, ingin 3", len(chat.Prices))
	}

	for _, price := range chat.Prices {
		switch chatPriceKey(price.Asset) {
		case "btc":
			if price.Price != 81000 {
				t.Fatalf("harga BTC = %v, ingin 81000 dari kartu Dasbor", price.Price)
			}
		case "bbca.jk", "cabai":
		default:
			t.Fatalf("aset tak terduga masuk konteks: %+v", price)
		}
	}
}

func TestCleanChatPlatformsCapsDescription(t *testing.T) {
	got := cleanChatPlatforms([]ChatPlatform{{
		Name:        "Indodax",
		Description: strings.Repeat("a", chatContextPlatformDescLimit+50),
	}})

	if len(got) != 1 {
		t.Fatalf("jumlah platform = %d, ingin 1", len(got))
	}
	if runes := []rune(got[0].Description); len(runes) != chatContextPlatformDescLimit {
		t.Fatalf("panjang deskripsi = %d rune, ingin %d", len(runes), chatContextPlatformDescLimit)
	}
}
