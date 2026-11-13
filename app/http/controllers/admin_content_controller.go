package controllers

import (
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

const (
	adminModuleMaxSlides    = 30
	adminModuleMaxBodyRunes = 8000
	adminPlatformMinItems   = 3
	adminPlatformMaxItems   = 8
	adminPlatformMaxItem    = 300
)

// AdminContentController menangani CRUD konten yang tampil di halaman user:
// modul Belajar, soal kuis modul, dan daftar platform investasi. Semua route-nya
// berada di bawah AuthMiddleware + RoleMiddleware("admin").
type AdminContentController struct{}

func NewAdminContentController() *AdminContentController {
	return &AdminContentController{}
}

type adminModuleInput struct {
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Icon        string                  `json:"icon"`
	Slides      []models.EducationSlide `json:"slides"`
}

// Modules mengirim seluruh modul beserta slide-nya. Panel admin memakai daftar
// yang sama untuk tabel dan untuk mengisi form edit, jadi tidak ada request
// detail terpisah setiap kali tombol Edit ditekan.
func (r *AdminContentController) Modules(ctx http.Context) http.Response {
	rows := make([]models.EducationModule, 0)
	if err := facades.Orm().Query().Order("id asc").Get(&rows); err != nil {
		facades.Log().Errorf("gagal membaca modul untuk admin: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal membaca daftar modul",
		})
	}

	data := make([]http.Json, 0, len(rows))
	for _, row := range rows {
		data = append(data, adminModulePayload(row))
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":  data,
		"total": len(data),
	})
}

func (r *AdminContentController) StoreModule(ctx http.Context) http.Response {
	var input adminModuleInput
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format modul tidak dikenali"})
	}

	clean, slides, message := cleanAdminModule(input)
	if message != "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": message})
	}

	module := models.EducationModule{
		Title:       clean.Title,
		Description: clean.Description,
		Icon:        clean.Icon,
		Content:     models.SlidesToJSON(slides),
	}

	if err := facades.Orm().Query().Create(&module); err != nil {
		facades.Log().Errorf("gagal menyimpan modul baru: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan modul"})
	}

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "Modul ditambahkan",
		"data":    adminModulePayload(module),
	})
}

func (r *AdminContentController) UpdateModule(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	module, found := educationModule(id)
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Modul tidak ditemukan"})
	}

	var input adminModuleInput
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format modul tidak dikenali"})
	}

	clean, slides, message := cleanAdminModule(input)
	if message != "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": message})
	}

	module.Title = clean.Title
	module.Description = clean.Description
	module.Icon = clean.Icon
	module.Content = models.SlidesToJSON(slides)

	if err := facades.Orm().Query().Save(&module); err != nil {
		facades.Log().Errorf("gagal memperbarui modul %d: %v", module.ID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal memperbarui modul"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Modul diperbarui",
		"data":    adminModulePayload(module),
	})
}

// DeleteModule menghapus modul. Foreign key education_quizzes dan
// user_module_progress memakai CascadeOnDelete, jadi soal dan progres modul itu
// ikut hilang; panel menampilkan konfirmasi sebelum memanggil endpoint ini.
func (r *AdminContentController) DeleteModule(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "ID modul tidak valid"})
	}

	result, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.EducationModule{})
	if err != nil {
		facades.Log().Errorf("gagal menghapus modul %d: %v", id, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menghapus modul"})
	}

	if result.RowsAffected == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Modul tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Modul dihapus"})
}

type adminQuizRow struct {
	ID            uint   `json:"id"`
	ModuleID      uint   `json:"module_id"`
	ModuleTitle   string `json:"module_title"`
	Question      string `json:"question"`
	OptionA       string `json:"option_a"`
	OptionB       string `json:"option_b"`
	OptionC       string `json:"option_c"`
	OptionD       string `json:"option_d"`
	CorrectAnswer string `json:"correct_answer"`
	Explanation   string `json:"explanation"`
}

