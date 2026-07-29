package auth

// Tests du flux de code de connexion par e-mail (comptes externes,
// auth_source='email'). Aucune base réelle n'est contactée : un DBTX en
// mémoire simule les requêtes login_code / user, sur le même principe que
// witnessDB dans backend/admin/pkg/presence/witness_test.go.

import (
	"back-rex-common/pkg/mailer"
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeUser struct {
	id         int32
	email      string
	roles      []string
	authSource string
}

type fakeLoginCode struct {
	id         int64
	userID     int32
	codeHash   string
	expiresAt  time.Time
	consumedAt *time.Time
	attempts   int32
}

// fakeAuthDB implémente DBTX pour les requêtes utilisées par email_otp.go.
type fakeAuthDB struct {
	usersByMail map[string]fakeUser
	codes       []*fakeLoginCode
	nextCodeID  int64
}

func newFakeAuthDB() *fakeAuthDB {
	return &fakeAuthDB{usersByMail: map[string]fakeUser{}}
}

func (f *fakeAuthDB) addUser(u fakeUser) { f.usersByMail[u.email] = u }

// activeCodeFor renvoie le dernier code non consommé d'un user, comme le
// ferait GetActiveLoginCodeByUser (ORDER BY created_at DESC LIMIT 1).
func (f *fakeAuthDB) activeCodeFor(userID int32) *fakeLoginCode {
	var latest *fakeLoginCode
	for _, c := range f.codes {
		if c.userID == userID && c.consumedAt == nil {
			latest = c
		}
	}
	return latest
}

func (f *fakeAuthDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(sql, "INSERT INTO login_code"):
		f.nextCodeID++
		f.codes = append(f.codes, &fakeLoginCode{
			id:        f.nextCodeID,
			userID:    args[0].(int32),
			codeHash:  args[1].(string),
			expiresAt: args[2].(pgtype.Timestamptz).Time,
		})
		return pgconn.CommandTag{}, nil
	case strings.Contains(sql, "SET consumed_at = NOW() WHERE user_id"):
		userID := args[0].(int32)
		for _, c := range f.codes {
			if c.userID == userID && c.consumedAt == nil {
				now := time.Now()
				c.consumedAt = &now
			}
		}
		return pgconn.CommandTag{}, nil
	case strings.Contains(sql, "SET consumed_at = NOW() WHERE id"):
		id := args[0].(int64)
		for _, c := range f.codes {
			if c.id == id {
				now := time.Now()
				c.consumedAt = &now
			}
		}
		return pgconn.CommandTag{}, nil
	case strings.Contains(sql, "SET attempts = attempts + 1"):
		id := args[0].(int64)
		for _, c := range f.codes {
			if c.id == id {
				c.attempts++
			}
		}
		return pgconn.CommandTag{}, nil
	}
	return pgconn.CommandTag{}, errors.New("unexpected Exec: " + sql)
}

func (f *fakeAuthDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query: " + sql)
}

type fakeRow struct {
	err  error
	scan func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return r.scan(dest...)
}

func (f *fakeAuthDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM public.user"):
		email := args[0].(string)
		authSource := args[1].(string)
		u, ok := f.usersByMail[email]
		if !ok || u.authSource != authSource {
			return &fakeRow{err: pgx.ErrNoRows}
		}
		return &fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*int32)) = u.id
			*(dest[1].(*pgtype.Int4)) = pgtype.Int4{Int32: 1, Valid: true}
			*(dest[2].(*string)) = "Nom"
			*(dest[3].(*string)) = "Prenom"
			*(dest[4].(*string)) = u.email
			*(dest[5].(*[]string)) = u.roles
			*(dest[6].(*pgtype.Bool)) = pgtype.Bool{}
			return nil
		}}
	case strings.Contains(sql, "FROM login_code"):
		userID := args[0].(int32)
		c := f.activeCodeFor(userID)
		if c == nil {
			return &fakeRow{err: pgx.ErrNoRows}
		}
		return &fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*int64)) = c.id
			*(dest[1].(*int32)) = c.userID
			*(dest[2].(*string)) = c.codeHash
			*(dest[3].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: c.expiresAt, Valid: true}
			*(dest[4].(*pgtype.Timestamptz)) = pgtype.Timestamptz{} // consumed_at : toujours NULL ici (filtré par la requête)
			*(dest[5].(*int32)) = c.attempts
			*(dest[6].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Valid: true}
			return nil
		}}
	}
	return &fakeRow{err: errors.New("unexpected QueryRow: " + sql)}
}

