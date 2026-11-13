package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M2026_10_18_000001_CreateUsersTable struct{}

func (m *M2026_10_18_000001_CreateUsersTable) Signature() string {
	return "2026_10_18_000001_create_users_table"
}

func (m *M2026_10_18_000001_CreateUsersTable) Up() error {
	return facades.Schema().Create("users", func(table schema.Blueprint) {
		table.ID()
		table.String("name")
		table.String("email")
		table.Unique("email")
		table.String("password")
		table.String("role").Default("user")
		table.Timestamps()
	})
}

func (m *M2026_10_18_000001_CreateUsersTable) Down() error {
	return facades.Schema().DropIfExists("users")
}
