package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261020000003AddUserProfileFields struct{}

// Signature The unique signature for the migration.
func (m *M20261020000003AddUserProfileFields) Signature() string {
	return "20261020000003_add_user_profile_fields"
}

// Up melengkapi tabel users dengan data profil yang bisa diisi sendiri oleh
// user. Semua kolom punya nilai default kosong supaya akun lama tetap valid
// tanpa backfill.
func (m *M20261020000003AddUserProfileFields) Up() error {
	if !facades.Schema().HasTable("users") {
		return nil
	}

	if !facades.Schema().HasColumn("users", "gender") {
		if err := facades.Schema().Table("users", func(table schema.Blueprint) {
			table.String("gender", 20).Default("")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasColumn("users", "avatar_url") {
		if err := facades.Schema().Table("users", func(table schema.Blueprint) {
			table.String("avatar_url", 500).Default("")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasColumn("users", "bio") {
		if err := facades.Schema().Table("users", func(table schema.Blueprint) {
			table.String("bio", 100).Default("")
		}); err != nil {
			return err
		}
	}

	return nil
}

func (m *M20261020000003AddUserProfileFields) Down() error {
	if !facades.Schema().HasTable("users") {
		return nil
	}

	columns := make([]string, 0, 3)
	for _, column := range []string{"gender", "avatar_url", "bio"} {
		if facades.Schema().HasColumn("users", column) {
			columns = append(columns, column)
		}
	}
	if len(columns) == 0 {
		return nil
	}

	return facades.Schema().Table("users", func(table schema.Blueprint) {
		table.DropColumn(columns...)
	})
}
