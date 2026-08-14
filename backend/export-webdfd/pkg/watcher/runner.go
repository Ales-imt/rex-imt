package watcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Modes d'exécution reconnus par la configuration (migration.watcher.runner).
const (
	ModeDocker = "docker"
	ModeLocal  = "local"
)

// DockerRunner délègue l'export à un conteneur, depuis un service qui tourne
// sur l'hôte.
//
// Les chemins des `-v` sont résolus par le démon, pas par l'appelant : ce mode
// suppose donc que base et out existent tels quels sur la machine du démon.
// C'est vrai sur l'hôte ; ça ne l'est plus si ce service est lui-même
// conteneurisé — utiliser ModeLocal dans ce cas.
type DockerRunner struct {
	Image string
}

func (r DockerRunner) String() string { return "docker " + r.Image }

func (r DockerRunner) Run(ctx context.Context, base, out string) error {
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"-v", base+":/data",
		"-v", out+":/out",
		r.Image)
	// La sortie du conteneur va au journal, pas au stdout du service.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// LocalRunner appelle l'export dans le processus courant, sans démon : le
// service EST alors le conteneur wine32-hf55, auquel on a ajouté ce binaire.
//
// C'est ce qui rend la conteneurisation possible. Lancer `docker run` depuis un
// conteneur imposerait de lui monter le socket du démon, donc de lui accorder
// l'équivalent du root sur l'hôte — inacceptable pour un service qui déballe
// des archives venues d'un partage réseau, et de toute façon indisponible sous
// podman.
//
// Le script d'entrée reçoit les chemins par l'environnement : il y monte les
// lecteurs DOS attendus par l'exécutable WinDev (D: la base, E: la sortie).
type LocalRunner struct {
	Entree string // script d'export ; défaut /entrypoint.sh
}

func (r LocalRunner) String() string { return "local " + r.script() }

func (r LocalRunner) script() string {
	if r.Entree == "" {
		return "/entrypoint.sh"
	}
	return r.Entree
}

func (r LocalRunner) Run(ctx context.Context, base, out string) error {
	script := r.script()
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("script d'export introuvable (%s) : le mode %q suppose que ce service tourne dans l'image wine32-hf55 : %w",
			script, ModeLocal, err)
	}

	cmd := exec.CommandContext(ctx, "/bin/bash", script)
	cmd.Env = append(os.Environ(),
		"HFSQL_BASE="+base,
		"HFSQL_OUT="+out,
	)
	// Le répertoire courant est volontairement hérité, et non déduit du chemin
	// du script : celui-ci vit à la racine alors que l'exécutable WinDev est
	// dans /app, où il est invoqué par son seul nom. C'est le WORKDIR de
	// l'image qui fait foi.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// NouveauRunner rend le Runner correspondant au mode demandé.
func NouveauRunner(mode, image, entree string) (Runner, error) {
	switch mode {
	case ModeDocker:
		return DockerRunner{Image: image}, nil
	case ModeLocal:
		return LocalRunner{Entree: entree}, nil
	default:
		return nil, fmt.Errorf("migration.watcher.runner inconnu %q (attendu : %s ou %s)",
			mode, ModeDocker, ModeLocal)
	}
}
