package controllers

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// AdminOpsController menangani pengaturan situs, riwayat login, dan kotak
// masukan. Route admin-nya berada di bawah AuthMiddleware + RoleMiddleware.
type AdminOpsController struct{}

func NewAdminOpsController() *AdminOpsController {
	return &AdminOpsController{}
}

// SiteSettings boleh dibaca tanpa login karena teks sambutan dan kontak
// ditampilkan juga kepada tamu. Isinya tidak pernah memuat data pribadi user.
func (r *AdminOpsController) SiteSettings(ctx http.Context) http.Response {
	row, err := siteSettingsRecord()
	if err != nil {
		facades.Log().Errorf("gagal membaca pengaturan situs: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal membaca pengaturan situs",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"data": siteSettingsPayload(row)})
}

func (r *AdminOpsController) UpdateSettings(ctx http.Context) http.Response {
	var input struct {
		WelcomeText    string `json:"welcome_text"`
		ContactEmail   string `json:"contact_email"`
		ContactPhone   string `json:"contact_phone"`
		ContactAddress string `json:"contact_address"`
	}

	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format pengaturan tidak dikenali"})
	}

	welcome := strings.TrimSpace(input.WelcomeText)
	email := strings.TrimSpace(input.ContactEmail)
	phone := strings.TrimSpace(input.ContactPhone)
	address := strings.TrimSpace(input.ContactAddress)

	if len([]rune(welcome)) > 1000 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Teks sambutan maksimal 1000 karakter"})
	}
	if len([]rune(email)) > 160 || len([]rune(phone)) > 60 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Email atau telepon kontak terlalu panjang"})
	}
	if len([]rune(address)) > 500 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Alamat kontak maksimal 500 karakter"})
	}

	row, err := siteSettingsRecord()
	if err != nil {
		facades.Log().Errorf("gagal menyiapkan pengaturan situs: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan pengaturan situs"})
	}

	row.WelcomeText = welcome
	row.ContactEmail = email
	row.ContactPhone = phone
	row.ContactAddress = address

	if err := facades.Orm().Query().Save(&row); err != nil {
		facades.Log().Errorf("gagal menyimpan pengaturan situs: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan pengaturan situs"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Pengaturan situs disimpan",
		"data":    siteSettingsPayload(row),
	})
}

// Activity menampilkan riwayat login terbaru. Dibatasi 200 baris supaya tabel
// admin tetap ringan tanpa perlu pagination tambahan.
func (r *AdminOpsController) Activity(ctx http.Context) http.Response {
	logs := make([]models.ActivityLog, 0)
	if err := facades.Orm().Query().
		Model(&models.ActivityLog{}).
		Order("id desc").
		Limit(200).
		Get(&logs); err != nil {
		facades.Log().Errorf("gagal membaca log aktivitas: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca log aktivitas"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"data": logs, "total": len(logs)})
}

func (r *AdminOpsController) Feedback(ctx http.Context) http.Response {
	items := make([]models.FeedbackMessage, 0)
	if err := facades.Orm().Query().
		Model(&models.FeedbackMessage{}).
		Order("id desc").
		Limit(200).
		Get(&items); err != nil {
		facades.Log().Errorf("gagal membaca masukan user: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca masukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"data": items, "total": len(items)})
}

func (r *AdminOpsController) UpdateFeedback(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "ID masukan tidak valid"})
	}

	var input struct {
		Status string `json:"status"`
	}
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != models.FeedbackStatusNew && status != models.FeedbackStatusRead {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Status harus 'baru' atau 'dibaca'"})
	}

	result, err := facades.Orm().Query().
		Table("feedback_messages").
		Where("id = ?", id).
		Update("status", status)
	if err != nil {
		facades.Log().Errorf("gagal memperbarui masukan %d: %v", id, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal memperbarui masukan"})
	}
	if result.RowsAffected == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Masukan tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Status masukan diperbarui"})
}

func (r *AdminOpsController) DeleteFeedback(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "ID masukan tidak valid"})
	}

	result, err := facades.Orm().Query().
		Table("feedback_messages").
		Where("id = ?", id).
		Delete(&models.FeedbackMessage{})
	if err != nil {
		facades.Log().Errorf("gagal menghapus masukan %d: %v", id, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menghapus masukan"})
	}
	if result.RowsAffected == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Masukan tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Masukan dihapus"})
}

// SubmitFeedback dipanggil dari halaman Profil Saya. Nama dan email diambil
// dari sesi, bukan dari body, supaya identitas pengirim tidak bisa dipalsukan.
func (r *AdminOpsController) SubmitFeedback(ctx http.Context) http.Response {
	userID := currentUserID(ctx)
	if userID == 0 {
		return ctx.Response().Json(http.StatusUnauthorized, http.Json{"error": "Sesi login tidak ditemukan"})
	}

	var input struct {
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
	}

	subject := strings.TrimSpace(input.Subject)
	message := strings.TrimSpace(input.Message)
	if subject == "" || message == "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Subjek dan pesan wajib diisi"})
	}
	if len([]rune(subject)) > 160 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Subjek maksimal 160 karakter"})
	}
	if len([]rune(message)) > 5000 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Pesan maksimal 5000 karakter"})
	}

	var user models.User
	if err := facades.Orm().Query().Where("id = ?", userID).FirstOrFail(&user); err != nil {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "User tidak ditemukan"})
	}

	item := models.FeedbackMessage{
		UserID:  user.ID,
		Name:    user.Name,
		Email:   user.Email,
		Subject: subject,
		Message: message,
		Status:  models.FeedbackStatusNew,
	}

	if err := facades.Orm().Query().Create(&item); err != nil {
		facades.Log().Errorf("gagal menyimpan masukan user %d: %v", user.ID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal mengirim masukan"})
	}

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "Masukan terkirim",
		"data":    item,
	})
}

func siteSettingsRecord() (models.SiteSetting, error) {
	var row models.SiteSetting
	if err := facades.Orm().Query().FirstOrFail(&row); err == nil {
		return row, nil
	}

	row = models.SiteSetting{}
	if err := facades.Orm().Query().Create(&row); err != nil {
		return models.SiteSetting{}, err
	}

	return row, nil
}

func siteSettingsPayload(row models.SiteSetting) http.Json {
	return http.Json{
		"welcome_text":    row.WelcomeText,
		"contact_email":   row.ContactEmail,
		"contact_phone":   row.ContactPhone,
		"contact_address": row.ContactAddress,
	}
}

// recordActivity menyimpan riwayat login/registrasi. Kegagalan menulis log
// tidak boleh menggagalkan proses auth, jadi hanya dicatat ke log server.
func recordActivity(ctx http.Context, user *models.User, action string) {
	if user == nil {
		return
	}

	entry := models.ActivityLog{
		UserID:    user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Action:    action,
		IP:        ctx.Request().Ip(),
		UserAgent: ctx.Request().Header("User-Agent"),
	}

	if err := facades.Orm().Query().Create(&entry); err != nil {
		facades.Log().Warningf("gagal menulis log aktivitas %s untuk user %d: %v", action, user.ID, err)
	}
}
