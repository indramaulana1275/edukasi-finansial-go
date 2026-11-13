package controllers

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// quizQuestionLimit adalah jumlah soal yang ditampilkan setiap kali user
// membuka kuis. Bank soal per modul tetap bisa berisi sepuluh soal, tetapi
// percobaan pertama hanya mengambil tiga soal acak.
const quizQuestionLimit = 3

// educationPassScore menghitung jumlah jawaban benar minimal supaya sebuah
// modul dinyatakan lulus. Untuk tiga soal yang ditampilkan, hasilnya dua.
func educationPassScore(total int) int {
	if total <= 0 {
		return 0
	}

	return total/2 + 1
}

type EducationController struct{}

func NewEducationController() *EducationController {
	return &EducationController{}
}

// Index mengembalikan daftar modul beserta status selesai milik user yang login,
// plus ringkasan progres (jumlah selesai dan persentasenya). Isi materi (slides)
// tidak ikut di sini supaya daftarnya tetap ringan; detailnya diambil lewat Show.
func (r *EducationController) Index(ctx http.Context) http.Response {
	modules := make([]models.EducationModule, 0)
	if err := facades.Orm().Query().Order("id asc").Get(&modules); err != nil {
		facades.Log().Errorf("gagal membaca modul edukasi: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal membaca modul edukasi",
		})
	}

	done := completedModuleIDs(currentUserID(ctx))

	completed := 0
	data := make([]http.Json, 0, len(modules))
	for _, module := range modules {
		if done[module.ID] {
			completed++
		}

		data = append(data, http.Json{
			"id":          module.ID,
			"title":       module.Title,
			"description": module.Description,
			"icon":        module.Icon,
			"slide_count": len(models.SlidesFromJSON(module.Content)),
			"completed":   done[module.ID],
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":     data,
		"progress": educationProgress(completed, len(modules)),
	})
}

// Show mengembalikan satu modul lengkap dengan slide dan status selesainya.
func (r *EducationController) Show(ctx http.Context) http.Response {
	module, found := educationModule(ctx.Request().RouteInt("id"))
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Modul tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data": educationModulePayload(module, completedModuleIDs(currentUserID(ctx))[module.ID]),
	})
}

// Complete menandai modul selesai untuk user yang sedang login. Dipakai tombol
// "Tandai Selesai" di frontend; aman dipanggil berkali-kali.
func (r *EducationController) Complete(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")

	module, found := educationModule(id)
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Modul tidak ditemukan"})
	}

	userID := currentUserID(ctx)
	if err := markModuleDone(userID, module.ID); err != nil {
		facades.Log().Errorf("gagal menyimpan progres modul %d user %d: %v", module.ID, userID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan progres modul"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Modul ditandai selesai",
		"data": http.Json{
			"module_id": module.ID,
			"status":    models.ModuleStatusDone,
		},
	})
}

// Quiz mengirim tiga soal acak dari bank soal sebuah modul. Urutan soal dan
// urutan pilihannya diacak setiap permintaan, tetapi setiap pilihan tetap punya
// id asli (a/b/c/d) yang dipakai server saat menilai. Jawaban benar tidak ikut
// dikirim sebelum user mengirim jawabannya.
func (r *EducationController) Quiz(ctx http.Context) http.Response {
	module, found := educationModule(ctx.Request().RouteInt("id"))
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Modul tidak ditemukan"})
	}

	questions := randomQuizQuestions(educationQuizQuestions(module.ID), quizQuestionLimit)

	items := make([]http.Json, 0, len(questions))
	for _, question := range questions {
		items = append(items, http.Json{
			"id":       question.ID,
			"question": question.Question,
			"options":  shuffledQuizOptions(question),
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data": http.Json{
			"module_id":  module.ID,
			"title":      module.Title,
			"pass_score": educationPassScore(len(items)),
			"questions":  items,
		},
	})
}

