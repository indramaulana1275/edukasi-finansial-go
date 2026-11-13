package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261021000002CreateActivityLogsTable struct{}

// Signature The unique signature for the migration.
func (m *M20261021000002CreateActivityLogsTable) Signature() string {
	return "20261021000002_create_activity_logs_table"
}

// Up creates the login activity table shown in Panel Developer. No foreign key
// is used so a deleted account leaves its history readable.
func (m *M20261021000002CreateActivityLogsTable) Up() error {
	if !facades.Schema().HasTable("activity_logs") {
		return facades.Schema().Create("activity_logs", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id").Default(0)
			table.String("name", 120).Default("")
			table.String("email", 160).Default("")
			table.String("role", 20).Default("")
			table.String("action", 40).Default("")
			table.String("ip", 64).Default("")
			table.Text("user_agent").Nullable()
			table.Timestamps()
			table.Index("user_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (m *M20261021000002CreateActivityLogsTable) Down() error {
	return facades.Schema().DropIfExists("activity_logs")
}
