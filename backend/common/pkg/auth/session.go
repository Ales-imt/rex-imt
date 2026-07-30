package auth

import (
	"back-rex-common/pkg/services"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
)

// ErrUserBlamed signale qu'un utilisateur banni tente d'ouvrir une session.
// Retourné par issueSession pour que les callers répondent par une erreur
// d'autorisation plutôt qu'une 500. Voir aussi le contrôle au rafraîchissement
// dans jwt.go.
var ErrUserBlamed = errors.New("Utilisateur banni")

// issueSession génère la paire de jetons JWT, enregistre le refresh token et
// répond avec un LoginResponse. Partagé par le login LDAP et la vérification
// de code par e-mail : seule l'authentification en amont diffère, l'émission
// de session est strictement identique dans les deux cas.
func issueSession(w http.ResponseWriter, r *http.Request, jwtCfg services.JWTConfig, userID int, roles []string) error {
	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db)

	// Un utilisateur banni ne doit jamais obtenir de session : on le bloque
	// avant d'émettre le moindre jeton (le contrôle jwt.go ne couvrait que le
	// rafraîchissement, laissant passer le login initial).
	user, err := queries.GetUserById(r.Context(), int32(userID))
	if err != nil {
		return err
	}
	if user.Blame.Valid && user.Blame.Bool {
		return ErrUserBlamed
	}

	claims := jwt.MapClaims{"roles": strings.Join(roles, ",")}
	subject := strconv.Itoa(userID)

	tokenPaire, err := genereTokenPaire(jwtCfg, nil, &claims, subject)
	if err != nil {
		return err
	}

	now := time.Now()
	err = queries.CreateRefreshToken(r.Context(), CreateRefreshTokenParams{
		Userid:       int32(userID),
		Token:        hashToken(tokenPaire.RefreshTokenInfo.Token),
		Expire:       services.ToPgTimestamptz(&tokenPaire.RefreshTokenInfo.Expiration),
		Session:      tokenPaire.RefreshTokenInfo.Session,
		TokenVersion: services.ToPgInt4(tokenPaire.RefreshTokenInfo.Version),
		Created:      services.ToPgTimestamptz(&now),
		Revoked:      false,
	})
	if err != nil {
		return err
	}

	render.JSON(w, r, &LoginResponse{
		Name:         user.Name,
		Surname:      user.Surname,
		Roles:        roles,
		AccessToken:  tokenPaire.AccessToken.Token,
		RefreshToken: tokenPaire.RefreshTokenInfo.Token,
	})
	return nil
}
