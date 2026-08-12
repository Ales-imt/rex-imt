// Package connect assemble le connecteur de programme désigné par la
// configuration.
//
// La fabrique vit dans un sous-paquet, et non dans pkg/programme, à cause d'un
// cycle d'import : les connecteurs concrets importent pkg/programme pour son
// DTO Cours, ce dernier ne peut donc pas les importer en retour.
package connect

import (
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/programme"
	"back-rex-eleve/pkg/programme/aurega"
	"back-rex-eleve/pkg/programme/bd"
	"back-rex-eleve/pkg/programme/webdfd"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sources reconnues par NewConnector.
const (
	SourceBD     = "bd"
	SourceWebdfd = "webdfd"
	SourceAurega = "aurega"
)

// NewConnector rend le connecteur correspondant à cfg.Programme.Source.
//
// Une source absente ou inconnue est une ERREUR, jamais un repli : servir un
// planning issu d'une autre source que celle configurée passerait inaperçu
// jusqu'à ce qu'un cours manque à l'appel.
func NewConnector(cfg *services.Config, db *pgxpool.Pool) (programme.ProgrammeConnector, error) {
	switch cfg.Programme.Source {
	case SourceBD:
		return &bd.Connector{DB: db}, nil
	case SourceWebdfd:
		if cfg.Programme.Webdfd.BaseURL == "" {
			return nil, fmt.Errorf("programme: source webdfd sans programme.webdfd.baseURL")
		}
		return &webdfd.Connector{PlanningURL: cfg.Programme.Webdfd.BaseURL, DB: db}, nil
	case SourceAurega:
		if cfg.Programme.Aurega.BaseURL == "" {
			return nil, fmt.Errorf("programme: source aurega sans programme.aurega.baseURL")
		}
		return &aurega.Connector{
			BaseURL: cfg.Programme.Aurega.BaseURL,
			APIKey:  cfg.Programme.Aurega.APIKey,
		}, nil
	case "":
		return nil, fmt.Errorf("programme: aucune source configurée (programme.source : %s, %s ou %s)",
			SourceBD, SourceWebdfd, SourceAurega)
	default:
		return nil, fmt.Errorf("programme: source inconnue %q (attendu : %s, %s ou %s)",
			cfg.Programme.Source, SourceBD, SourceWebdfd, SourceAurega)
	}
}
