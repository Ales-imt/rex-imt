package watcher

import (
	"back-rex-common/pkg/hfsqlexport"
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// emptempName est le nom sous lequel l'analyse est deposee dans la base :
// l'exe d'export la cherche a cote des .FIC, sous ce nom precis.
const emptempName = "emptemp.WDD"

// emptemp est l'analyse WinDev, embarquee dans le binaire. Elle decrit la
// structure des fichiers HFSQL et suit donc le code plutot que les donnees :
// un binaire deploye seul reste capable d'exporter.
//
//go:embed emptemp.WDD
var emptemp []byte

// Exporter enchaine, pour chaque archive deposee, les etapes de l'export :
// verification, dezippage, conversion HFSQL vers JSON.
type Exporter struct {
	Data    string        // repertoire ou est dezippee la base (ex. .../hfsql/data)
	Timeout time.Duration // garde-fou sur la duree de l'export
	Keep    int           // exports conserves sous Data ; 0 = tous
	Runner  Runner        // comment l'export est execute
}

// Runner execute la conversion HFSQL vers JSON sur la base `base`, sortie dans
// `out`. Deux implementations, selon l'endroit ou tourne le service :
//
//   - DockerRunner : le service tourne sur l'hote et delegue a un conteneur ;
//   - LocalRunner  : le service EST le conteneur wine32-hf55, et appelle wine
//     directement — aucun acces au demon docker n'est alors necessaire.
//
// C'est la seule difference entre les deux deploiements : tout le reste
// (inotify, dezippage, marqueur) est identique.
type Runner interface {
	Run(ctx context.Context, base, out string) error
	String() string
}

// Process traite une archive deposee et rend le repertoire json produit.
//
// Un fichier qui n'est pas un zip valide est ignore sans erreur : le repertoire
// de depot peut recevoir autre chose. Le repertoire rendu est alors vide, ce
// qui distingue « rien a faire » de « export reussi ».
func (e *Exporter) Process(name string) (string, error) {
	if err := VerifyZip(name); err != nil {
		log.Printf("Ignoré (pas un zip valide) : %s (%v)", name, err)
		return "", nil
	}

	root, err := ZipRoot(name)
	if err != nil {
		return "", err
	}

	log.Printf("Dézippage de %s dans %s", name, e.Data)
	base := filepath.Join(e.Data, root)
	if err := os.RemoveAll(base); err != nil {
		return "", fmt.Errorf("purge de %s : %w", base, err)
	}
	if err := Unzip(name, e.Data); err != nil {
		return "", fmt.Errorf("dézippage de %s : %w", name, err)
	}
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		return "", fmt.Errorf("répertoire attendu absent : %s", base)
	}

	// L'analyse WinDev est indispensable a l'exe d'export : sans elle, il ne
	// sait pas decrire les fichiers .FIC.
	if err := os.WriteFile(filepath.Join(base, emptempName), emptemp, 0o644); err != nil {
		return "", fmt.Errorf("écriture de %s : %w", emptempName, err)
	}

	// Un repertoire de sortie par export, pour eviter les conflits entre
	// depots rapproches (montage /out et purge du json).
	nom := hfsqlexport.NomExport(time.Now())
	out := filepath.Join(e.Data, nom)
	json := filepath.Join(out, hfsqlexport.SousDossierJSON)
	if err := os.RemoveAll(out); err != nil {
		return "", fmt.Errorf("purge de %s : %w", out, err)
	}
	if err := os.MkdirAll(json, 0o755); err != nil {
		return "", fmt.Errorf("création de %s : %w", json, err)
	}

	log.Printf("Export de %s (%s)", base, e.Runner)
	if err := e.run(base, out); err != nil {
		return "", err
	}

	// Le marqueur vient EN DERNIER, une fois tous les fichiers ecrits : c'est
	// lui seul qui autorise rex-sync a lire cet export. Sans marqueur, l'export
	// reste invisible du consommateur — mieux vaut un export ignore qu'un
	// planning lu a moitie, qui ferait annuler des seances a tort.
	if err := hfsqlexport.Marquer(e.Data, nom); err != nil {
		return "", err
	}

	// La purge vient apres : un echec laisse les exports precedents en place,
	// seule copie exploitable des donnees.
	if err := hfsqlexport.Purger(e.Data, e.Keep); err != nil {
		// Le disque qui se remplit est un probleme moins urgent qu'un export
		// perdu : on signale sans invalider l'export qui vient de reussir.
		log.Printf("watcher: purge des anciens exports : %v", err)
	}

	log.Printf("Terminé : %s", json)
	return json, nil
}

// run applique le garde-fou de duree puis delegue au Runner.
func (e *Exporter) run(base, out string) error {
	ctx := context.Background()
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	if err := e.Runner.Run(ctx, base, out); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("export interrompu après %s", e.Timeout)
		}
		return fmt.Errorf("export échoué (%s) : %w", e.Runner, err)
	}
	return nil
}
