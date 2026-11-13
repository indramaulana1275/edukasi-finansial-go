package models

import (
	"encoding/json"
	"strings"

	"github.com/goravel/framework/database/orm"
)

// Status baris user_module_progress. Nilai ini yang dikirim dan diterima API.
const (
	ModuleStatusDone    = "selesai"
	ModuleStatusPending = "belum"
)

// EducationModule adalah satu modul edukasi. Content disimpan sebagai JSON
// berupa daftar slide ({"title": ..., "body": ...}), sehingga satu baris bisa
// menampung materi sepanjang apa pun tanpa tabel tambahan.
type EducationModule struct {
	orm.Model
	Title       string `gorm:"type:string;not null" json:"title"`
	Description string `gorm:"type:string;not null" json:"description"`
	// Icon berisi emoji atau teks pendek yang ditampilkan di daftar modul.
	// Kolomnya ditambahkan lewat migrasi konten admin, jadi modul lama tetap
	// terbaca dengan nilai kosong.
	Icon    string `gorm:"type:string" json:"icon"`
	Content string `gorm:"type:text;not null" json:"content"`
}

// UserModuleProgress mencatat modul mana yang sudah diselesaikan seorang user.
// Satu baris per pasangan user_id + module_id.
type UserModuleProgress struct {
	orm.Model
	UserID   uint   `gorm:"not null" json:"user_id"`
	ModuleID uint   `gorm:"not null" json:"module_id"`
	Status   string `gorm:"type:string;not null" json:"status"`
}

// TableName dipatok manual: GORM akan menjamakkan "Progress" menjadi
// "user_module_progresses", sedangkan tabelnya memang bernama
// user_module_progress.
func (r *UserModuleProgress) TableName() string {
	return "user_module_progress"
}

// EducationQuiz adalah satu soal pilihan ganda di akhir sebuah modul. Jawaban
// benarnya disimpan di sini dan hanya dinilai di server; endpoint kuis tidak
// pernah mengirim correct_answer ke frontend sebelum user menjawab.
type EducationQuiz struct {
	orm.Model
	ModuleID uint   `gorm:"not null" json:"module_id"`
	Question string `gorm:"type:text;not null" json:"question"`
	OptionA  string `gorm:"type:text;not null" json:"option_a"`
	OptionB  string `gorm:"type:text;not null" json:"option_b"`
	OptionC  string `gorm:"type:text;not null" json:"option_c"`
	OptionD  string `gorm:"type:string" json:"option_d"`
	// Explanation adalah pembahasan singkat yang ditampilkan setelah user
	// mengirim jawaban. Sebelum submit, endpoint kuis tidak mengirimnya.
	Explanation   string `gorm:"type:string" json:"explanation"`
	CorrectAnswer string `gorm:"type:string;not null" json:"correct_answer"`
}

// EducationSlide adalah satu bagian materi di dalam kolom content.
type EducationSlide struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// SlidesFromJSON mengubah kolom content menjadi daftar slide. Kalau isinya bukan
// JSON yang sah, seluruh teks dijadikan satu slide supaya modul tetap terbaca
// alih-alih hilang.
func SlidesFromJSON(raw string) []EducationSlide {
	var slides []EducationSlide
	if err := json.Unmarshal([]byte(raw), &slides); err == nil && len(slides) > 0 {
		return slides
	}

	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return []EducationSlide{{Body: trimmed}}
	}

	return []EducationSlide{}
}

// SlidesToJSON menyusun kembali kolom content dari daftar slide. Dipakai panel
// admin saat menyimpan hasil edit; seeder memakai helper yang sama.
func SlidesToJSON(slides []EducationSlide) string {
	encoded, err := json.Marshal(slides)
	if err != nil {
		return "[]"
	}

	return string(encoded)
}