var codeRe = regexp.MustCompile(`\d{6}`)

func captureSender(sent *[]mailer.Message) EmailSender {
	return func(ctx context.Context, cfg services.SMTPConfig, msg mailer.Message) error {
		*sent = append(*sent, msg)
		return nil
	}
}

// ── request : anti-énumération ───────────────────────────────────────────────

func TestRequestEmailCode_UnknownAccount_NoEmailSent(t *testing.T) {
	db := newFakeAuthDB()
	queries := New(db)
	var sent []mailer.Message

	err := requestEmailCode(context.Background(), queries, services.SMTPConfig{}, captureSender(&sent), "inconnu@exemple.org")
	if err != nil {
		t.Fatalf("un compte inconnu ne doit jamais produire d'erreur (anti-énumération), reçu: %v", err)
	}
	if len(sent) != 0 {
		t.Fatalf("aucun e-mail ne doit être envoyé pour un compte inconnu, reçu %d", len(sent))
	}
	if len(db.codes) != 0 {
		t.Fatalf("aucun code ne doit être créé pour un compte inconnu, reçu %d", len(db.codes))
	}
}

func TestRequestEmailCode_LdapAccount_NotServed(t *testing.T) {
	// Un compte interne (auth_source=ldap) ne doit jamais recevoir de code :
	// /auth/email/* ne sert que les comptes auth_source='email'.
	db := newFakeAuthDB()
	db.addUser(fakeUser{id: 1, email: "interne@mines-ales.fr", roles: []string{RoleProf}, authSource: AuthSourceLDAP})
	queries := New(db)
	var sent []mailer.Message

	if err := requestEmailCode(context.Background(), queries, services.SMTPConfig{}, captureSender(&sent), "interne@mines-ales.fr"); err != nil {
		t.Fatalf("réponse anti-énumération attendue, pas d'erreur: %v", err)
	}
	if len(sent) != 0 {
		t.Fatalf("un compte ldap ne doit jamais recevoir de code email, reçu %d envoi(s)", len(sent))
	}
}

// ── cycle nominal request → verify ───────────────────────────────────────────

