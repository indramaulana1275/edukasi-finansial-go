package bootstrap

import (
	"github.com/goravel/framework/contracts/database/seeder"

	"edukasi-finansial-goedukasi-finansial-go/database/seeders"
)

func Seeders() []seeder.Seeder {
	return []seeder.Seeder{
		&seeders.AdminUserSeeder{},
		&seeders.RegularUserSeeder{},
		&seeders.EducationModuleSeeder{},
		// Kuis menempel ke modul lewat judul, jadi modulnya harus diisi lebih dulu.
		&seeders.EducationQuizSeeder{},
		&seeders.InvestmentPlatformSeeder{},
	}
}
