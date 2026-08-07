// Package account porte le cycle de vie des comptes utilisateurs : une seule
// définition de « désactiver » et de « anonymiser », partagée par la purge RGPD
// automatique (pkg/rgpd) et par le CRUD admin (pkg/user).
//
// Cycle en deux étapes, ancré sur la date de sortie lue dans Auréga :
//
//	départ + 1 an   → Disable   : accès coupé, identité CONSERVÉE (elle reste
//	                              nécessaire pour rattacher les pièces de
//	                              présence à une personne).
//	départ + 10 ans → Anonymize : identité effacée EN PLACE, id conservé.
//
// INVARIANT ABSOLU : ni presence_ledger, ni pointage.user_id, ni la fonction de
// hash ne sont jamais touchés. Un compte porteur de présence n'est jamais
// supprimé — il est anonymisé en place, ce qui satisfait la FK RESTRICT et
// laisse tous les hachés recalculables.
package account

import (
	"back-rex-admin/pkg/account/gen"
	"context"
	"fmt"
)

// DBTX accepte indifféremment un pool ou une transaction.
type DBTX = gen.DBTX

// Outcome décrit ce qui a réellement été fait à un compte. Il sert à répondre
// honnêtement à l'UI : on n'annonce jamais une suppression qui n'a pas eu lieu.
type Outcome string

const (
	// OutcomeDeleted : compte réellement supprimé (aucune présence enregistrée).
	OutcomeDeleted Outcome = "supprime"
	// OutcomeAnonymized : identité effacée en place, horizon de conservation échu.
	OutcomeAnonymized Outcome = "anonymise"
	// OutcomeDisabled : conservation légale en cours — accès coupé, identité gardée.
	OutcomeDisabled Outcome = "conserve"
)

// Disable coupe l'accès d'un compte sans toucher à son identité, et révoque ses
// sessions et codes de connexion en cours.
//
// Idempotent : changed vaut false si le compte était déjà désactivé. La
// révocation est rejouée dans tous les cas — elle est elle-même idempotente et
// c'est la garantie qu'aucune session ne survit à un appel.
func Disable(ctx context.Context, db DBTX, userID int32) (changed bool, err error) {
	q := gen.New(db)

	n, err := q.DisableUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("désactivation compte %d: %w", userID, err)
	}

	if err := revoke(ctx, q, userID); err != nil {
		return false, err
	}
	return n > 0, nil
}

// Anonymize efface l'identité nominative en place, une fois l'horizon de
// conservation échu. L'id entier est conservé : c'est lui qui est scellé dans
// presence_ledger. Les pointages ne sont pas touchés (aucun DELETE, donc aucune
// cascade).
//
// Idempotent : changed vaut false si le compte était déjà anonymisé.
func Anonymize(ctx context.Context, db DBTX, userID int32) (changed bool, err error) {
	q := gen.New(db)

	n, err := q.AnonymizeUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("anonymisation compte %d: %w", userID, err)
	}

	if err := revoke(ctx, q, userID); err != nil {
		return false, err
	}
	return n > 0, nil
}

// HasPresence indique si le compte porte au moins un maillon de registre, donc
// s'il est soumis à l'obligation de conservation.
func HasPresence(ctx context.Context, db DBTX, userID int32) (bool, error) {
	has, err := gen.New(db).HasPresenceLedger(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("vérification présence compte %d: %w", userID, err)
	}
	return has, nil
}

// Delete supprime physiquement un compte. À n'appeler qu'après avoir vérifié
// HasPresence == false : les FK ON DELETE CASCADE détruiraient sinon les
// pointages de l'utilisateur.
func Delete(ctx context.Context, db DBTX, userID int32) (int64, error) {
	n, err := gen.New(db).DeleteUserPhysical(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("suppression compte %d: %w", userID, err)
	}
	return n, nil
}

// Email retourne l'adresse d'un compte, pour interroger Auréga sur sa sortie.
func Email(ctx context.Context, db DBTX, userID int32) (string, error) {
	email, err := gen.New(db).GetUserEmailByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("lecture email compte %d: %w", userID, err)
	}
	return email, nil
}

// revoke supprime les sessions et codes de connexion d'un compte.
func revoke(ctx context.Context, q *gen.Queries, userID int32) error {
	if _, err := q.RevokeRefreshTokensByUser(ctx, userID); err != nil {
		return fmt.Errorf("révocation tokens compte %d: %w", userID, err)
	}
	if _, err := q.RevokeLoginCodesByUser(ctx, userID); err != nil {
		return fmt.Errorf("révocation codes compte %d: %w", userID, err)
	}
	return nil
}
