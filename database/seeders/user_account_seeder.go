package seeders

import (
	"golang.org/x/crypto/bcrypt"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// upsertSeedUser memastikan satu akun bootstrap ada dengan kredensial yang
// diketahui. Password selalu di-hash ulang memakai bcrypt — algoritma yang sama
// dipakai AuthController.Login — sehingga akun hasil restore database yang
// hash-nya hilang atau rusak tetap bisa dipakai setelah seeder dijalankan.
func upsertSeedUser(name, email, password, role string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var existing models.User
	if err := facades.Orm().Query().Where("email = ?", email).First(&existing); err != nil {
		return err
	}

	if existing.ID == 0 {
		return facades.Orm().Query().Create(&models.User{
			Name:     name,
			Email:    email,
			Password: string(hashedPassword),
			Role:     role,
		})
	}

	_, err = facades.Orm().Query().
		Table("users").
		Where("id = ?", existing.ID).
		Update(map[string]any{
			"name":     name,
			"role":     role,
			"password": string(hashedPassword),
		})

	return err
}