type adminQuizInput struct {
	ModuleID      uint   `json:"module_id"`
	Question      string `json:"question"`
	OptionA       string `json:"option_a"`
	OptionB       string `json:"option_b"`
	OptionC       string `json:"option_c"`
	OptionD       string `json:"option_d"`
	CorrectAnswer string `json:"correct_answer"`
	Explanation   string `json:"explanation"`
}

func (r *AdminContentController) Quizzes(ctx http.Context) http.Response {
	modules := make([]models.EducationModule, 0)
	if err := facades.Orm().Query().Order("id asc").Get(&modules); err != nil {
		facades.Log().Errorf("gagal membaca modul untuk kuis admin: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca daftar kuis"})
	}

	moduleTitles := make(map[uint]string, len(modules))
	for _, module := range modules {
		moduleTitles[module.ID] = module.Title
	}

	quizzes := make([]models.EducationQuiz, 0)
	if err := facades.Orm().Query().Order("module_id asc").Order("id asc").Get(&quizzes); err != nil {
		facades.Log().Errorf("gagal membaca soal kuis untuk admin: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca daftar kuis"})
	}

	data := make([]adminQuizRow, 0, len(quizzes))
	for _, quiz := range quizzes {
		data = append(data, adminQuizRow{
			ID:            quiz.ID,
			ModuleID:      quiz.ModuleID,
			ModuleTitle:   moduleTitles[quiz.ModuleID],
			Question:      quiz.Question,
			OptionA:       quiz.OptionA,
			OptionB:       quiz.OptionB,
			OptionC:       quiz.OptionC,
			OptionD:       quiz.OptionD,
			CorrectAnswer: quiz.CorrectAnswer,
			Explanation:   quiz.Explanation,
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":  data,
		"total": len(data),
	})
}

func (r *AdminContentController) StoreQuiz(ctx http.Context) http.Response {
	var input adminQuizInput
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format soal tidak dikenali"})
	}

	clean, message := cleanAdminQuiz(input)
	if message != "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": message})
	}

	quiz := models.EducationQuiz{
		ModuleID:      clean.ModuleID,
		Question:      clean.Question,
		OptionA:       clean.OptionA,
		OptionB:       clean.OptionB,
		OptionC:       clean.OptionC,
		OptionD:       clean.OptionD,
		CorrectAnswer: clean.CorrectAnswer,
		Explanation:   clean.Explanation,
	}

	if err := facades.Orm().Query().Create(&quiz); err != nil {
		facades.Log().Errorf("gagal menyimpan soal kuis: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan soal kuis"})
	}

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "Soal kuis ditambahkan",
		"data":    quiz,
	})
}

func (r *AdminContentController) UpdateQuiz(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	quiz, found := adminQuiz(id)
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Soal kuis tidak ditemukan"})
	}

	var input adminQuizInput
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format soal tidak dikenali"})
	}

	clean, message := cleanAdminQuiz(input)
	if message != "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": message})
	}

	quiz.ModuleID = clean.ModuleID
	quiz.Question = clean.Question
	quiz.OptionA = clean.OptionA
	quiz.OptionB = clean.OptionB
	quiz.OptionC = clean.OptionC
	quiz.OptionD = clean.OptionD
	quiz.CorrectAnswer = clean.CorrectAnswer
	quiz.Explanation = clean.Explanation

	if err := facades.Orm().Query().Save(&quiz); err != nil {
		facades.Log().Errorf("gagal memperbarui soal kuis %d: %v", quiz.ID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal memperbarui soal kuis"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Soal kuis diperbarui",
		"data":    quiz,
	})
}

func (r *AdminContentController) DeleteQuiz(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "ID soal tidak valid"})
	}

	result, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.EducationQuiz{})
	if err != nil {
		facades.Log().Errorf("gagal menghapus soal kuis %d: %v", id, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menghapus soal kuis"})
	}

	if result.RowsAffected == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Soal kuis tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Soal kuis dihapus"})
}

