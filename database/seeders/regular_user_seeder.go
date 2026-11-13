package seeders

import (
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// Akun uji untuk login sebagai pengguna biasa, bukan admin. Dipakai untuk
// memastikan alur non-admin (profil, portofolio, kuis risiko) tetap jalan.
const (
	regularUserName     = "User Finanika"
	regularUserEmail    = "user@finanika.com"
	regularUserPassword = "user123"
)

type RegularUserSeeder struct{}

// Signature The name and signature of the seeder.
func (s *RegularUserSeeder) Signature() string {
	return "RegularUserSeeder"
}

// Run membuat akun user biasa. Seperti AdminUserSeeder, password direset ke
// nilai bootstrap setiap kali seeder dijalankan supaya kredensial uji di
// dokumentasi selalu valid.
func (s *RegularUserSeeder) Run() error {
	return upsertSeedUser(regularUserName, regularUserEmail, regularUserPassword, models.UserRoleUser)
}
