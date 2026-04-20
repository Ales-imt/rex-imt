package auth

import (
	"back-rex-common/pkg/services"
	"context"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

var SecurityCtxKey = &services.ContextKey{Name: "security_entry"}

type SecurityCtx struct {
	UserID  int
	Session *string
	Role    *string
}

func Security(
	jwt services.JWTConfig,
	allowedRoles *[]string,
) func(next http.Handler) http.Handler {

	security := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			claim, err := getClaims(r, jwt.Secret)
			if err != nil {
				services.AuthenticationError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

			context := SecurityCtx{}

			userId, err := getSubjectFromClaims(claim)
			if err != nil {
				services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

			id, err := strconv.Atoi(*userId)
			if err != nil {
				services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			}

			context.UserID = id

			if (*claim)["session_id"] != nil {
				session, ok := (*claim)["session_id"].(string)
				if !ok {
					services.InvalidRequestError(w, r, "session_id introuvable", services.NO_INFORMATION, nil)
					return
				}
				context.Session = &session

				pgCtx := services.GetPgCtx(r.Context())
				queries := New(pgCtx.Db)
				_, err := queries.GetRefreshTokenBySession(r.Context(), session)
				if err != nil {
					services.AuthenticationError(w, r, "session révoquée", services.NO_INFORMATION, nil)
					return
				}
			}

			if (*claim)["roles"] != nil {
				role, ok := (*claim)["roles"].(string)
				if !ok {
					services.InvalidRequestError(w, r, "roles introuvable", services.NO_INFORMATION, nil)
					return
				}
				context.Role = &role
			}

			if allowedRoles != nil && context.Role != nil {
				// Vérifie le rôle
				if !containsRole(*context.Role, allowedRoles) {
					services.AuthorizationError(w, r, "droit insuffissant", services.NO_INFORMATION, nil)
					return
				}
			}

			requestCtx := setSecurityCtx(r, &context)
			r = r.WithContext(requestCtx)

			next.ServeHTTP(w, r)
		})
	}
	return security
}

// Helper pour vérifier si le rôle est autorisé
func containsRole(role string, allowedRoles *[]string) bool {
	for userRole := range strings.SplitSeq(role, ",") {
		if slices.Contains(*allowedRoles, strings.TrimSpace(userRole)) {
			return true
		}
	}
	return false
}

func GetSecurityUserId(ctx context.Context) *int {
	if pgFromCtx, ok := ctx.Value(SecurityCtxKey).(*SecurityCtx); ok {
		return &pgFromCtx.UserID
	}
	log.Fatal("GetSecurityCtx n'est pas du type *SecurityCtx")
	return nil // ne passera jamais ici, car sortira avant
}

func GetSecuritySession(ctx context.Context) *string {
	if pgFromCtx, ok := ctx.Value(SecurityCtxKey).(*SecurityCtx); ok {
		return pgFromCtx.Session
	}
	log.Fatal("GetSecurityCtx n'est pas du type *SecurityCtx")
	return nil // ne passera jamais ici, car sortira avant
}

func setSecurityCtx(r *http.Request, value *SecurityCtx) context.Context {
	return context.WithValue(r.Context(), SecurityCtxKey, value)
}
