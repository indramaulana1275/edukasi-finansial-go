package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261018162714CreateEducationModulesTable struct{}

// Signature The unique signature for the migration.
func (r *M20261018162714CreateEducationModulesTable) Signature() string {
	return "20261018162714_create_education_modules_table"
}

// Up Run the migrations.
func (r *M20261018162714CreateEducationModulesTable) Up() error {
	if !facades.Schema().HasTable("education_modules") {
		return facades.Schema().Create("education_modules", func(table schema.Blueprint) {
			table.ID()
			table.String("title")
			table.String("description", 500)
			// Isi materi disimpan sebagai JSON (daftar slide), bukan HTML, supaya
			// frontend tinggal merender tanpa menyimpan markup di database.
			table.Text("content")
			table.Timestamps()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20261018162714CreateEducationModulesTable) Down() error {
	return facades.Schema().DropIfExists("education_modules")
}
