package services

import (
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type FormValidation struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type ContextKey struct {
	Name string
}

func (k *ContextKey) String() string {
	return "context value: " + k.Name
}

func ToPgText(value string) pgtype.Text {
	return pgtype.Text{
		String: value,
		Valid:  true,
	}
}

func ToPgInt4(value int) pgtype.Int4 {
	return pgtype.Int4{
		Int32: int32(value),
		Valid: true,
	}
}

func ToPgBool(value bool) pgtype.Bool {
	return pgtype.Bool{
		Bool:  value,
		Valid: true,
	}
}

func ToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{
			Valid: false,
		}
	}
	return pgtype.Timestamptz{
		Time:  *t,
		Valid: true,
	}
}

// IsValidEmail vérifie si identifiant est un mail valide
func IsValidEmail(identifiant string) bool {
	// Expression régulière simple pour valider un email
	var re = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(identifiant)
}

func CreateCookie(name string, path string, expires time.Time, value string) http.Cookie {
	return http.Cookie{
		Name: name,
		Path: path,
		//Secure: true,
		Expires:  expires,
		Value:    value,
		SameSite: http.SameSiteStrictMode,
	}

}
