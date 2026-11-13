package middleware

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

const (
	contextUserID   = "user_id"
	contextUserRole = "user_role"
	tokenCookieName = "FINANIKA_token"
)

type AuthMiddleware struct{}

func NewAuthMiddleware() http.Middleware {
	return &AuthMiddleware{}
}

func (m *AuthMiddleware) Signature() string {
	return "auth"
}

func (m *AuthMiddleware) Handle(ctx http.Context) {
	tokenString, ok := tokenFromRequest(ctx)
	if !ok {
		if wantsHTML(ctx) {
			redirectToLogin(ctx)
			return
		}

		unauthorized(ctx, "Authorization header must be a Bearer token")
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(facades.Config().GetString("jwt.secret")), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		if wantsHTML(ctx) {
			redirectToLogin(ctx)
			return
		}

		unauthorized(ctx, "Invalid or expired token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		if wantsHTML(ctx) {
			redirectToLogin(ctx)
			return
		}

		unauthorized(ctx, "Invalid token claims")
		return
	}

	ctx.WithValue(contextUserID, claims["sub"])
	ctx.WithValue(contextUserRole, claims["role"])
	ctx.Request().Next()
}

// tokenFromRequest prefers the Bearer header used by the API client and falls
// back to the HttpOnly cookie so full-page navigation can be guarded on the
// server as well.
func tokenFromRequest(ctx http.Context) (string, bool) {
	if token, ok := bearerToken(ctx.Request().Header("Authorization")); ok {
		return token, true
	}

	return cookieToken(ctx.Request().Header("Cookie"))
}

func cookieToken(header string) (string, bool) {
	for _, part := range strings.Split(header, ";") {
		name, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found || name != tokenCookieName {
			continue
		}

		value = strings.TrimSpace(value)
		if value == "" {
			return "", false
		}

		return value, true
	}

	return "", false
}

func wantsHTML(ctx http.Context) bool {
	return strings.Contains(strings.ToLower(ctx.Request().Header("Accept")), "text/html")
}

func redirectToLogin(ctx http.Context) {
	ctx.Response().Redirect(http.StatusFound, "/dasbor#/masuk").Abort()
}

func bearerToken(header string) (string, bool) {
	token, found := strings.CutPrefix(header, "Bearer ")
	if !found {
		return "", false
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}

	return token, true
}

func unauthorized(ctx http.Context, message string) {
	ctx.Response().Json(http.StatusUnauthorized, http.Json{"error": message}).Abort()
}
