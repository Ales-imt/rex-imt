#!/bin/bash
# Conversion d'une base HFSQL en JSON par l'executable WinDev, sous wine.
#
# Transpose de hfsql/docker/entrypoint.sh. Deux differences, imposees par le
# fait que ce script n'est plus l'ENTRYPOINT du conteneur mais un programme
# appele par rex-export (pkg/watcher, runner "local") :
#
#   1. Les chemins ne viennent plus des montages -v (/data, /out) mais des
#      variables HFSQL_BASE et HFSQL_OUT. Le meme conteneur traite plusieurs
#      depots successifs, chacun avec ses propres repertoires.
#   2. Les arguments de l'exe ne viennent plus du CMD du conteneur, qui n'est
#      plus transmis a ce script : ils sont repris ci-dessous en valeurs par
#      defaut, utilisees quand aucun argument n'est fourni.
set -u

fail() { echo "ERREUR: $*" >&2; exit 1; }

export HOME="${HOME:-/home/wineuser}"

BASE="${HFSQL_BASE:-}"
OUT="${HFSQL_OUT:-}"

[ -n "$BASE" ] || fail "HFSQL_BASE non definie (base HFSQL dezippee)"
[ -n "$OUT" ]  || fail "HFSQL_OUT non definie (repertoire de sortie)"

# Echouer vite et clairement : sans ces controles, un probleme de droits se
# traduit par une boite de dialogue modale WinDev sur le display Xvfb invisible,
# et le conteneur reste bloque indefiniment.
[ -d "$BASE" ] || fail "$BASE n'existe pas"
[ -d "$OUT" ]  || fail "$OUT n'existe pas"
# HFSQL ouvre les .FIC/.NDX en lecture-ecriture : la base ne peut pas etre en
# lecture seule, meme si on ne veut qu'en extraire des donnees.
[ -w "$BASE" ] || fail "$BASE non inscriptible par l'UID $(id -u) : HFSQL ouvre les .FIC/.NDX en lecture-ecriture"
[ -w "$OUT" ]  || fail "$OUT non inscriptible par l'UID $(id -u)"
[ -w "$HOME" ] || fail "HOME=$HOME non inscriptible par l'UID $(id -u)"

# Les lettres de lecteur attendues par l'exe. Refaites a chaque appel : elles
# changent d'un depot a l'autre.
ln -sfn "$BASE" "$WINEPREFIX/dosdevices/d:" || fail "impossible de lier d: vers $BASE"
ln -sfn "$OUT"  "$WINEPREFIX/dosdevices/e:" || fail "impossible de lier e: vers $OUT"

# WinDev n'accepte que des repertoires existants pour /JSON=
mkdir -p "$OUT/json" || fail "impossible de creer $OUT/json"

# L'exe cree ses fichiers de travail a cote de lui, dans /app. Le conteneur
# etant desormais durable — il enchaine les depots au lieu de mourir apres
# chacun, comme le faisait `docker run --rm` — ces fichiers survivraient d'un
# export a l'autre. On repart d'un repertoire propre, sans toucher aux binaires
# livres avec l'image (.exe et .DLL).
find /app -maxdepth 1 -type f \( -iname '*.fic' -o -iname '*.ndx' -o -iname '*.mmo' \) -delete 2>/dev/null || true

# Valeurs par defaut, reprises du CMD de l'image d'origine.
if [ "$#" -eq 0 ]; then
    set -- "Mon_Projet3.exe" '/BDD=D:\' '/WDD=D:\emptemp.WDD' '/JSON=E:\json\' "/SILENCE"
fi

# Un exe WinDev cree sa fenetre principale meme en /SILENCE : il faut un
# display, sinon wine abandonne (nodrv_CreateWindow) et explorer.exe ne demarre
# pas, ce qui entraine les erreurs "class not registered".
xvfb-run -a --server-args="-screen 0 1024x768x24" \
    bash -c 'timeout -k 10 "${RUN_TIMEOUT:-600}" wine "$@"; rc=$?; wineserver -w; exit $rc' _ "$@"
rc=$?

[ $rc -eq 124 ] && echo "ERREUR: delai de ${RUN_TIMEOUT:-600}s depasse (dialogue bloquante ?)" >&2
exit $rc
