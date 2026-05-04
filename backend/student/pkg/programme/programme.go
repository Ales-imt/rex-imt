package programme

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/render"
)

const elevesURL = "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe?TYPE=eleves_txt"
const planningBaseURL = "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe"

// Cours est l'évènement renvoyé au front.
type Cours struct {
	Date  string `json:"date"` // YYYY-MM-DD
	HD    string `json:"hd"`   // HH:MM
	HF    string `json:"hf"`   // HH:MM
	Cours string `json:"cours"`
	Salle string `json:"salle"`
	Prof  string `json:"prof"`
	Promo string `json:"promo"`
}

// Cache des élèves : email -> VALCLE
var elevesCache struct {
	sync.RWMutex
	data      map[string]string
	expiresAt time.Time
}

// parseKV transforme "K1;V1;K2;V2;..." en map.
func parseKV(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
}

func formatHeure(h string) string {
	if len(h) == 4 {
		return h[:2] + ":" + h[2:]
	}
	return h
}

func formatDate(d string) string {
	if len(d) == 8 {
		return d[:4] + "-" + d[4:6] + "-" + d[6:]
	}
	return d
}

func fetchAndBuildElevesMap() (map[string]string, error) {
	resp, err := http.Get(elevesURL)
	if err != nil {
		return nil, fmt.Errorf("eleves_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		ev := kv["EV"]
		if ev == "" {
			continue
		}
		// Les systèmes Cybema peuvent nommer le champ MAIL ou EMAIL.
		email := strings.ToLower(kv["MEL"])
		if email == "" {
			email = strings.ToLower(kv["MEL"])
		}
		if email == "" {
			continue
		}
		result[email] = ev
	}
	return result, nil
}

func getValcle(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	elevesCache.RLock()
	if time.Now().Before(elevesCache.expiresAt) {
		v, ok := elevesCache.data[email]
		elevesCache.RUnlock()
		if ok {
			return v, nil
		}
		return "", fmt.Errorf("étudiant non trouvé pour l'email %s", email)
	}
	elevesCache.RUnlock()

	// Rafraîchissement du cache (write lock avec double-check).
	elevesCache.Lock()
	defer elevesCache.Unlock()
	if time.Now().Before(elevesCache.expiresAt) {
		if v, ok := elevesCache.data[email]; ok {
			return v, nil
		}
		return "", fmt.Errorf("étudiant non trouvé pour l'email %s", email)
	}

	data, err := fetchAndBuildElevesMap()
	if err != nil {
		return "", err
	}
	elevesCache.data = data
	elevesCache.expiresAt = time.Now().Add(time.Hour)

	v, ok := elevesCache.data[email]
	if !ok {
		return "", fmt.Errorf("étudiant non trouvé pour l'email %s", email)
	}
	return v, nil
}

func GetProgramme(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetSecurityUserId(r.Context())
	pgCtx := services.GetPgCtx(r.Context())

	user, err := auth.New(pgCtx.Db).GetUserById(context.Background(), int32(*userID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	valcle, err := getValcle(user.Email)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// Plage de dates : accepte YYYY-MM-DD ou YYYYMMDD.
	start := strings.ReplaceAll(r.URL.Query().Get("start"), "-", "")
	end := strings.ReplaceAll(r.URL.Query().Get("end"), "-", "")
	if start == "" {
		start = time.Now().AddDate(0, 0, -7).Format("20060102")
	}
	if end == "" {
		end = time.Now().AddDate(0, 1, 0).Format("20060102")
	}

	url := fmt.Sprintf("%s?TYPE=planning_txt&DATEDEBUT=%s&DATEFIN=%s&TYPECLE=evcleunik&VALCLE=%s",
		planningBaseURL, start, end, valcle)

	resp, err := http.Get(url)
	if err != nil {
		services.InternalServerError(w, r, "planning inaccessible: "+err.Error(), services.NO_INFORMATION, nil)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	var cours []Cours
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		if kv["DATE"] == "" {
			continue
		}
		cours = append(cours, Cours{
			Date:  formatDate(kv["DATE"]),
			HD:    formatHeure(kv["HD"]),
			HF:    formatHeure(kv["HF"]),
			Cours: kv["COURS"],
			Salle: kv["SALLE"],
			Prof:  kv["PROF"],
			Promo: kv["PROMO"],
		})
	}
	if cours == nil {
		cours = []Cours{}
	}
	fmt.Println("Cours trouve: ", cours)
	render.JSON(w, r, cours)
}
