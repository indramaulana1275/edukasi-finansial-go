package controllers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/goravel/framework/contracts/http"
	"golang.org/x/crypto/bcrypt"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

const authCookieName = "FINANIKA_token"

type AuthController struct{}

func (ctrl *AuthController) Register(ctx http.Context) http.Response {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if errors, ok := validate(ctx, map[string]any{
		"name":     "required",
		"email":    "required|email",
		"password": "required|min_len:6",
	}); !ok {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"errors": errors})
	}

	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
	}

	var existing models.User
	if err := facades.Orm().Query().Where("email = ?", input.Email).FirstOrFail(&existing); err == nil {
		return ctx.Response().Json(http.StatusConflict, http.Json{"error": "Email already registered"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Failed to hash password"})
	}

	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword),
		Role:     models.UserRoleUser,
	}

	if err := facades.Orm().Query().Create(&user); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Failed to create user"})
	}

	recordActivity(ctx, &user, "register")

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "User registered successfully",
		"user":    userResource(&user),
	})
}

func (ctrl *AuthController) Login(ctx http.Context) http.Response {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if errors, ok := validate(ctx, map[string]any{
		"email":    "required|email",
		"password": "required",
	}); !ok {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"errors": errors})
	}

	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
	}

	var user models.User
	if err := facades.Orm().Query().Where("email = ?", input.Email).FirstOrFail(&user); err != nil {
		return ctx.Response().Json(http.StatusUnauthorized, http.Json{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return ctx.Response().Json(http.StatusUnauthorized, http.Json{"error": "Invalid credentials"})
	}

	token, err := generateJWT(&user)
	if err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Failed to generate token"})
	}

	recordActivity(ctx, &user, "login")

	return ctx.Response().Cookie(authCookie(token)).Json(http.StatusOK, http.Json{
		"message": "Login successful",
		"token":   token,
		"user":    userResource(&user),
	})
}

func (ctrl *AuthController) Logout(ctx http.Context) http.Response {
	return ctx.Response().Cookie(authCookie("")).Json(http.StatusOK, http.Json{"message": "Logout successful"})
}

func (ctrl *AuthController) Me(ctx http.Context) http.Response {
	var user models.User
	if err := facades.Orm().Query().Where("id = ?", ctx.Value("user_id")).FirstOrFail(&user); err != nil {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "User not found"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"user": userResource(&user)})
}

func userResource(user *models.User) http.Json {
	return http.Json{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"gender":     user.Gender,
		"avatar_url": user.AvatarURL,
		"bio":        user.Bio,
	}
}

// authCookie keeps the JWT available for full-page navigation. The API client
// still sends the Bearer header; the HttpOnly cookie is only a server-side
// guard so protected URLs can redirect instead of returning raw JSON.
func authCookie(token string) http.Cookie {
	ttl := facades.Config().GetInt("jwt.ttl", 60)
	if ttl <= 0 {
		ttl = 60
	}

	expires := time.Now().Add(time.Duration(ttl) * time.Minute)
	maxAge := ttl * 60
	if token == "" {
		expires = time.Now().Add(-time.Hour)
		maxAge = -1
	}

	return http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: "Lax",
	}
}

func generateJWT(user *models.User) (string, error) {
	ttl := facades.Config().GetInt("jwt.ttl", 60)

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(ttl) * time.Minute).Unix(),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(facades.Config().GetString("jwt.secret")))
}

func validate(ctx http.Context, rules map[string]any) (map[string]map[string]string, bool) {
	validator, err := ctx.Request().Validate(rules)
	if err != nil {
		return map[string]map[string]string{"_error": {"message": err.Error()}}, false
	}

	if validator.Fails() {
		return validator.Errors().All(), false
	}

	return nil, true
}
