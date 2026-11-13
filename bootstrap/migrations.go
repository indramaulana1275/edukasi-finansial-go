package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/database/migrations"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20210101000001CreateJobsTable{},
		&migrations.M2026_10_18_000001_CreateUsersTable{},
		&migrations.M20261018022327CreateChatMessagesTable{},
		&migrations.M20261018025715CreateRiskProfilesTable{},
		&migrations.M20261018041810CreatePortfoliosTable{},
		&migrations.M20261018162714CreateEducationModulesTable{},
		&migrations.M20261018162714CreateUserModuleProgressTable{},
		&migrations.M20261018235900CreateEducationQuizzesTable{},
		&migrations.M20261020000001EnsureUsersRoleEnum{},
		&migrations.M20261020000002CreateAdminContentTables{},
		&migrations.M20261020000003AddUserProfileFields{},
		&migrations.M20261021000001CreateSiteSettingsTable{},
		&migrations.M20261021000002CreateActivityLogsTable{},
		&migrations.M20261021000003CreateFeedbackMessagesTable{},
		&migrations.M20261024000001CreateMarketCacheTable{},
		&migrations.M20261113000001CreateNlgTestResultsTable{},
		&migrations.M20261113000002CreateQuizResultsTable{},
	}
}
