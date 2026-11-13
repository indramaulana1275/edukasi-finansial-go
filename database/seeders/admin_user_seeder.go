package seeders

import "edukasi-finansial-goedukasi-finansial-go/app/models"

// Akun developer awal. Password sengaja ditulis di sini karena ini akun
// bootstrap yang diminta untuk pengujian Panel Developer.
const (
	developerAdminName     = "Indra"
	developerAdminEmail    = "admin@finanika.com"
	developerAdminPassword = "Indra27"
)

type AdminUserSeeder struct{}

// Signature The name and signature of the seeder.
func (s *AdminUserSeeder) Signature() string {
	return "AdminUserSeeder"
}

// Run memastikan akun admin pertama selalu ada. Kalau emailnya sudah terdaftar,
// role dan password direset ke nilai bootstrap supaya kredensial yang
// didokumentasikan tetap bisa dipakai.
func (s *AdminUserSeeder) Run() error {
	return upsertSeedUser(developerAdminName, developerAdminEmail, developerAdminPassword, models.UserRoleAdmin)
}
