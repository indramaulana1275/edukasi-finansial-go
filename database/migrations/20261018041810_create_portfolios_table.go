package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261018041810CreatePortfoliosTable struct{}

// Signature The unique signature for the migration.
func (r *M20261018041810CreatePortfoliosTable) Signature() string {
	return "20261018041810_create_portfolios_table"
}

// Up Run the migrations.
func (r *M20261018041810CreatePortfoliosTable) Up() error {
	if !facades.Schema().HasTable("portfolios") {
		return facades.Schema().Create("portfolios", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id")
			table.String("asset_symbol")
			table.String("asset_type")
			table.Double("quantity")
			table.Double("buy_price")
			table.Timestamps()
			table.Foreign("user_id").References("id").On("users").CascadeOnDelete()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20261018041810CreatePortfoliosTable) Down() error {
	return facades.Schema().DropIfExists("portfolios")
}
