package services

import (
	"strings"
	"testing"
)

func TestNormaliseChatMode(t *testing.T) {
	cases := map[string]ChatMode{
		"":         ChatModeMedium,
		"medium":   ChatModeMedium,
		"cepat":    ChatModeQuick,
		"quick":    ChatModeQuick,
		"analisis": ChatModeAnalysis,
		"analysis": ChatModeAnalysis,
		"lainnya":  ChatModeMedium,
	}

	for input, want := range cases {
		if got := NormaliseChatMode(input); got != want {
			t.Fatalf("NormaliseChatMode(%q) = %q, ingin %q", input, got, want)
		}
	}
}

func TestQuickChatPromptOmitsRiskProfileAndAdvice(t *testing.T) {
	chat := ChatContext{Profile: "Konservatif", Language: "id", Mode: ChatModeQuick}
	prompt := chatSystemInstruction(chat)

	for _, forbidden := range []string{"Profil risiko pengguna:", "Panduan saran DCA", "Konservatif"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt Mode Cepat tidak boleh memuat %q: %q", forbidden, prompt)
		}
	}

	if !strings.Contains(prompt, "JANGAN memberikan saran investasi") {
		t.Fatalf("prompt Mode Cepat harus melarang saran investasi: %q", prompt)
	}
}

func TestMediumAndAnalysisPromptsKeepRiskProfile(t *testing.T) {
	chat := ChatContext{Profile: "Konservatif", Language: "id", Mode: ChatModeMedium}
	medium := chatSystemInstruction(chat)

	if !strings.Contains(medium, "Profil risiko pengguna: Konservatif") {
		t.Fatalf("prompt Mode Medium harus memuat profil risiko: %q", medium)
	}
	if !strings.Contains(medium, "Mode Medium") {
		t.Fatalf("prompt Mode Medium harus memuat aturan mode: %q", medium)
	}
	if !strings.Contains(medium, "1-2 paragraf") || !strings.Contains(medium, "satu saran singkat") {
		t.Fatalf("prompt Mode Medium harus meminta jawaban dan saran singkat: %q", medium)
	}
	if strings.Contains(medium, "4-6 paragraf") {
		t.Fatalf("prompt Mode Medium tidak boleh meminta analisis panjang: %q", medium)
	}

	chat.Mode = ChatModeAnalysis
	analysis := chatSystemInstruction(chat)

	if !strings.Contains(analysis, "Profil risiko pengguna: Konservatif") {
		t.Fatalf("prompt Mode Analisis harus memuat profil risiko: %q", analysis)
	}
	if !strings.Contains(analysis, "Mode Analisis") {
		t.Fatalf("prompt Mode Analisis harus memuat aturan mode: %q", analysis)
	}
	if !strings.Contains(analysis, "4-6 paragraf") || !strings.Contains(analysis, "saran DCA") {
		t.Fatalf("prompt Mode Analisis harus meminta analisis panjang dan saran DCA: %q", analysis)
	}
}

func TestMediumAndAnalysisPromptsUseLanguageClosingLabel(t *testing.T) {
	chat := ChatContext{Profile: "Moderat", Language: "en", Mode: ChatModeMedium}

	medium := chatSystemInstruction(chat)
	if !strings.Contains(medium, `"Suggestion:"`) {
		t.Fatalf("prompt Mode Medium berbahasa Inggris harus memakai Suggestion:: %q", medium)
	}
	if strings.Contains(medium, `"Saran:"`) {
		t.Fatalf("prompt Mode Medium berbahasa Inggris tidak boleh memakai Saran:: %q", medium)
	}

	chat.Mode = ChatModeAnalysis
	analysis := chatSystemInstruction(chat)
	if !strings.Contains(analysis, `"Suggestion:"`) {
		t.Fatalf("prompt Mode Analisis berbahasa Inggris harus memakai Suggestion:: %q", analysis)
	}
	if strings.Contains(analysis, `"Saran:"`) {
		t.Fatalf("prompt Mode Analisis berbahasa Inggris tidak boleh memakai Saran:: %q", analysis)
	}
}

func TestQuickChatIgnoresLongHistoryAndProfile(t *testing.T) {
	chat := ChatContext{Profile: "Agresif", Language: "id", Mode: ChatModeQuick}
	prompt := chatSystemInstruction(chat)

	for _, forbidden := range []string{"Profil risiko pengguna:", "Agresif", "Panduan saran DCA", "Penutup jawaban:"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt Mode Cepat tidak boleh memuat %q: %q", forbidden, prompt)
		}
	}
	if !strings.Contains(prompt, "Riwayat percakapan hanya konteks") {
		t.Fatalf("prompt Mode Cepat harus menetralkan jawaban panjang dari riwayat: %q", prompt)
	}
}

func TestQuickChatUserPromptDoesNotAskForClosingAdvice(t *testing.T) {
	prompt := chatUserPrompt(ChatContext{Language: "id", Mode: ChatModeQuick}, "Apa itu DCA?", "")

	if strings.Contains(prompt, `Tutup dengan baris "Saran:"`) {
		t.Fatalf("prompt user Mode Cepat tidak boleh meminta baris Saran: %q", prompt)
	}
}
