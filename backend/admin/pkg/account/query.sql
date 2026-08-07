-- Cycle de vie d'un compte utilisateur — définition UNIQUE, partagée entre la
-- purge RGPD automatique (backend/admin/pkg/rgpd) et le CRUD admin
-- (backend/admin/pkg/user).
--
-- INVARIANT : une ligne "user" référencée par presence_ledger n'est JAMAIS
-- supprimée. La FK fk_presence_ledger_user est ON DELETE RESTRICT et le hash
-- du registre scelle l'ENTIER user_id (voir common/pkg/ledger/hash.go). On
-- anonymise donc EN PLACE : l'id est conservé, la FK reste satisfaite, tous les
-- hachés restent recalculables et VerifyChain continue de passer.
--
-- Attention : les autres FK vers "user" (pointage, postit, eleve_groupe,
-- student, login_code, refresh_tokens) sont ON DELETE CASCADE. Un DELETE sur un
-- compte détruirait donc ses pointages — raison de plus pour ne jamais
-- supprimer un compte porteur de présence.

-- name: HasPresenceLedger :one
-- Un compte porteur d'au moins un maillon de registre est soumis à conservation.
SELECT EXISTS(SELECT 1 FROM presence_ledger WHERE user_id = $1);

-- name: DisableUser :execrows
-- Étape 1 du cycle : coupe l'accès sans toucher à l'identité, qui reste
-- nécessaire pour rattacher les pièces de présence à une personne pendant tout
-- l'horizon de conservation. Idempotent : 0 ligne affectée si déjà désactivé.
UPDATE public.user
SET disabled_at = now(),
    version     = version + 1
WHERE id = $1 AND disabled_at IS NULL;

-- name: AnonymizeUser :execrows
-- Étape 2 du cycle : efface l'identité nominative EN PLACE, une fois l'horizon
-- de conservation échu. L'id entier est conservé — c'est lui qui est scellé
-- dans le registre.
--
-- L'email reçoit un placeholder unique par id : la colonne est NOT NULL et
-- porte la contrainte UNIQUE user_email_key.
-- auth_source sert de marqueur d'idempotence (aucune contrainte CHECK ne
-- restreint cette colonne) : 0 ligne affectée au second appel.
--
-- Note : la colonne ldapid n'existe plus (supprimée par le changeset
-- v0.0.1-001a-alter-user-ldapid-blame), il n'y a donc rien à y neutraliser.
UPDATE public.user
SET name        = '',
    surname     = '',
    email       = 'anonymise-' || id || '@invalid.local',
    auth_source = 'anonymized',
    disabled_at = COALESCE(disabled_at, now()),
    version     = version + 1
WHERE id = $1 AND auth_source <> 'anonymized';

-- name: RevokeRefreshTokensByUser :execrows
-- Révocation à la désactivation. Même geste que CleanUpTokens
-- (common/pkg/auth/jwt_services.go), restreint à un utilisateur.
DELETE FROM refresh_tokens WHERE user_id = $1;

-- name: RevokeLoginCodesByUser :execrows
-- Révocation à la désactivation. Même geste que CleanUpLoginCodes
-- (StartLoginCodeCleanup), restreint à un utilisateur.
DELETE FROM login_code WHERE user_id = $1;

-- name: DeleteUserPhysical :execrows
-- Suppression réelle, réservée aux comptes SANS aucun maillon de présence.
-- Les FK ON DELETE CASCADE nettoient postit, login_code, refresh_tokens,
-- eleve_groupe, student et migration.*_map.
DELETE FROM public.user WHERE id = $1;

-- name: GetUserEmailByID :one
-- Email d'un compte, pour interroger Auréga sur sa date de sortie.
SELECT email FROM public.user WHERE id = $1;
