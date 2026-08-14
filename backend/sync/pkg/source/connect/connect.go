// Package connect assemble la source désignée par la configuration.
//
// La fabrique vit dans un sous-paquet, et non dans pkg/source, à cause d'un
// cycle d'import : les sources concrètes importent pkg/source pour son
// interface et ses DTO, ce dernier ne peut donc pas les importer en retour.
// Même découpage que back-rex-eleve/pkg/programme/connect.
package connect

import (
	"back-rex-common/pkg/services"
	"back-rex-sync/pkg/source"
	"back-rex-sync/pkg/source/hfsql"
	"back-rex-sync/pkg/source/webdfd"
	"fmt"
)

// Nouvelle rend la source correspondant à cfg.Migration.Source.
//
// Une source absente ou inconnue est une ERREUR, jamais un repli : synchroniser
// depuis une autre source que celle configurée passerait inaperçu jusqu'à ce
// qu'un cours manque à l'appel.
//
// jsonDir n'est utilisé que par la source hfsql : c'est le répertoire d'export
// à monter, qui change à chaque nouveau dépôt.
func Nouvelle(cfg *services.Config, jsonDir string) (source.Source, error) {
	switch cfg.Migration.Source {
	case source.NomWebdfd:
		if cfg.Webdfd.BaseURL == "" {
			return nil, fmt.Errorf("migration: source webdfd sans webdfd.baseURL")
		}
		return webdfd.NewSource(cfg.Webdfd), nil
	case source.NomHFSQL:
		if jsonDir == "" {
			return nil, fmt.Errorf("migration: source hfsql sans répertoire d'export")
		}
		return hfsql.NewSource(jsonDir)
	case "":
		return nil, fmt.Errorf("migration: aucune source configurée (migration.source : %s ou %s)",
			source.NomWebdfd, source.NomHFSQL)
	default:
		return nil, fmt.Errorf("migration: source inconnue %q (attendu : %s ou %s)",
			cfg.Migration.Source, source.NomWebdfd, source.NomHFSQL)
	}
}
