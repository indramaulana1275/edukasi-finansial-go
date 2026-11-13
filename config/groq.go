package config

import (
	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("groq", map[string]any{
		// API Key Groq. Provider cadangan untuk FINANIKA: dipakai hanya ketika
		// Gemini menolak permintaan (429) atau sedang sibuk (503), supaya chat
		// tidak berhenti di pesan "sistem sibuk".
		//
		// Ambil dari https://console.groq.com/keys lalu simpan di .env sebagai
		// GROQ_API_KEY.
		//
		// Dibaca lewat config.Env, bukan os.Getenv: Goravel memuat .env ke viper
		// tanpa memanggil os.Setenv, jadi os.Getenv selalu kosong untuk nilai
		// yang ditulis di .env (hanya variabel environment OS sungguhan yang
		// terbaca). Lihat "Traps" di AGENTS.md.
		"api_key": config.Env("GROQ_API_KEY", ""),

		// Model Groq cadangan. Nilai defaultnya di app/services/ai_service.go
		// (groqDefaultModel) harus sama dengan yang di .env.
		//
		// Groq rutin menghentikan model lama: llama-3.3-70b-versatile sudah
		// tidak ada lagi, jadi defaultnya memakai model yang masih dilayani.
		"model": config.Env("GROQ_MODEL", "openai/gpt-oss-120b"),

		// Base URL Groq (REST-nya mengikuti format OpenAI).
		"base_url": config.Env("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
	})
}
