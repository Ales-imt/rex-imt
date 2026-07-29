package auth

import (
	"back-rex-common/pkg/mailer"
	"back-rex-common/pkg/services"
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
)

// Origine de l'authentification d'un compte (colonne user.auth_source).
const (
	AuthSourceLDAP  = "ldap"
	AuthSourceEmail = "email"
)

// InternalEmailDomains liste les domaines dont les comptes sont gérés par
// l'annuaire LDAP interne. Tout autre domaine est un intervenant externe
// authentifié par code e-mail à usage unique.
var InternalEmailDomains = []string{
	"@etu.mines-ales.fr",
	"@mines-ales.fr",
}

// AuthSourceForEmail déduit la source d'authentification d'un compte à partir
// du domaine de son e-mail : LDAP pour les domaines internes (mines-ales),
// code e-mail pour tout le reste.
func AuthSourceForEmail(email string) string {
	e := strings.ToLower(strings.TrimSpace(email))
	for _, domain := range InternalEmailDomains {
		if strings.HasSuffix(e, domain) {
			return AuthSourceLDAP
		}
	}
	return AuthSourceEmail
}

const (
	loginCodeDigits     = 6
	loginCodeTTL        = 10 * time.Minute
	loginCodeMaxAttempt = 5
)

// EmailSender abstrait l'envoi du code par e-mail, pour permettre son
// remplacement dans les tests sans toucher au réseau.
type EmailSender func(ctx context.Context, cfg services.SMTPConfig, msg mailer.Message) error

const genericEmailCodeResponse = "Si un compte existe, un code a été envoyé."

type EmailCodeRequest struct {
	Email string `json:"email"`
}

func (c *EmailCodeRequest) Bind(r *http.Request) error {
	if c.Email == "" {
		return fmt.Errorf("email requis")
	}
	if len(c.Email) > 254 {
		return fmt.Errorf("email trop long")
	}
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	return nil
}

// RequestEmailCode gère POST /auth/email/request. Réponse générique que le
// compte existe ou non (anti-énumération) : le seul signal observable de
// l'extérieur doit être « un e-mail a peut-être été envoyé ».
func RequestEmailCode(w http.ResponseWriter, r *http.Request, smtpCfg services.SMTPConfig, send EmailSender) {
	data := &EmailCodeRequest{}
	if err := render.Bind(r, data); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db)

	if err := requestEmailCode(r.Context(), queries, smtpCfg, send, data.Email); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, map[string]string{"message": genericEmailCodeResponse})
}

// requestEmailCode contient la logique métier de la demande de code,
// indépendamment du contexte HTTP : testable avec un DBTX en mémoire (voir
// email_otp_test.go). Un compte inconnu ou non « email » n'est pas une
// erreur : c'est le chemin normal de l'anti-énumération.
func requestEmailCode(ctx context.Context, queries *Queries, smtpCfg services.SMTPConfig, send EmailSender, email string) error {
	userByMail, err := queries.GetUserByMailAndSource(ctx, GetUserByMailAndSourceParams{
		Email:      email,
		AuthSource: AuthSourceEmail,
	})
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	code, err := generateNumericCode(loginCodeDigits)
	if err != nil {
		return err
	}

	if err := queries.InvalidateLoginCodes(ctx, userByMail.ID); err != nil {
		return err
	}

	now := time.Now()
	expiresAt := now.Add(loginCodeTTL)
	err = queries.CreateLoginCode(ctx, CreateLoginCodeParams{
		UserID:    userByMail.ID,
		CodeHash:  hashToken(code),
		ExpiresAt: services.ToPgTimestamptz(&expiresAt),
		CreatedAt: services.ToPgTimestamptz(&now),
	})
	if err != nil {
		return err
	}

	if err := send(ctx, smtpCfg, mailer.Message{
		From:    smtpCfg.From,
		To:      []string{userByMail.Email},
		Subject: "Votre code de connexion",
		Body:    fmt.Sprintf("Votre code de connexion : %s\n\nCe code expire dans 10 minutes et ne peut être utilisé qu'une seule fois.", code),
	}); err != nil {
		slog.Error("envoi du code de connexion échoué", "error", err)
	}

	return nil
}

type EmailCodeVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (c *EmailCodeVerifyRequest) Bind(r *http.Request) error {
	if c.Email == "" || c.Code == "" {
		return fmt.Errorf("email et code requis")
	}
	if len(c.Email) > 254 || len(c.Code) > 32 {
		return fmt.Errorf("email ou code invalide")
	}
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	c.Code = strings.TrimSpace(c.Code)
	return nil
}

// VerifyEmailCode gère POST /auth/email/verify. Ne crée jamais d'utilisateur ;
// les rôles proviennent uniquement de la ligne `user` en base.
func VerifyEmailCode(w http.ResponseWriter, r *http.Request, jwtCfg services.JWTConfig) {
	data := &EmailCodeVerifyRequest{}
	if err := render.Bind(r, data); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db)

	userID, roles, err := verifyEmailCode(r.Context(), queries, data.Email, data.Code)
	if err != nil {
		if validationErr, ok := err.(*services.AppValidationError); ok {
			services.InvalidRequestError(w, r, validationErr.Error(), services.NO_INFORMATION, nil)
		} else {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		}
		return
	}

	if err := issueSession(w, r, jwtCfg, int(userID), roles); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
}

// verifyEmailCode contient la logique métier de vérification, indépendamment
// du contexte HTTP (voir email_otp_test.go). Retourne l'utilisateur et ses
// rôles (lus en base, jamais du client) à qui émettre une session.
func verifyEmailCode(ctx context.Context, queries *Queries, email string, code string) (int32, []string, error) {
	userByMail, err := queries.GetUserByMailAndSource(ctx, GetUserByMailAndSourceParams{
		Email:      email,
		AuthSource: AuthSourceEmail,
	})
	if err != nil {
		return 0, nil, services.NewAppValidationError("Code invalide", "code")
	}

	loginCode, err := queries.GetActiveLoginCodeByUser(ctx, userByMail.ID)
	if err == pgx.ErrNoRows {
		return 0, nil, services.NewAppValidationError("Code invalide", "code")
	}
	if err != nil {
		return 0, nil, err
	}

	if loginCode.ConsumedAt.Valid ||
		time.Now().After(loginCode.ExpiresAt.Time) ||
		loginCode.Attempts >= loginCodeMaxAttempt {
		return 0, nil, services.NewAppValidationError("Code invalide ou expiré", "code")
	}

	if hashToken(code) != loginCode.CodeHash {
		if err := queries.IncrementLoginCodeAttempts(ctx, loginCode.ID); err != nil {
			return 0, nil, err
		}
		return 0, nil, services.NewAppValidationError("Code invalide", "code")
	}

	if err := queries.ConsumeLoginCode(ctx, loginCode.ID); err != nil {
		return 0, nil, err
	}

	return userByMail.ID, userByMail.Roles, nil
}

// generateNumericCode génère un code numérique à `digits` chiffres via un
// CSPRNG (crypto/rand), zéro-paddé à gauche.
func generateNumericCode(digits int) (string, error) {
	max := int64(1)
	for range digits {
		max *= 10
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}