// QuizSubmit menilai jawaban kuis. Body-nya {"answers": {"<id soal>": "<id pilihan>"}}.
// Hanya soal yang benar-benar dikirim yang dinilai, jadi tiga soal acak dari bank
// soal tidak dihukum karena tujuh soal lain tidak muncul di layar.
func (r *EducationController) QuizSubmit(ctx http.Context) http.Response {
	module, found := educationModule(ctx.Request().RouteInt("id"))
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Modul tidak ditemukan"})
	}

	var input struct {
		Answers map[string]string `json:"answers"`
	}
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format jawaban tidak dikenali"})
	}

	if len(input.Answers) == 0 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Jawaban kuis belum diisi"})
	}

	bank := educationQuizQuestions(module.ID)
	if len(bank) == 0 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Modul ini belum memiliki kuis"})
	}

	expected := quizQuestionLimit
	if len(bank) < expected {
		expected = len(bank)
	}
	if len(input.Answers) != expected {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": fmt.Sprintf("Jawab semua %d soal yang tampil.", expected),
		})
	}

	byID := make(map[uint]models.EducationQuiz, len(bank))
	for _, question := range bank {
		byID[question.ID] = question
	}

	served := make([]models.EducationQuiz, 0, expected)
	for rawID := range input.Answers {
		var id uint
		if _, err := fmt.Sscanf(strings.TrimSpace(rawID), "%d", &id); err != nil {
			return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Ada jawaban untuk soal yang tidak dikenali"})
		}
		question, ok := byID[id]
		if !ok {
			return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Ada jawaban untuk soal yang tidak dikenali"})
		}
		served = append(served, question)
	}

	// Urutan review dibuat stabil supaya tidak berubah-ubah karena urutan map.
	sort.Slice(served, func(i, j int) bool { return served[i].ID < served[j].ID })

	score := 0
	review := make([]http.Json, 0, len(served))
	for _, question := range served {
		answer := normaliseQuizOption(input.Answers[fmt.Sprintf("%d", question.ID)])
		correctAnswer := normaliseQuizOption(question.CorrectAnswer)
		correct := answer != "" && answer == correctAnswer
		if correct {
			score++
		}

		review = append(review, http.Json{
			"question_id":    question.ID,
			"answer":         answer,
			"correct_answer": correctAnswer,
			"correct":        correct,
			"explanation":    question.Explanation,
		})
	}

	passScore := educationPassScore(len(served))
	passed := score >= passScore
	userID := currentUserID(ctx)

	result := models.QuizResult{
		UserID:   userID,
		ModuleID: module.ID,
		Score:    score,
		Total:    len(served),
		Passed:   passed,
	}
	if err := facades.Orm().Query().Create(&result); err != nil {
		facades.Log().Errorf("gagal menyimpan hasil kuis modul %d user %d: %v", module.ID, userID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan hasil kuis"})
	}

	if passed {
		if err := markModuleDone(userID, module.ID); err != nil {
			facades.Log().Errorf("gagal menyimpan progres modul %d user %d: %v", module.ID, userID, err)

			return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan progres modul"})
		}
	}

	message := fmt.Sprintf("Skor kamu %d dari %d. Minimal %d jawaban benar untuk lulus.", score, len(served), passScore)
	if passed {
		message = "Selamat, kamu lulus modul ini!"
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data": http.Json{
			"result_id":  result.ID,
			"module_id":  module.ID,
			"score":      score,
			"total":      len(served),
			"pass_score": passScore,
			"passed":     passed,
			"completed":  passed || completedModuleIDs(userID)[module.ID],
			"message":    message,
			"review":     review,
		},
	})
}

// educationModule mengambil satu modul. Boolean kedua membedakan "tidak ada"
// dari kegagalan baca tanpa bergantung pada orm.Model yang terisi.
func educationModule(id int) (models.EducationModule, bool) {
	if id <= 0 {
		return models.EducationModule{}, false
	}

	var module models.EducationModule
	if err := facades.Orm().Query().Where("id = ?", id).First(&module); err != nil || module.ID == 0 {
		return models.EducationModule{}, false
	}

	return module, true
}

// educationQuizQuestions membaca soal satu modul dengan urutan tetap.
func educationQuizQuestions(moduleID uint) []models.EducationQuiz {
	questions := make([]models.EducationQuiz, 0)
	if err := facades.Orm().Query().
		Where("module_id = ?", moduleID).
		Order("id asc").
		Get(&questions); err != nil {
		facades.Log().Errorf("gagal membaca kuis modul %d: %v", moduleID, err)

		return []models.EducationQuiz{}
	}

	return questions
}

