package migration

import "back-rex-sync/pkg/source"

// sourceFixe est une source.Source en dur : elle sert à éprouver les étapes de
// synchronisation sans dépendre d'un flux amont. Le parsing de chaque source
// réelle est testé chez elle (pkg/migration/webdfd, pkg/migration/watcher).
type sourceFixe struct {
	promos   []source.Promo
	profs    []source.Personne
	eleves   []source.Personne
	cours    map[string]string
	salles   []source.Salle
	planning map[string][]source.Creneau
	membres  map[string][]string
}

func (s sourceFixe) Promos() ([]source.Promo, error)       { return s.promos, nil }
func (s sourceFixe) Profs() ([]source.Personne, error)     { return s.profs, nil }
func (s sourceFixe) Eleves() ([]source.Personne, error)    { return s.eleves, nil }
func (s sourceFixe) CoursNoms() (map[string]string, error) { return s.cours, nil }
func (s sourceFixe) Salles() ([]source.Salle, error)       { return s.salles, nil }

func (s sourceFixe) Planning(p0cle, debut, fin string) ([]source.Creneau, error) {
	var out []source.Creneau
	for _, e := range s.planning[p0cle] {
		if e.Date == "" || e.Date < debut || e.Date > fin {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (s sourceFixe) MembresGroupe(grcle string) ([]string, string, error) {
	return s.membres[grcle], "", nil
}

var _ source.Source = sourceFixe{}
