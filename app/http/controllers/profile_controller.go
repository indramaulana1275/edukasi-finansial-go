package controllers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	frameworkhttp "github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

const (
	profileNameMaxRunes   = 120
	profileBioMaxRunes    = 100
	profileAvatarMaxBytes = 2 << 20
	profileAvatarDir      = "public/uploads/avatars"
)

type ProfileController struct{}

func NewProfileController() *ProfileController {
	return &ProfileController{}
}

type profileInput struct {
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Bio    string `json:"bio"`
}

// Update menyimpan data profil milik user yang sedang login. Endpoint menerima
// JSON maupun multipart; multipart dipakai saat ada foto profil. Email tidak
// pernah dibaca dari body karena alamatnya berasal dari registrasi.
func (r *ProfileController) Update(ctx frameworkhttp.Context) frameworkhttp.Response {
	userID := currentUserID(ctx)
	if userID == 0 {
		return ctx.Response().Json(frameworkhttp.StatusUnauthorized, frameworkhttp.Json{"error": "Sesi login tidak ditemukan"})
	}

	var user models.User
	if err := facades.Orm().Query().Where("id = ?", userID).FirstOrFail(&user); err != nil {
		return ctx.Response().Json(frameworkhttp.StatusNotFound, frameworkhttp.Json{"error": "User tidak ditemukan"})
	}

	input, err := bindProfileInput(ctx)
	if err != nil {
		return ctx.Response().Json(frameworkhttp.StatusBadRequest, frameworkhttp.Json{"error": "Format profil tidak dikenali"})
	}

	name := strings.TrimSpace(input.Name)
	gender, genderOK := normaliseProfileGender(input.Gender)
	bio := strings.TrimSpace(input.Bio)

	if name == "" {
		return profileError(ctx, "Nama lengkap wajib diisi.")
	}
	if profileRuneLen(name) > profileNameMaxRunes {
		return profileError(ctx, "Nama lengkap maksimal 120 karakter.")
	}
	if !genderOK {
		return profileError(ctx, "Jenis kelamin harus Laki-laki atau Perempuan.")
	}
	if profileRuneLen(bio) > profileBioMaxRunes {
		return profileError(ctx, "Biodata maksimal 100 karakter.")
	}

	user.Name = name
	user.Gender = gender
	user.Bio = bio

	avatarURL, err := saveProfileAvatar(ctx, user.ID, user.AvatarURL)
	if err != nil {
		if errors.Is(err, errProfileAvatarType) {
			return profileError(ctx, "Format foto belum didukung. Pakai JPG, PNG, atau WEBP.")
		}
		if errors.Is(err, errProfileAvatarTooLarge) {
			return profileError(ctx, "Ukuran foto maksimal 2 MB.")
		}

		facades.Log().Errorf("gagal menyimpan foto profil user %d: %v", user.ID, err)
		return ctx.Response().Json(frameworkhttp.StatusInternalServerError, frameworkhttp.Json{"error": "Foto profil gagal disimpan"})
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	if err := facades.Orm().Query().Save(&user); err != nil {
		facades.Log().Errorf("gagal menyimpan profil user %d: %v", user.ID, err)
		return ctx.Response().Json(frameworkhttp.StatusInternalServerError, frameworkhttp.Json{"error": "Profil gagal disimpan"})
	}

	return ctx.Response().Json(frameworkhttp.StatusOK, frameworkhttp.Json{
		"message": "Profil diperbarui",
		"user":    userResource(&user),
	})
}

func bindProfileInput(ctx frameworkhttp.Context) (profileInput, error) {
	var input profileInput

	if strings.HasPrefix(ctx.Request().Header("Content-Type"), "multipart/form-data") {
		input.Name = ctx.Request().Input("name")
		input.Gender = ctx.Request().Input("gender")
		input.Bio = ctx.Request().Input("bio")
		return input, nil
	}

	if err := ctx.Request().Bind(&input); err != nil {
		return profileInput{}, err
	}

	return input, nil
}

func normaliseProfileGender(raw string) (string, bool) {
	gender := strings.TrimSpace(raw)
	switch {
	case gender == "":
		return "", true
	case strings.EqualFold(gender, "laki-laki"), strings.EqualFold(gender, "laki laki"), strings.EqualFold(gender, "pria"):
		return "Laki-laki", true
	case strings.EqualFold(gender, "perempuan"), strings.EqualFold(gender, "wanita"):
		return "Perempuan", true
	default:
		return "", false
	}
}

var (
	errProfileAvatarType     = errors.New("tipe foto profil tidak didukung")
	errProfileAvatarTooLarge = errors.New("foto profil terlalu besar")
)

func saveProfileAvatar(ctx frameworkhttp.Context, userID uint, current string) (string, error) {
	file, err := ctx.Request().File("avatar")
	if err != nil || file == nil {
		return "", nil
	}

	handle, err := os.Open(file.File())
	if err != nil {
		return "", err
	}
	defer handle.Close()
	defer os.Remove(file.File())

	data, err := io.ReadAll(io.LimitReader(handle, profileAvatarMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	if len(data) > profileAvatarMaxBytes {
		return "", errProfileAvatarTooLarge
	}

	extensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
	}
	extension, ok := extensions[http.DetectContentType(data)]
	if !ok {
		return "", errProfileAvatarType
	}

	if err := os.MkdirAll(profileAvatarDir, 0o755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d-%d%s", userID, time.Now().UnixNano(), extension)
	target := filepath.Join(profileAvatarDir, filename)
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}

	return "/" + filepath.ToSlash(target), nil
}

func profileError(ctx frameworkhttp.Context, message string) frameworkhttp.Response {
	return ctx.Response().Json(frameworkhttp.StatusUnprocessableEntity, frameworkhttp.Json{"error": message})
}

func profileRuneLen(value string) int {
	return len([]rune(value))
}