// completedModuleIDs mengembalikan himpunan id modul yang sudah selesai, dibaca
// sekali untuk seluruh daftar supaya tidak ada satu query per baris.
func completedModuleIDs(userID uint) map[uint]bool {
	rows := make([]models.UserModuleProgress, 0)
	if err := facades.Orm().Query().
		Where("user_id = ?", userID).
		Where("status = ?", models.ModuleStatusDone).
		Get(&rows); err != nil {
		return map[uint]bool{}
	}

	done := make(map[uint]bool, len(rows))
	for _, row := range rows {
		done[row.ModuleID] = true
	}

	return done
}

// markModuleDone menyimpan status "selesai" untuk satu modul. Baris progresnya
// dibuat sekali lalu dipakai ulang, jadi menandai dua kali tidak menggandakan data.
func markModuleDone(userID, moduleID uint) error {
	var progress models.UserModuleProgress
	err := facades.Orm().Query().
		Where("user_id = ?", userID).
		Where("module_id = ?", moduleID).
		First(&progress)

	switch {
	case err != nil || progress.ID == 0:
		progress = models.UserModuleProgress{UserID: userID, ModuleID: moduleID, Status: models.ModuleStatusDone}

		return facades.Orm().Query().Create(&progress)
	case progress.Status != models.ModuleStatusDone:
		progress.Status = models.ModuleStatusDone

		return facades.Orm().Query().Save(&progress)
	}

	return nil
}

// educationProgress menghitung persentase modul selesai. Pembulatan ke bawah
// dipakai supaya angkanya konsisten dengan jumlah modul yang benar-benar selesai.
func educationProgress(completed, total int) http.Json {
	percent := 0
	if total > 0 {
		percent = completed * 100 / total
	}

	return http.Json{
		"completed": completed,
		"total":     total,
		"percent":   percent,
	}
}

func educationModulePayload(module models.EducationModule, completed bool) http.Json {
	return http.Json{
		"id":          module.ID,
		"title":       module.Title,
		"description": module.Description,
		"icon":        module.Icon,
		"slides":      models.SlidesFromJSON(module.Content),
		"completed":   completed,
	}
}

// randomQuizQuestions mengambil sebagian soal dari bank soal tanpa mengubah
// irisan aslinya. Setiap permintaan bisa menampilkan kombinasi yang berbeda.
func randomQuizQuestions(bank []models.EducationQuiz, limit int) []models.EducationQuiz {
	if limit <= 0 || len(bank) == 0 {
		return []models.EducationQuiz{}
	}

	picked := make([]models.EducationQuiz, len(bank))
	copy(picked, bank)
	rand.Shuffle(len(picked), func(i, j int) {
		picked[i], picked[j] = picked[j], picked[i]
	})

	if len(picked) > limit {
		picked = picked[:limit]
	}

	return picked
}

// shuffledQuizOptions mengacak urutan tampilan pilihan, tetapi id tiap pilihan
// tetap huruf aslinya. Dengan begitu penilaian mencocokkan id pilihan yang benar,
// bukan posisi A/B/C/D di layar.
func shuffledQuizOptions(question models.EducationQuiz) []http.Json {
	options := []struct {
		id   string
		text string
	}{
		{id: "a", text: question.OptionA},
		{id: "b", text: question.OptionB},
		{id: "c", text: question.OptionC},
		{id: "d", text: question.OptionD},
	}

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	items := make([]http.Json, 0, len(options))
	for _, option := range options {
		if strings.TrimSpace(option.text) == "" {
			continue
		}
		items = append(items, http.Json{
			"id":   option.id,
			"text": option.text,
		})
	}

	return items
}

// normaliseQuizOption menyamakan bentuk jawaban supaya "A", " a ", dan "opt_a"
// sama-sama menunjuk pilihan yang sama.
func normaliseQuizOption(answer string) string {
	value := strings.ToLower(strings.TrimSpace(answer))
	value = strings.TrimPrefix(value, "opt_")
	switch value {
	case "a", "b", "c", "d":
		return value
	default:
		return ""
	}
}
