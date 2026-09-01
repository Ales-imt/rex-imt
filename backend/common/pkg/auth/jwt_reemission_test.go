package auth

// Tests de la réémission idempotente du refresh (jetonReemissible +
// regenereRefreshToken) : un client qui a perdu la réponse de sa rotation
// rejoue son jeton et reçoit LA MÊME réponse, sans limite de temps, tant que
// le jeton courant n'a pas été consommé à son tour. Fake DBTX en mémoire, sur
// le même principe que fakeAuthDB (email_otp_test.go).

import (
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var cfgTest = services.JWTConfig{
	Secret:                "secret-de-test",
	RefreshTokenExpiresIn: 90 * 24 * time.Hour,
}

// ligneDepuis fabrique la ligne refresh_tokens qui correspondrait exactement
// au jeton généré — ce que fait CreateRefreshToken en production.
func ligneDepuis(info *RefreshTokenInfo, userID int32) RefreshToken {
	return RefreshToken{
		ID:           7,
		UserID:       userID,
		Token:        hashToken(info.Token),
		ExpiresAt:    pgtype.Timestamptz{Time: info.Expiration, Valid: true},
		TokenVersion: pgtype.Int4{Int32: int32(info.Version), Valid: true},
		Session:      info.Session,
	}
}

func TestRegenereRefreshToken_Identique(t *testing.T) {
	info, err := generateRefreshToken(nil, "4200", cfgTest)
	if err != nil {
		t.Fatal(err)
	}
	row := ligneDepuis(info, 4200)

	brut, err := regenereRefreshToken(&row, cfgTest)
	if err != nil {
		t.Fatalf("régénération refusée sur une ligne fidèle : %v", err)
	}
	if brut != info.Token {
		t.Fatal("le jeton régénéré doit être OCTET POUR OCTET celui d'origine — c'est toute l'idempotence")
	}
}

func TestRegenereRefreshToken_DivergenceDetectee(t *testing.T) {
	info, err := generateRefreshToken(nil, "4200", cfgTest)
	if err != nil {
		t.Fatal(err)
	}
	row := ligneDepuis(info, 4200)
	row.TokenVersion = pgtype.Int4{Int32: int32(info.Version) + 1, Valid: true} // ligne altérée

	if _, err := regenereRefreshToken(&row, cfgTest); err == nil {
		t.Fatal("une ligne dont les claims ne redonnent pas le hachage stocké doit être refusée, pas servie")
	}
}

// ─── jetonReemissible ────────────────────────────────────────────────────────

// fakeReemissionDB ne sert que GetRefreshTokenByPrev : une seule ligne
// vivante, retrouvée par le hachage de son prédécesseur.
type fakeReemissionDB struct {
	vivant *RefreshToken
}

func (f *fakeReemissionDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected Exec: " + sql)
}

func (f *fakeReemissionDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query: " + sql)
}

func (f *fakeReemissionDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if !strings.Contains(sql, "WHERE prev_token") {
		return &fakeRow{err: errors.New("unexpected QueryRow: " + sql)}
	}
	prev := args[0].(pgtype.Text)
	if f.vivant == nil || !f.vivant.PrevToken.Valid || f.vivant.PrevToken.String != prev.String {
		return &fakeRow{err: pgx.ErrNoRows}
	}
	return &fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*int32)) = f.vivant.ID
		*(dest[1].(*int32)) = f.vivant.UserID
		*(dest[2].(*string)) = f.vivant.Token
		*(dest[3].(*pgtype.Timestamptz)) = f.vivant.ExpiresAt
		*(dest[4].(*pgtype.Timestamptz)) = f.vivant.CreatedAt
		*(dest[5].(*bool)) = f.vivant.Revoked
		*(dest[6].(*pgtype.Int4)) = f.vivant.TokenVersion
		*(dest[7].(*string)) = f.vivant.Session
		*(dest[8].(*pgtype.Text)) = f.vivant.PrevToken
		*(dest[9].(*pgtype.Timestamptz)) = f.vivant.PrevConsumedAt
		return nil
	}}
}

