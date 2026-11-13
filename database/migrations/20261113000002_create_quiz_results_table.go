package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261113000002CreateQuizResultsTable struct{}

// Signature The unique signature for the migration.
func (m *M20261113000002CreateQuizResultsTable) Signature() string {
	return "20261113000002_create_quiz_results_table"
}

// Up stores every education quiz attempt instead of only the latest progress.
func (m *M20261113000002CreateQuizResultsTable) Up() error {
	if !facades.Schema().HasTable("quiz_results") {
		return facades.Schema().Create("quiz_results", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id")
			table.UnsignedBigInteger("module_id")
			table.Integer("score")
			table.Integer("total")
			table.Boolean("passed").Default(false)
			table.Timestamps()
			table.Index("user_id")
			table.Index("module_id")
			table.Foreign("user_id").References("id").On("users").CascadeOnDelete()
			table.Foreign("module_id").References("id").On("education_modules").CascadeOnDelete()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (m *M20261113000002CreateQuizResultsTable) Down() error {
	return facades.Schema().DropIfExists("quiz_results")
}
