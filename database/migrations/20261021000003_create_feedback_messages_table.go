package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261021000003CreateFeedbackMessagesTable struct{}

// Signature The unique signature for the migration.
func (m *M20261021000003CreateFeedbackMessagesTable) Signature() string {
	return "20261021000003_create_feedback_messages_table"
}

// Up creates the feedback inbox read by the admin panel.
func (m *M20261021000003CreateFeedbackMessagesTable) Up() error {
	if !facades.Schema().HasTable("feedback_messages") {
		return facades.Schema().Create("feedback_messages", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id").Default(0)
			table.String("name", 120).Default("")
			table.String("email", 160).Default("")
			table.String("subject", 160).Default("")
			table.Text("message")
			table.String("status", 20).Default("baru")
			table.Timestamps()
			table.Index("user_id")
			table.Index("status")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (m *M20261021000003CreateFeedbackMessagesTable) Down() error {
	return facades.Schema().DropIfExists("feedback_messages")
}
