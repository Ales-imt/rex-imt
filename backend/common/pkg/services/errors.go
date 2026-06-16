package services

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

type ErrorResponse struct {
	HTTPStatusCode int       `json:"-"`
	Code           ErrorCode `json:"code"`
	Message        string    `json:"message"`
	Details        any       `json:"details,omitempty"`
}

func (e *ErrorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func RenderError(w http.ResponseWriter, r *http.Request, httpStatus int, code ErrorCode, message string, detail any, logPrefix string) {
	// Le message d'origine est toujours loggé côté serveur, jamais perdu.
	level := slog.LevelError
	if httpStatus < 500 {
		level = slog.LevelDebug
	}
	slog.Log(r.Context(), level, "erreur",
		"type", logPrefix, "status", httpStatus, "code", code,
		"method", r.Method, "path", r.URL.Path, "detail_interne", message,
	)

	// 400 : on garde le message (validation d'entrée, rédigé par nous, non sensible).
	// Tout le reste (401/403/409/500) : message générique pour ne rien divulguer.
	clientMsg := message
	if httpStatus != http.StatusBadRequest {
		switch httpStatus {
		case 401:
			clientMsg = "Authentification requise"
		case 403:
			clientMsg = "Accès refusé"
		case 409:
			clientMsg = "Conflit lors de la modification"
		default:
			clientMsg = "Une erreur interne est survenue"
		}
	}

	resp := &ErrorResponse{
		HTTPStatusCode: httpStatus,
		Code:           code,
		Message:        clientMsg,
		Details:        detail, // toujours préservé (errors de validation)
	}
	if err := render.Render(w, r, resp); err != nil {
		slog.Error("unable to render response", "logPrefix", logPrefix, "error", err)
	}
}

// Fonctions spécifiques
func InvalidRequestError(w http.ResponseWriter, r *http.Request, message string, code ErrorCode, detail any) {
	RenderError(w, r, 400, code, message, detail, "InvalidRequestError")
}

func AuthenticationError(w http.ResponseWriter, r *http.Request, message string, code ErrorCode, detail any) {
	RenderError(w, r, 401, code, message, detail, "AuthenticationError")
}

func AuthorizationError(w http.ResponseWriter, r *http.Request, message string, code ErrorCode, detail any) {
	RenderError(w, r, 403, code, message, detail, "AuthorizationError")
}

func ConflictError(w http.ResponseWriter, r *http.Request, message string, code ErrorCode, detail any) {
	RenderError(w, r, 409, code, message, detail, "ConflictError")
}

func InternalServerError(w http.ResponseWriter, r *http.Request, message string, code ErrorCode, detail any) {
	RenderError(w, r, 500, code, message, detail, "InternalServerError")
}

type ErrorCode int

const (
	NO_INFORMATION ErrorCode = iota
	VALIDATION_ERROR
	OPTIMISTIC_LOCKING_FAILURE
)
