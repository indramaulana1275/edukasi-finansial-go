package controllers

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/services"
)

// chatRequest adalah isi satu permintaan chat setelah diurai dari JSON.
type chatRequest struct {
	Message string
	Context *services.ChatContext
	Mode    string
}

// ConversationMessage adalah teks yang disimpan ke chat_messages.
func (r chatRequest) ConversationMessage() string {
	return r.Message
}

// bindChatRequest mengurai isi permintaan chat sebagai JSON.
func bindChatRequest(ctx http.Context) (chatRequest, error) {
	var body struct {
		Message string                `json:"message"`
		Context *services.ChatContext `json:"context"`
		Mode    string                `json:"mode"`
	}
	if err := ctx.Request().Bind(&body); err != nil {
		return chatRequest{}, fmt.Errorf("isi permintaan tidak terbaca: %w", err)
	}

	return chatRequest{Message: strings.TrimSpace(body.Message), Context: body.Context, Mode: strings.TrimSpace(body.Mode)}, nil
}

// chatInputFailure menandai isi permintaan yang tidak bisa diurai sebagai 400.
func chatInputFailure(ctx http.Context, err error) http.Response {
	return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
}
