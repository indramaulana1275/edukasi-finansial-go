package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261018022327CreateChatMessagesTable struct{}

// Signature The unique signature for the migration.
func (r *M20261018022327CreateChatMessagesTable) Signature() string {
	return "20261018022327_create_chat_messages_table"
}

// Up Run the migrations.
func (r *M20261018022327CreateChatMessagesTable) Up() error {
	if !facades.Schema().HasTable("chat_messages") {
		return facades.Schema().Create("chat_messages", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id")
			table.String("role")
			table.Text("message")
			table.Timestamps()
			table.Foreign("user_id").References("id").On("users").CascadeOnDelete()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20261018022327CreateChatMessagesTable) Down() error {
	return facades.Schema().DropIfExists("chat_messages")
}
