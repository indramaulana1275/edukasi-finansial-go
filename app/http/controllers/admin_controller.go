package controllers

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// AdminController menangani fitur Panel Developer. Semua route-nya berada di
// bawah AuthMiddleware + RoleMiddleware("admin"), jadi handler di sini tidak
// perlu mengulang pemeriksaan role dari token.
type AdminController struct{}

func NewAdminController() *AdminController {
	return &AdminController{}
}

// adminUserRow adalah bentuk aman satu baris users untuk dikirim ke panel.
// Password tidak pernah ikut karena memang tidak ada di struct ini.
type adminUserRow struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	Gender          string    `json:"gender"`
	AvatarURL       string    `json:"avatar_url"`
	Bio             string    `json:"bio"`
	ModulesDone     int       `json:"modules_done"`
	ModulesTotal    int       `json:"modules_total"`
	ProgressPercent int       `json:"progress_percent"`
	CreatedAt       time.Time `json:"created_at"`
}

// Users mengembalikan seluruh akun terdaftar, terbaru lebih dulu. Panel
// membutuhkannya untuk tabel Manajemen User.
// Dashboard mengirim angka ringkas untuk kartu statistik di Panel Developer.
// Hitungannya dibaca langsung dari tabel supaya tetap benar setelah tambah,
// edit, atau hapus konten.
func (r *AdminController) Dashboard(ctx http.Context) http.Response {
	totalUsers, err := facades.Orm().Query().Model(&models.User{}).Count()
	if err != nil {
		facades.Log().Errorf("gagal menghitung user untuk admin: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca statistik"})
	}

	totalAdmins, err := facades.Orm().Query().
		Model(&models.User{}).
		Where("role = ?", models.UserRoleAdmin).
		Count()
	if err != nil {
		facades.Log().Errorf("gagal menghitung admin: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca statistik"})
	}

	totalModules, err := facades.Orm().Query().Model(&models.EducationModule{}).Count()
	if err != nil {
		facades.Log().Errorf("gagal menghitung modul: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca statistik"})
	}

	totalQuizzes, err := facades.Orm().Query().Model(&models.EducationQuiz{}).Count()
	if err != nil {
		facades.Log().Errorf("gagal menghitung soal kuis: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca statistik"})
	}

	totalPlatforms, err := facades.Orm().Query().Model(&models.InvestmentPlatform{}).Count()
	if err != nil {
		facades.Log().Errorf("gagal menghitung platform: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal membaca statistik"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data": http.Json{
			"total_users":     totalUsers,
			"total_admins":    totalAdmins,
			"total_modules":   totalModules,
			"total_quizzes":   totalQuizzes,
			"total_platforms": totalPlatforms,
		},
	})
}

func (r *AdminController) Users(ctx http.Context) http.Response {
	rows := make([]adminUserRow, 0)
	if err := facades.Orm().Query().
		Table("users").
		Select("id", "name", "email", "role", "gender", "avatar_url", "bio", "created_at").
		Order("id desc").
		Get(&rows); err != nil {
		facades.Log().Errorf("gagal membaca daftar user untuk admin: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal membaca daftar user",
		})
	}

	attachAdminProgress(rows)

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":  rows,
		"total": len(rows),
	})
}

// attachAdminProgress mengisi kolom Progres Belajar dari user_module_progress.
// Satu query agregat dipakai untuk seluruh tabel, bukan satu query per user.
func attachAdminProgress(rows []adminUserRow) {
	totalCount, err := facades.Orm().Query().Model(&models.EducationModule{}).Count()
	if err != nil {
		facades.Log().Warningf("gagal menghitung total modul untuk progres admin: %v", err)
	}
	total := int(totalCount)

	type progressRow struct {
		UserID uint
		Done   int
	}

	progress := make([]progressRow, 0)
	if err := facades.Orm().Query().
		Table("user_module_progress").
		Select("user_id, COUNT(*) AS done").
		Where("status = ?", models.ModuleStatusDone).
		Group("user_id").
		Get(&progress); err != nil {
		facades.Log().Warningf("gagal membaca progres belajar untuk admin: %v", err)
	}

	doneByUser := make(map[uint]int, len(progress))
	for _, row := range progress {
		doneByUser[row.UserID] = row.Done
	}

	for index := range rows {
		done := doneByUser[rows[index].ID]
		rows[index].ModulesDone = done
		rows[index].ModulesTotal = total
		rows[index].ProgressPercent = adminProgressPercent(done, total)
	}
}

func adminProgressPercent(done int, total int) int {
	if total <= 0 || done <= 0 {
		return 0
	}
	if done > total {
		done = total
	}

	return (done*100 + total/2) / total
}

// UpdateRole mempromosikan user menjadi admin atau mengembalikannya ke role
// user. Admin tidak boleh mengubah role akunnya sendiri supaya panel tidak
// terkunci di tengah sesi.
func (r *AdminController) UpdateRole(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "ID user tidak valid"})
	}

	actorID := currentUserID(ctx)
	if uint(id) == actorID {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "Role akun sendiri tidak bisa diubah dari panel",
		})
	}

	var input struct {
		Role string `json:"role"`
	}
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
	}

	role := strings.ToLower(strings.TrimSpace(input.Role))
	if role != models.UserRoleUser && role != models.UserRoleAdmin {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "Role harus 'user' atau 'admin'",
		})
	}

	user, found := adminUser(uint(id))
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "User tidak ditemukan"})
	}

	if _, err := facades.Orm().Query().
		Table("users").
		Where("id = ?", user.ID).
		Update("role", role); err != nil {
		facades.Log().Errorf("gagal mengubah role user %d: %v", user.ID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal mengubah role user",
		})
	}

	user.Role = role

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Role user diperbarui",
		"data":    user,
	})
}

// DeleteUser menghapus akun beserta data turunannya (foreign key memakai
// CascadeOnDelete). Admin tidak bisa menghapus akun yang sedang dipakai.
func (r *AdminController) DeleteUser(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "ID user tidak valid"})
	}

	if uint(id) == currentUserID(ctx) {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "Admin tidak bisa menghapus akunnya sendiri",
		})
	}

	result, err := facades.Orm().Query().
		Table("users").
		Where("id = ?", id).
		Delete(&models.User{})
	if err != nil {
		facades.Log().Errorf("gagal menghapus user %d: %v", id, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal menghapus user",
		})
	}

	if result.RowsAffected == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "User tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "User dihapus"})
}

func adminUser(id uint) (adminUserRow, bool) {
	var user adminUserRow
	if err := facades.Orm().Query().
		Table("users").
		Select("id", "name", "email", "role", "gender", "avatar_url", "bio", "created_at").
		Where("id = ?", id).
		FirstOrFail(&user); err != nil {
		return adminUserRow{}, false
	}

	return user, true
}
