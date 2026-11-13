package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261113000001CreateNlgTestResultsTable struct{}

// Signature The unique signature for the migration.
func (m *M20261113000001CreateNlgTestResultsTable) Signature() string {
	return "20261113000001_create_nlg_test_results_table"
}

// Up creates the table used by the admin NLG evaluation panel.
func (m *M20261113000001CreateNlgTestResultsTable) Up() error {
	if !facades.Schema().HasTable("nlg_test_results") {
		return facades.Schema().Create("nlg_test_results", func(table schema.Blueprint) {
			table.ID()
			table.String("profile", 40)
			table.String("topic", 60)
			table.Text("narrative")
			table.Integer("score_data")
			table.Integer("score_content")
			table.Integer("score_relevance")
			table.Integer("score_clarity")
			table.Float("average_score", 8, 2)
			table.Timestamps()
			table.Index("profile")
			table.Index("topic")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (m *M20261113000001CreateNlgTestResultsTable) Down() error {
	return facades.Schema().DropIfExists("nlg_test_results")
}
