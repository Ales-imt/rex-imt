package auth

import (
	"back-rex-common/pkg/services"
	"log/slog"
	"net/http"
)

func Logout(w http.ResponseWriter, r *http.Request, jwt services.JWTConfig) {

	session := GetSecuritySession(r.Context())

	if session != nil {
		pgCtx := services.GetPgCtx(r.Context())
		queries := New(pgCtx.Db)

		err := queries.DeleteRefreshTokenBySession(r.Context(), *session)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		// Instrumentation temporaire (branche pb) : savoir QUAND une session
		// disparaît volontairement — un rejeu ultérieur de ses jetons partira
		// en « session fermée », pas en « déjà consommé ».
		slog.InfoContext(r.Context(), "logout : session supprimée",
			"session", *session, "ip", ipClient(r), "user_agent", r.UserAgent())
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content

}
