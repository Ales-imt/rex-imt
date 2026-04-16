package services

import (
	"log"
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

// Fonction générique pour toutes les erreurs
func RenderError(w http.ResponseWriter, r *http.Request, httpStatus int, code ErrorCode, message string, detail any, logPrefix string) {
	resp := &ErrorResponse{
		HTTPStatusCode: httpStatus,
		Code:           code,
		Message:        message,
		Details:        detail,
	}
	log.Printf("%s: %v", logPrefix, resp)
	if err := render.Render(w, r, resp); err != nil {
		log.Printf(" %s: unable to render response: %v", logPrefix, err)
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