// chaineDeTest crée un prédécesseur (v_n, en clair) et la ligne vivante
// (v_n+1) qui le référence par prev_token.
func chaineDeTest(t *testing.T, cfg services.JWTConfig) (predecesseur string, vivant RefreshToken) {
	t.Helper()
	infoPred, err := generateRefreshToken(nil, "4200", cfg)
	if err != nil {
		t.Fatal(err)
	}
	predRow := ligneDepuis(infoPred, 4200)
	infoVivant, err := generateRefreshToken(&predRow, "4200", cfg)
	if err != nil {
		t.Fatal(err)
	}
	vivant = ligneDepuis(infoVivant, 4200)
	vivant.PrevToken = pgtype.Text{String: hashToken(infoPred.Token), Valid: true}
	vivant.PrevConsumedAt = pgtype.Timestamptz{Time: time.Now().Add(-2 * time.Hour), Valid: true}
	return infoPred.Token, vivant
}

func TestJetonReemissible_PredecesseurDirect(t *testing.T) {
	pred, vivant := chaineDeTest(t, cfgTest)
	db := &fakeReemissionDB{vivant: &vivant}

	// Deux heures après la rotation perdue : aucune fenêtre temporelle ne doit
	// s'y opposer — c'est le scénario « je rouvre l'appli bien plus tard ».
	row, err := jetonReemissible(context.Background(), New(db), cfgTest, pred, hashToken(pred))
	if err != nil {
		t.Fatalf("le prédécesseur direct doit ouvrir droit à réémission, reçu : %v", err)
	}
	if row.Token != vivant.Token {
		t.Fatal("la ligne rendue doit être le jeton VIVANT, celui à réémettre")
	}

	// Et la boucle se ferme : la ligne rendue doit se régénérer à l'identique.
	brut, err := regenereRefreshToken(&row, cfgTest)
	if err != nil {
		t.Fatalf("la ligne vivante doit être régénérable : %v", err)
	}
	if hashToken(brut) != vivant.Token {
		t.Fatal("le jeton régénéré ne correspond pas au hachage vivant")
	}
}

func TestJetonReemissible_ChaineAvancee(t *testing.T) {
	pred, vivant := chaineDeTest(t, cfgTest)
	// La chaîne a avancé : le vivant référence un AUTRE prédécesseur.
	vivant.PrevToken = pgtype.Text{String: hashToken("un-autre-jeton"), Valid: true}
	db := &fakeReemissionDB{vivant: &vivant}

	if _, err := jetonReemissible(context.Background(), New(db), cfgTest, pred, hashToken(pred)); err != pgx.ErrNoRows {
		t.Fatalf("un jeton dépassé de deux crans est un vrai rejeu : attendu ErrNoRows, reçu %v", err)
	}
}

func TestJetonReemissible_JwtPerime(t *testing.T) {
	cfgPerime := cfgTest
	cfgPerime.RefreshTokenExpiresIn = -time.Hour // jeton né périmé
	pred, err := generateRefreshToken(nil, "4200", cfgPerime)
	if err != nil {
		t.Fatal(err)
	}
	_, vivant := chaineDeTest(t, cfgTest)
	vivant.PrevToken = pgtype.Text{String: hashToken(pred.Token), Valid: true}
	db := &fakeReemissionDB{vivant: &vivant}

	if _, err := jetonReemissible(context.Background(), New(db), cfgTest, pred.Token, hashToken(pred.Token)); err != pgx.ErrNoRows {
		t.Fatalf("un JWT périmé ne doit rien ouvrir, même prédécesseur direct : attendu ErrNoRows, reçu %v", err)
	}
}

func TestJetonReemissible_JwtEtranger(t *testing.T) {
	autreCfg := cfgTest
	autreCfg.Secret = "autre-secret"
	pred, err := generateRefreshToken(nil, "4200", autreCfg) // signé par un autre secret
	if err != nil {
		t.Fatal(err)
	}
	_, vivant := chaineDeTest(t, cfgTest)
	vivant.PrevToken = pgtype.Text{String: hashToken(pred.Token), Valid: true}
	db := &fakeReemissionDB{vivant: &vivant}

	if _, err := jetonReemissible(context.Background(), New(db), cfgTest, pred.Token, hashToken(pred.Token)); err != pgx.ErrNoRows {
		t.Fatalf("une signature étrangère ne doit rien ouvrir : attendu ErrNoRows, reçu %v", err)
	}
}
