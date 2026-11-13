package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

type M20261020000001EnsureUsersRoleEnum struct{}

// Signature The unique signature for the migration.
func (m *M20261020000001EnsureUsersRoleEnum) Signature() string {
	return "20261020000001_ensure_users_role_enum"
}

// Up mengubah users.role menjadi enum('user','admin'). Kolom role sudah ada
// sejak tabel users dibuat, tetapi masih varchar; enum membuat nilai yang tidak
// dikenal ditolak oleh database, bukan hanya oleh kode aplikasi.
func (m *M20261020000001EnsureUsersRoleEnum) Up() error {
	if !facades.Schema().HasTable("users") {
		return nil
	}

	// Baris lama tanpa role (mis. hasil insert manual) tidak boleh menggagalkan
	// perubahan tipe kolom.
	if _, err := facades.Orm().Query().
		Table("users").
		Where("role IS NULL OR role = ''").
		Update("role", models.UserRoleUser); err != nil {
		return err
	}

	return facades.Schema().Table("users", func(table schema.Blueprint) {
		table.Enum("role", []any{models.UserRoleUser, models.UserRoleAdmin}).
			Default(models.UserRoleUser).
			Change()
	})
}

// Down mengembalikan role ke varchar supaya migrasi bisa dibalik.
func (m *M20261020000001EnsureUsersRoleEnum) Down() error {
	if !facades.Schema().HasTable("users") {
		return nil
	}

	return facades.Schema().Table("users", func(table schema.Blueprint) {
		table.String("role").Default(models.UserRoleUser).Change()
	})
}
