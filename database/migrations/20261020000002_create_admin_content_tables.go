package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261020000002CreateAdminContentTables struct{}

// Signature The unique signature for the migration.
func (m *M20261020000002CreateAdminContentTables) Signature() string {
	return "20261020000002_create_admin_content_tables"
}

// Up menambahkan kolom yang dibutuhkan panel admin ke tabel edukasi yang sudah
// ada, lalu membuat tabel investment_platforms. Tabel modul dan kuis sengaja
// tidak diduplikasi: education_modules dan education_quizzes sudah menampung
// data yang sama, dan user_module_progress menunjuk ke id modul lama.
func (m *M20261020000002CreateAdminContentTables) Up() error {
	if facades.Schema().HasTable("education_modules") && !facades.Schema().HasColumn("education_modules", "icon") {
		if err := facades.Schema().Table("education_modules", func(table schema.Blueprint) {
			table.String("icon", 20).Default("")
		}); err != nil {
			return err
		}
	}

	if facades.Schema().HasTable("education_quizzes") {
		if !facades.Schema().HasColumn("education_quizzes", "option_d") {
			if err := facades.Schema().Table("education_quizzes", func(table schema.Blueprint) {
				table.String("option_d", 500).Default("")
			}); err != nil {
				return err
			}
		}

		if !facades.Schema().HasColumn("education_quizzes", "explanation") {
			if err := facades.Schema().Table("education_quizzes", func(table schema.Blueprint) {
				table.String("explanation", 1000).Default("")
			}); err != nil {
				return err
			}
		}
	}

	if facades.Schema().HasTable("investment_platforms") {
		return nil
	}

	return facades.Schema().Create("investment_platforms", func(table schema.Blueprint) {
		table.ID()
		table.String("name", 120)
		table.String("category", 30)
		table.String("type", 30)
		table.String("logo_domain", 120)
		table.Text("description")
		// Pros dan cons adalah JSON array; bentuknya divalidasi di controller.
		table.Text("pros")
		table.Text("cons")
		table.Timestamps()
		table.Unique("name")
	})
}

// Down membalik perubahan. Kolom yang ditambahkan ke tabel lama ikut dilepas
// supaya migrasi bisa diulang dari database kosong.
func (m *M20261020000002CreateAdminContentTables) Down() error {
	if facades.Schema().HasTable("education_modules") && facades.Schema().HasColumn("education_modules", "icon") {
		if err := facades.Schema().Table("education_modules", func(table schema.Blueprint) {
			table.DropColumn("icon")
		}); err != nil {
			return err
		}
	}

	if facades.Schema().HasTable("education_quizzes") {
		if facades.Schema().HasColumn("education_quizzes", "option_d") && facades.Schema().HasColumn("education_quizzes", "explanation") {
			if err := facades.Schema().Table("education_quizzes", func(table schema.Blueprint) {
				table.DropColumn("option_d", "explanation")
			}); err != nil {
				return err
			}
		}
	}

	return facades.Schema().DropIfExists("investment_platforms")
}
