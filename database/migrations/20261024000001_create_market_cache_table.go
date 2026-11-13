package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261024000001CreateMarketCacheTable struct{}

// Signature The unique signature for the migration.
func (m *M20261024000001CreateMarketCacheTable) Signature() string {
	return "20261024000001_create_market_cache_table"
}

// Up membuat cache pasar yang bertahan setelah server restart. Isinya salinan
// JSON terakhir dari CoinGecko/Yahoo/PIHPS, dipakai sebagai cadangan saat
// upstream membalas rate limit atau sedang mati.
func (m *M20261024000001CreateMarketCacheTable) Up() error {
	if !facades.Schema().HasTable("market_cache") {
		return facades.Schema().Create("market_cache", func(table schema.Blueprint) {
			table.ID()
			table.String("cache_key", 191).Default("")
			table.LongText("payload").Nullable()
			table.BigInteger("cached_at").Default(0)
			table.Index("cache_key")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (m *M20261024000001CreateMarketCacheTable) Down() error {
	return facades.Schema().DropIfExists("market_cache")
}
