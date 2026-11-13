package config

import (
	"strings"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

func init() {
	config := facades.Config()

	// Satu API key Gemini utama. GEMINI_API_KEYS (lama/dipisah koma) tetap
	// dibaca sebagai cadangan supaya .env lama tidak langsung berhenti bekerja.
	rawKeys := envString(config.Env("GEMINI_API_KEY", ""))
	if rawKeys == "" {
		rawKeys = envString(config.Env("GEMINI_API_KEYS", ""))
	}

	apiKeys := splitGeminiAPIKeys(rawKeys)

	config.Add("gemini", map[string]any{
		// Daftar API Key Google Gemini, dicoba berurutan oleh AIService.
		//
		// Ambil dari Google AI Studio (https://aistudio.google.com/apikey)
		// lalu simpan di .env sebagai GEMINI_API_KEY.
		"api_keys": apiKeys,

		// Key pertama tetap tersedia untuk pemanggil lama yang masih membaca
		// gemini.api_key. Kode baru sebaiknya memakai gemini.api_keys.
		"api_key": firstGeminiAPIKey(apiKeys),

		// Model Gemini yang dipakai FINANIKA.
		//
		// gemini-1.5-flash dan gemini-2.5-flash sudah tidak bisa dipakai akun
		// baru, jadi default diarahkan ke model yang direkomendasikan Google.
		"model": config.Env("GEMINI_MODEL", "gemini-3.8-flash"),

		// Base URL REST API Gemini.
		"base_url": config.Env("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
	})
}

// envString membaca nilai env sebagai string. config.Env mengembalikan any,
// jadi nil atau tipe lain diperlakukan sebagai string kosong.
func envString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}

	return ""
}

// splitGeminiAPIKeys memecah daftar key dan membuang entri kosong atau duplikat.
// Urutan dipertahankan karena AIService mencoba key sesuai urutan ini.
func splitGeminiAPIKeys(raw string) []string {
	keys := make([]string, 0)
	seen := make(map[string]bool)

	for _, part := range strings.Split(raw, ",") {
		key := strings.TrimSpace(part)
		if key == "" || seen[key] {
			continue
		}

		seen[key] = true
		keys = append(keys, key)
	}

	return keys
}

func firstGeminiAPIKey(keys []string) string {
	if len(keys) == 0 {
		return ""
	}

	return keys[0]
}
