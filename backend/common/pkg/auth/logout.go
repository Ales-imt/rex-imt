package auth

import (
	"back-rex-common/pkg/services"
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
	}

	tokenPaire := genereInvalidTokenPaire()

	accessCokie := tokenPaire.accessToCookies()
	http.SetCookie(w, &accessCokie)

	refreshCookie := tokenPaire.refreshToCookies()
	http.SetCookie(w, &refreshCookie)

	w.WriteHeader(http.StatusNoContent) // 204 No Content

}
