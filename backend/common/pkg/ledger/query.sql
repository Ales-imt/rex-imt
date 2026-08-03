-- Requêtes du registre de présence (presence_ledger), partagées entre
-- back-rex-admin (ancrage TSA, vérification) et back-rex-eleve (pointage).
-- L'UNIQUE implémentation du chaînage est dans pkg/ledger/ledger.go.

-- name: AcquireLedgerLock :exec
-- Sérialise les écritures concurrentes du registre : sans ce verrou, deux
-- insertions simultanées pourraient partager le même prev_hash et casser la
-- chaîne. Verrou de niveau transaction, relâché automatiquement au COMMIT ou
-- au ROLLBACK. À n'appeler que dans une transaction déjà ouverte.
SELECT pg_advisory_xact_lock(@key);

-- name: GetLastLedgerEntry :one
-- Dernier maillon du registre (démarrage d'AppendLedger et ancrage TSA).
SELECT seq, hash FROM presence_ledger ORDER BY seq DESC LIMIT 1;

-- name: InsertLedgerEntry :one
-- Insère un maillon dans le registre. recorded_at est fourni explicitement
-- (pas de DEFAULT) pour être connu avant le calcul du hash.
INSERT INTO presence_ledger (seance_id, user_id, statut, event_at, recorded_at, prev_hash, hash)
VALUES (@seance_id, @user_id, @statut, @event_at, @recorded_at, @prev_hash, @hash)
RETURNING seq;

-- name: ListLedgerBySeq :many
-- Parcours ordonné pour VerifyChain : recalcule chaque hash et compare.
SELECT seq, seance_id, user_id, statut, event_at, recorded_at, prev_hash, hash
FROM presence_ledger
ORDER BY seq ASC;

-- name: GetLedgerEntryByHash :one
-- Recherche un maillon par son empreinte (vérification d'un témoin externe :
-- le hash scellé par la TSA doit exister dans la chaîne actuelle).
SELECT seq, event_at, recorded_at FROM presence_ledger WHERE hash = @hash;
