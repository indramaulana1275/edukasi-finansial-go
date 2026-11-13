package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type M20261018235900CreateEducationQuizzesTable struct{}

// Signature The unique signature for the migration.
func (r *M20261018235900CreateEducationQuizzesTable) Signature() string {
	return "20261018235900_create_education_quizzes_table"
}

// Up Run the migrations.
func (r *M20261018235900CreateEducationQuizzesTable) Up() error {
	if !facades.Schema().HasTable("education_quizzes") {
		return facades.Schema().Create("education_quizzes", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("module_id")
			table.Text("question")
			table.Text("option_a")
			table.Text("option_b")
			table.Text("option_c")
			// Huruf pilihan yang benar. Opsi D dan penjelasan ditambahkan
			// lewat migrasi konten admin, jadi kuis lama tetap terbaca.
			table.String("correct_answer")
			table.Timestamps()
			table.Foreign("module_id").References("id").On("education_modules").CascadeOnDelete()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20261018235900CreateEducationQuizzesTable) Down() error {
	return facades.Schema().DropIfExists("education_quizzes")
}
