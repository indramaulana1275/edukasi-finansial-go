package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261018162714CreateUserModuleProgressTable struct{}

// Signature The unique signature for the migration.
func (r *M20261018162714CreateUserModuleProgressTable) Signature() string {
	return "20261018162714_create_user_module_progress_table"
}

// Up Run the migrations.
func (r *M20261018162714CreateUserModuleProgressTable) Up() error {
	if !facades.Schema().HasTable("user_module_progress") {
		return facades.Schema().Create("user_module_progress", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id")
			table.UnsignedBigInteger("module_id")
			table.String("status")
			table.Timestamps()
			// Satu baris per pasangan user+modul; statusnya "selesai" atau "belum".
			table.Unique("user_id", "module_id")
			table.Foreign("user_id").References("id").On("users").CascadeOnDelete()
			table.Foreign("module_id").References("id").On("education_modules").CascadeOnDelete()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20261018162714CreateUserModuleProgressTable) Down() error {
	return facades.Schema().DropIfExists("user_module_progress")
}
