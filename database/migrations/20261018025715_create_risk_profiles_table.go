package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261018025715CreateRiskProfilesTable struct{}

// Signature The unique signature for the migration.
func (r *M20261018025715CreateRiskProfilesTable) Signature() string {
	return "20261018025715_create_risk_profiles_table"
}

// Up Run the migrations.
func (r *M20261018025715CreateRiskProfilesTable) Up() error {
	if !facades.Schema().HasTable("risk_profiles") {
		return facades.Schema().Create("risk_profiles", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id")
			table.Integer("skor")
			table.String("tipe_risiko")
			table.Timestamps()
			table.Foreign("user_id").References("id").On("users").CascadeOnDelete()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20261018025715CreateRiskProfilesTable) Down() error {
	return facades.Schema().DropIfExists("risk_profiles")
}
