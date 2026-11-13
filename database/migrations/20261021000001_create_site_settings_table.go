package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261021000001CreateSiteSettingsTable struct{}

// Signature The unique signature for the migration.
func (m *M20261021000001CreateSiteSettingsTable) Signature() string {
	return "20261021000001_create_site_settings_table"
}

// Up creates the single-row site settings table used by Panel Developer.
func (m *M20261021000001CreateSiteSettingsTable) Up() error {
	if !facades.Schema().HasTable("site_settings") {
		return facades.Schema().Create("site_settings", func(table schema.Blueprint) {
			table.ID()
			table.Text("welcome_text").Nullable()
			table.String("contact_email", 160).Default("")
			table.String("contact_phone", 60).Default("")
			table.Text("contact_address").Nullable()
			table.Timestamps()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (m *M20261021000001CreateSiteSettingsTable) Down() error {
	return facades.Schema().DropIfExists("site_settings")
}
