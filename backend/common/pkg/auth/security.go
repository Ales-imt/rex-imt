package auth

import (
	"back-rex-common/pkg/services"
	"context"
	"log"
	"log/slog"
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

// Authenticate vérifie le JWT et la session associée, puis place les
// informations d'identité (SecurityCtx) dans le contexte de la requête.
// Elle ne vérifie aucun rôle : à combiner avec RequireRoles si besoin.
func Authenticate(jwt services.JWTConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
					// Instrumentation temporaire (branche pb) : ce 401 est
					// l'autre déclencheur de déconnexion côté client (son
					// intercepteur efface les jetons) — il partait en Debug,
					// invisible en production.
					slog.WarnContext(r.Context(), "accès refusé : session absente en base (fermée ou révoquée)",
						"user", context.UserID, "session", session,
						"chemin", r.URL.Path, "ip", ipClient(r),
						"referer", r.Referer(), "user_agent", r.UserAgent())
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

			requestCtx := setSecurityCtx(r, &context)
			r = r.WithContext(requestCtx)

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoles vérifie que l'utilisateur authentifié (via Authenticate, qui
// doit s'exécuter avant dans la chaîne de middlewares) possède un des rôles
// autorisés.
func RequireRoles(allowedRoles []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, ok := r.Context().Value(SecurityCtxKey).(*SecurityCtx)
			if !ok || ctx.Role == nil || !containsRole(*ctx.Role, allowedRoles) {
				services.AuthorizationError(w, r, "droit insuffissant", services.NO_INFORMATION, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Security combine Authenticate et RequireRoles. allowedRoles == nil
// désactive le contrôle de rôle (authentification seule).
func Security(
	jwt services.JWTConfig,
	allowedRoles *[]string,
) func(next http.Handler) http.Handler {
	authenticate := Authenticate(jwt)
	if allowedRoles == nil {
		return authenticate
	}
	requireRoles := RequireRoles(*allowedRoles)

	return func(next http.Handler) http.Handler {
		return authenticate(requireRoles(next))
	}
}

// Helper pour vérifier si le rôle est autorisé
func containsRole(role string, allowedRoles []string) bool {
	for userRole := range strings.SplitSeq(role, ",") {
		if slices.Contains(allowedRoles, strings.TrimSpace(userRole)) {
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

// HasRole indique si l'utilisateur authentifié possède le rôle donné.
func HasRole(ctx context.Context, role string) bool {
	ctxValue, ok := ctx.Value(SecurityCtxKey).(*SecurityCtx)
	if !ok || ctxValue.Role == nil {
		return false
	}
	return containsRole(*ctxValue.Role, []string{role})
}

func setSecurityCtx(r *http.Request, value *SecurityCtx) context.Context {
	return context.WithValue(r.Context(), SecurityCtxKey, value)
}
