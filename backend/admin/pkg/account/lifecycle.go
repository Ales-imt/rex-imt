package account

// Décision « supprimer, anonymiser ou conserver » appliquée à un compte sur
// demande d'un administrateur. Définition unique, partagée par DeleteUser et
// DeleteUserBulk (backend/admin/pkg/user).
//
// L'esprit de la règle : on ne supprime que ce qui n'est adossé à aucune
// obligation de conservation, et dans le doute on conserve.

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// foreignKeyViolation est le SQLSTATE d'une violation de clé étrangère.
const foreignKeyViolation = "23503"

// Beginner ouvre une transaction. Satisfait par *pgxpool.Pool comme par pgx.Tx
// (qui produit alors un point de sauvegarde).
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// SortieLookup retourne la date de sortie d'un compte, lue dans Auréga.
//
// ok == false signifie « date indéterminée » : compte inconnu d'Auréga, ou base
// indisponible. L'appelant traite ce cas en conservant le compte — jamais en
// l'anonymisant.
type SortieLookup func(ctx context.Context, email string) (date time.Time, ok bool)

// Decision décrit ce qui a été fait à un compte et pourquoi, pour que l'UI
// puisse l'annoncer sans travestir la réalité.
type Decision struct {
	UserID  int32   `json:"user_id"`
	Outcome Outcome `json:"outcome"`
	Message string  `json:"message"`
}

const (
	msgDeleted    = "compte supprimé"
	msgAnonymized = "compte anonymisé (données de présence échues)"
	msgRetained   = "suppression impossible : conservation légale des données de présence en cours " +
		"(art. 17.3.b RGPD) ; le compte a été désactivé et sera anonymisé automatiquement à l'échéance"
)

// Resolve applique à un compte la règle de fin de vie et retourne ce qui a
// réellement été fait.
//
//	1. aucun maillon de présence          → suppression physique ;
//	2. présence + horizon échu            → anonymisation en place ;
//	3. présence + horizon non échu,
//	   ou date de sortie indéterminée     → désactivation, compte conservé.
//
// Le cas 1 comporte un filet de sécurité : si le DELETE bute sur une FK
// RESTRICT inattendue (une contrainte ajoutée depuis, non anticipée ici), on
// bascule sur le cas 2/3 plutôt que de remonter une erreur.
func Resolve(
	ctx context.Context,
	db Beginner,
	reader DBTX,
	userID int32,
	lookup SortieLookup,
	now time.Time,
	anonymizeAfterYears int,
) (Decision, error) {
	hasPresence, err := HasPresence(ctx, reader, userID)
	if err != nil {
		return Decision{}, err
	}

	if !hasPresence {
		deleted, err := tryDelete(ctx, db, userID)
		if err != nil {
			return Decision{}, err
		}
		if deleted {
			return Decision{UserID: userID, Outcome: OutcomeDeleted, Message: msgDeleted}, nil
		}
		// FK RESTRICT inattendue : on retombe sur la branche de conservation.
	}

	return retain(ctx, db, reader, userID, lookup, now, anonymizeAfterYears)
}

// tryDelete tente la suppression physique dans sa propre transaction. Retourne
// false (sans erreur) si une clé étrangère l'a refusée.
func tryDelete(ctx context.Context, db Beginner, userID int32) (bool, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	if _, err := Delete(ctx, tx, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
			return false, nil
		}
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// retain applique la branche « conservation » : anonymisation si l'horizon est
// échu, désactivation sinon.
func retain(
	ctx context.Context,
	db Beginner,
	reader DBTX,
	userID int32,
	lookup SortieLookup,
	now time.Time,
	anonymizeAfterYears int,
) (Decision, error) {
	echu := false
	if lookup != nil {
		email, err := Email(ctx, reader, userID)
		if err != nil {
			return Decision{}, err
		}
		if datefin, ok := lookup(ctx, email); ok {
			echu = datefin.Before(now.AddDate(-anonymizeAfterYears, 0, 0))
		}
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return Decision{}, err
	}
	defer tx.Rollback(ctx)

	outcome, message := OutcomeDisabled, msgRetained
	if echu {
		if _, err := Anonymize(ctx, tx, userID); err != nil {
			return Decision{}, err
		}
		outcome, message = OutcomeAnonymized, msgAnonymized
	} else if _, err := Disable(ctx, tx, userID); err != nil {
		return Decision{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Decision{}, err
	}
	return Decision{UserID: userID, Outcome: outcome, Message: message}, nil
}
