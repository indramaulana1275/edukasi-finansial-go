package config

import (
	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("insight", map[string]any{
		// Sumber berita halaman Insight Global.
		//
		// Kalau true, daftarnya diambil dari feed RSS Google News (gratis, tanpa
		// API key) dan daftar contoh di app/services/insight_service.go hanya
		// dipakai saat feed gagal. Set INSIGHT_FEED=false untuk demo tanpa
		// internet supaya berita contoh yang selalu tampil.
		"feed_enabled": config.Env("INSIGHT_FEED", true),
	})
}