func TestEmailCodeCycle_Nominal(t *testing.T) {
	db := newFakeAuthDB()
	db.addUser(fakeUser{id: 42, email: "prof.externe@exemple.org", roles: []string{RoleProf}, authSource: AuthSourceEmail})
	queries := New(db)
	var sent []mailer.Message

	if err := requestEmailCode(context.Background(), queries, services.SMTPConfig{}, captureSender(&sent), "prof.externe@exemple.org"); err != nil {
		t.Fatalf("requestEmailCode: %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("attendu un e-mail envoyé, reçu %d", len(sent))
	}
	code := codeRe.FindString(sent[0].Body)
	if code == "" {
		t.Fatalf("aucun code à 6 chiffres trouvé dans le message: %q", sent[0].Body)
	}

	userID, roles, err := verifyEmailCode(context.Background(), queries, "prof.externe@exemple.org", code)
	if err != nil {
		t.Fatalf("verifyEmailCode: %v", err)
	}
	if userID != 42 {
		t.Fatalf("userID attendu 42, reçu %d", userID)
	}
	if len(roles) != 1 || roles[0] != RoleProf {
		t.Fatalf("rôles attendus [PROF], reçu %v", roles)
	}

	c := db.codes[0]
	if c.consumedAt == nil {
		t.Fatal("le code doit être marqué consommé après vérification réussie")
	}

	// Rejeu du même code : doit échouer (déjà consommé).
	if _, _, err := verifyEmailCode(context.Background(), queries, "prof.externe@exemple.org", code); err == nil {
		t.Fatal("un code déjà consommé ne doit pas pouvoir être réutilisé")
	}
}

func TestRequestEmailCode_NewCodeInvalidatesPrevious(t *testing.T) {
	db := newFakeAuthDB()
	db.addUser(fakeUser{id: 7, email: "ext@exemple.org", roles: []string{RoleProf}, authSource: AuthSourceEmail})
	queries := New(db)
	var sent []mailer.Message
	send := captureSender(&sent)

	if err := requestEmailCode(context.Background(), queries, services.SMTPConfig{}, send, "ext@exemple.org"); err != nil {
		t.Fatal(err)
	}
	firstCode := codeRe.FindString(sent[0].Body)

	if err := requestEmailCode(context.Background(), queries, services.SMTPConfig{}, send, "ext@exemple.org"); err != nil {
		t.Fatal(err)
	}
	secondCode := codeRe.FindString(sent[1].Body)

	if _, _, err := verifyEmailCode(context.Background(), queries, "ext@exemple.org", firstCode); err == nil {
		t.Fatal("l'ancien code doit être invalidé par la nouvelle demande")
	}
	if _, _, err := verifyEmailCode(context.Background(), queries, "ext@exemple.org", secondCode); err != nil {
		t.Fatalf("le nouveau code doit rester valide: %v", err)
	}
}

// ── expiration ────────────────────────────────────────────────────────────────

func TestVerifyEmailCode_Expired(t *testing.T) {
	db := newFakeAuthDB()
	db.addUser(fakeUser{id: 9, email: "ext@exemple.org", roles: []string{RoleProf}, authSource: AuthSourceEmail})
	queries := New(db)

	code := "123456"
	db.codes = append(db.codes, &fakeLoginCode{
		id: 1, userID: 9, codeHash: hashToken(code),
		expiresAt: time.Now().Add(-time.Minute), // déjà expiré
	})

	if _, _, err := verifyEmailCode(context.Background(), queries, "ext@exemple.org", code); err == nil {
		t.Fatal("un code expiré doit être refusé")
	}
}

// ── verrouillage après trop de tentatives ────────────────────────────────────

func TestVerifyEmailCode_TooManyAttempts(t *testing.T) {
	db := newFakeAuthDB()
	db.addUser(fakeUser{id: 11, email: "ext@exemple.org", roles: []string{RoleProf}, authSource: AuthSourceEmail})
	queries := New(db)

	code := "123456"
	db.codes = append(db.codes, &fakeLoginCode{
		id: 1, userID: 11, codeHash: hashToken(code),
		expiresAt: time.Now().Add(loginCodeTTL), attempts: loginCodeMaxAttempt,
	})

	if _, _, err := verifyEmailCode(context.Background(), queries, "ext@exemple.org", code); err == nil {
		t.Fatal("un code ayant atteint le nombre maximal de tentatives doit être refusé, même avec la bonne valeur")
	}
}

func TestVerifyEmailCode_WrongCode_IncrementsAttempts(t *testing.T) {
	db := newFakeAuthDB()
	db.addUser(fakeUser{id: 13, email: "ext@exemple.org", roles: []string{RoleProf}, authSource: AuthSourceEmail})
	queries := New(db)

	db.codes = append(db.codes, &fakeLoginCode{
		id: 1, userID: 13, codeHash: hashToken("123456"),
		expiresAt: time.Now().Add(loginCodeTTL),
	})

	if _, _, err := verifyEmailCode(context.Background(), queries, "ext@exemple.org", "000000"); err == nil {
		t.Fatal("un code incorrect doit être refusé")
	}
	if db.codes[0].attempts != 1 {
		t.Fatalf("attempts attendu 1 après un échec, reçu %d", db.codes[0].attempts)
	}
}

func TestVerifyEmailCode_UnknownAccount(t *testing.T) {
	db := newFakeAuthDB()
	queries := New(db)

	if _, _, err := verifyEmailCode(context.Background(), queries, "inconnu@exemple.org", "123456"); err == nil {
		t.Fatal("un compte inconnu doit être refusé sans détail supplémentaire")
	}
}
