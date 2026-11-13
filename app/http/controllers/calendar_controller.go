package controllers

import (
	"time"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/services"
)

type CalendarController struct {
	calendar *services.CalendarService
	ai       *services.AIService
}

func NewCalendarController(calendar *services.CalendarService, ai *services.AIService) *CalendarController {
	return &CalendarController{calendar: calendar, ai: ai}
}

// Index menyajikan agenda ekonomi global yang akan datang. Daftarnya masih
// statis dari CalendarService, jadi membuka halaman tidak memanggil upstream
// mana pun dan tidak memakai kuota AI.
func (r *CalendarController) Index(ctx http.Context) http.Response {
	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":       r.calendar.List(ctx.Request().Query("language", "id")),
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Explain menjawab tombol "Tanya FINANIKA AI" pada satu kartu agenda. Body-nya
// {"id":"...","language":"id"} dan agendanya dicari di daftar server, jadi model
// tidak bisa disuruh membahas topik lain.
func (r *CalendarController) Explain(ctx http.Context) http.Response {
	if errors, ok := validate(ctx, map[string]any{"id": "required"}); !ok {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"errors": errors})
	}

	var input struct {
		ID       string `json:"id"`
		Language string `json:"language"`
	}
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{
			"error": "Format permintaan tidak dikenali",
		})
	}

	event, found := r.calendar.Find(input.ID, input.Language)
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{
			"error": "Agenda itu tidak ada di Kalender Ekonomi",
		})
	}

	explanation, err := r.ai.ExplainCalendarEvent(event, input.Language)
	if err != nil {
		facades.Log().Errorf("gagal menjelaskan agenda %q: %v", event.ID, err)

		return aiFailure(ctx, err)
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"id":          event.ID,
		"name":        event.Name,
		"explanation": explanation,
	})
}