type adminPlatformInput struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Type        string   `json:"type"`
	LogoDomain  string   `json:"logo_domain"`
	Description string   `json:"description"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
}

func (r *AdminContentController) Platforms(ctx http.Context) http.Response {
	rows := make([]models.InvestmentPlatform, 0)
	if err := facades.Orm().Query().Order("id asc").Get(&rows); err != nil {
		facades.Log().Errorf("gagal membaca platform untuk admin: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca daftar platform"})
	}

	data := make([]http.Json, 0, len(rows))
	for _, row := range rows {
		data = append(data, platformPayload(row))
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":  data,
		"total": len(data),
	})
}

func (r *AdminContentController) StorePlatform(ctx http.Context) http.Response {
	var input adminPlatformInput
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format platform tidak dikenali"})
	}

	clean, message := cleanAdminPlatform(input)
	if message != "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": message})
	}

	taken, err := adminPlatformNameTaken(clean.Name, 0)
	if err != nil {
		facades.Log().Errorf("gagal memeriksa nama platform: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan platform"})
	}
	if taken {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Nama platform sudah dipakai"})
	}

	platform := models.InvestmentPlatform{
		Name:        clean.Name,
		Category:    clean.Category,
		Type:        clean.Type,
		LogoDomain:  clean.LogoDomain,
		Description: clean.Description,
		Pros:        models.PlatformStringsToJSON(clean.Pros),
		Cons:        models.PlatformStringsToJSON(clean.Cons),
	}

	if err := facades.Orm().Query().Create(&platform); err != nil {
		facades.Log().Errorf("gagal menyimpan platform: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan platform"})
	}

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "Platform ditambahkan",
		"data":    platformPayload(platform),
	})
}

func (r *AdminContentController) UpdatePlatform(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	platform, found := adminPlatform(id)
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Platform tidak ditemukan"})
	}

	var input adminPlatformInput
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format platform tidak dikenali"})
	}

	clean, message := cleanAdminPlatform(input)
	if message != "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": message})
	}

	taken, err := adminPlatformNameTaken(clean.Name, platform.ID)
	if err != nil {
		facades.Log().Errorf("gagal memeriksa nama platform: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal memperbarui platform"})
	}
	if taken {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Nama platform sudah dipakai"})
	}

	platform.Name = clean.Name
	platform.Category = clean.Category
	platform.Type = clean.Type
	platform.LogoDomain = clean.LogoDomain
	platform.Description = clean.Description
	platform.Pros = models.PlatformStringsToJSON(clean.Pros)
	platform.Cons = models.PlatformStringsToJSON(clean.Cons)

	if err := facades.Orm().Query().Save(&platform); err != nil {
		facades.Log().Errorf("gagal memperbarui platform %d: %v", platform.ID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal memperbarui platform"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Platform diperbarui",
		"data":    platformPayload(platform),
	})
}

func (r *AdminContentController) DeletePlatform(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "ID platform tidak valid"})
	}

	result, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.InvestmentPlatform{})
	if err != nil {
		facades.Log().Errorf("gagal menghapus platform %d: %v", id, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menghapus platform"})
	}

	if result.RowsAffected == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Platform tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Platform dihapus"})
}

func adminModulePayload(module models.EducationModule) http.Json {
	slides := models.SlidesFromJSON(module.Content)

	return http.Json{
		"id":          module.ID,
		"title":       module.Title,
		"description": module.Description,
		"icon":        module.Icon,
		"slides":      slides,
		"slide_count": len(slides),
	}
}

func cleanAdminModule(input adminModuleInput) (adminModuleInput, []models.EducationSlide, string) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Icon = strings.TrimSpace(input.Icon)

	if input.Title == "" {
		return input, nil, "Judul modul wajib diisi"
	}
	if adminRuneLen(input.Title) > 200 {
		return input, nil, "Judul modul maksimal 200 karakter"
	}
	if input.Description == "" {
		return input, nil, "Deskripsi modul wajib diisi"
	}
	if adminRuneLen(input.Description) > 500 {
		return input, nil, "Deskripsi modul maksimal 500 karakter"
	}
	if adminRuneLen(input.Icon) > 20 {
		return input, nil, "Icon modul maksimal 20 karakter"
	}
	if input.Icon == "" {
		input.Icon = "📘"
	}

	slides := make([]models.EducationSlide, 0, len(input.Slides))
	for index, slide := range input.Slides {
		title := strings.TrimSpace(slide.Title)
		body := strings.TrimSpace(slide.Body)

		// Baris kosong dari form (misalnya 10 kotak yang belum diisi semua)
		// dilewati; yang sudah diisi tetap divalidasi.
		if title == "" && body == "" {
			continue
		}
		if title == "" {
			return input, nil, "Judul slide " + adminIntString(index+1) + " wajib diisi"
		}
		if body == "" {
			return input, nil, "Isi slide " + adminIntString(index+1) + " wajib diisi"
		}
		if adminRuneLen(title) > 200 {
			return input, nil, "Judul slide " + adminIntString(index+1) + " maksimal 200 karakter"
		}
		if adminRuneLen(body) > adminModuleMaxBodyRunes {
			return input, nil, "Isi slide " + adminIntString(index+1) + " maksimal 8000 karakter"
		}

		slides = append(slides, models.EducationSlide{Title: title, Body: body})
	}

	if len(slides) == 0 {
		return input, nil, "Modul harus punya minimal satu slide"
	}
	if len(slides) > adminModuleMaxSlides {
		return input, nil, "Modul maksimal berisi 30 slide"
	}

	return input, slides, ""
}

func cleanAdminQuiz(input adminQuizInput) (adminQuizInput, string) {
	if input.ModuleID == 0 {
		return input, "Modul soal wajib dipilih"
	}
	if _, found := educationModule(int(input.ModuleID)); !found {
		return input, "Modul soal tidak ditemukan"
	}

	input.Question = strings.TrimSpace(input.Question)
	input.OptionA = strings.TrimSpace(input.OptionA)
	input.OptionB = strings.TrimSpace(input.OptionB)
	input.OptionC = strings.TrimSpace(input.OptionC)
	input.OptionD = strings.TrimSpace(input.OptionD)
	input.Explanation = strings.TrimSpace(input.Explanation)
	input.CorrectAnswer = strings.ToLower(strings.TrimSpace(input.CorrectAnswer))

	if input.Question == "" {
		return input, "Pertanyaan wajib diisi"
	}
	if input.OptionA == "" || input.OptionB == "" || input.OptionC == "" || input.OptionD == "" {
		return input, "Pilihan A sampai D wajib diisi"
	}
	if adminRuneLen(input.OptionA) > 500 || adminRuneLen(input.OptionB) > 500 ||
		adminRuneLen(input.OptionC) > 500 || adminRuneLen(input.OptionD) > 500 {
		return input, "Setiap pilihan maksimal 500 karakter"
	}
	if input.CorrectAnswer != "a" && input.CorrectAnswer != "b" &&
		input.CorrectAnswer != "c" && input.CorrectAnswer != "d" {
		return input, "Kunci jawaban harus a, b, c, atau d"
	}
	if input.Explanation == "" {
		return input, "Penjelasan wajib diisi"
	}
	if adminRuneLen(input.Explanation) > 1000 {
		return input, "Penjelasan maksimal 1000 karakter"
	}

	return input, ""
}

func cleanAdminPlatform(input adminPlatformInput) (adminPlatformInput, string) {
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.ToLower(strings.TrimSpace(input.Category))
	input.Type = strings.TrimSpace(input.Type)
	input.LogoDomain = cleanPlatformDomain(input.LogoDomain)
	input.Description = strings.TrimSpace(input.Description)

	if input.Name == "" {
		return input, "Nama platform wajib diisi"
	}
	if adminRuneLen(input.Name) > 120 {
		return input, "Nama platform maksimal 120 karakter"
	}
	if !validPlatformCategory(input.Category) {
		return input, "Kategori harus crypto, saham, atau komoditas"
	}
	if input.Type != models.PlatformTypeLocal && input.Type != models.PlatformTypeForeign {
		return input, "Asal platform harus 'Lokal' atau 'Luar Negeri'"
	}
	if adminRuneLen(input.LogoDomain) > 120 {
		return input, "Domain logo maksimal 120 karakter"
	}
	if input.Description == "" {
		return input, "Deskripsi platform wajib diisi"
	}
	if adminRuneLen(input.Description) > 5000 {
		return input, "Deskripsi platform maksimal 5000 karakter"
	}

	pros, message := cleanPlatformItems(input.Pros, "Kelebihan")
	if message != "" {
		return input, message
	}
	cons, message := cleanPlatformItems(input.Cons, "Kekurangan")
	if message != "" {
		return input, message
	}
	if !hasPlatformCostWord(pros) {
		return input, "Minimal satu poin Kelebihan harus membahas biaya (fee, biaya, spread, komisi, atau admin)"
	}
	if !hasPlatformCostWord(cons) {
		return input, "Minimal satu poin Kekurangan harus membahas biaya (fee, biaya, spread, komisi, atau admin)"
	}

	input.Pros = pros
	input.Cons = cons

	return input, ""
}

func cleanPlatformItems(items []string, label string) ([]string, string) {
	lower := strings.ToLower(label)

	if len(items) < adminPlatformMinItems {
		return nil, label + " minimal tiga poin"
	}
	if len(items) > adminPlatformMaxItems {
		return nil, label + " maksimal delapan poin"
	}

	clean := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))

	for index, item := range items {
		text := strings.TrimSpace(item)
		if text == "" {
			return nil, "Poin " + lower + " ke-" + adminIntString(index+1) + " tidak boleh kosong"
		}
		if adminRuneLen(text) > adminPlatformMaxItem {
			return nil, "Poin " + lower + " ke-" + adminIntString(index+1) + " maksimal 300 karakter"
		}
		if seen[text] {
			return nil, "Poin " + lower + " ke-" + adminIntString(index+1) + " sama dengan poin sebelumnya"
		}

		seen[text] = true
		clean = append(clean, text)
	}

	return clean, ""
}

func cleanPlatformDomain(raw string) string {
	domain := strings.ToLower(strings.TrimSpace(raw))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	if index := strings.IndexAny(domain, "/?#"); index >= 0 {
		domain = domain[:index]
	}

	return strings.TrimPrefix(strings.Trim(domain, " ."), "www.")
}

func validPlatformCategory(category string) bool {
	return category == models.PlatformCategoryCrypto ||
		category == models.PlatformCategorySaham ||
		category == models.PlatformCategoryKomoditas
}

func hasPlatformCostWord(items []string) bool {
	keywords := []string{"fee", "biaya", "spread", "komisi", "admin"}

	for _, item := range items {
		lower := strings.ToLower(item)
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				return true
			}
		}
	}

	return false
}

func adminPlatformNameTaken(name string, exceptID uint) (bool, error) {
	query := facades.Orm().Query().Model(&models.InvestmentPlatform{}).Where("name = ?", name)
	if exceptID > 0 {
		query = query.Where("id <> ?", exceptID)
	}

	count, err := query.Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func adminQuiz(id int) (models.EducationQuiz, bool) {
	if id <= 0 {
		return models.EducationQuiz{}, false
	}

	var quiz models.EducationQuiz
	if err := facades.Orm().Query().Where("id = ?", id).First(&quiz); err != nil || quiz.ID == 0 {
		return models.EducationQuiz{}, false
	}

	return quiz, true
}

func adminPlatform(id int) (models.InvestmentPlatform, bool) {
	if id <= 0 {
		return models.InvestmentPlatform{}, false
	}

	var platform models.InvestmentPlatform
	if err := facades.Orm().Query().Where("id = ?", id).First(&platform); err != nil || platform.ID == 0 {
		return models.InvestmentPlatform{}, false
	}

	return platform, true
}

func adminRuneLen(value string) int {
	return len([]rune(value))
}

func adminIntString(value int) string {
	return strconv.Itoa(value)
}
