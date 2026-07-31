package bullettin

// Suivi des générations de bulletins (modèle job + polling).
// Chaque génération tourne dans une goroutine ; le front interroge l'état via
// /generate/{id}/status et récupère le zip via /generate/{id}/download.
//
// État en mémoire : convient à une instance unique (un seul conteneur admin).
// À revoir si le service passe en multi-répliques.

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"time"
)

type jobState string

const (
	jobRunning jobState = "running"
	jobDone    jobState = "done"
	jobError   jobState = "error"
)

// job décrit une génération en cours ou terminée. Seuls les champs exportés
// sont sérialisés dans /status ; zipPath et created restent internes.
type job struct {
	ID     string   `json:"id"`
	State  jobState `json:"state"`
	Done   int      `json:"done"`
	Total  int      `json:"total"`
	Format string   `json:"format"`
	Error  string   `json:"error,omitempty"`

	zipPath string // chemin du zip final, une fois l'état "done"
	created time.Time
}

type jobStore struct {
	mu   sync.Mutex
	jobs map[string]*job
	ttl  time.Duration
}

func newJobStore(ttl time.Duration) *jobStore {
	s := &jobStore{jobs: map[string]*job{}, ttl: ttl}
	go s.cleanupLoop()
	return s
}

func (s *jobStore) create(total int, format string) *job {
	j := &job{
		ID:      randomID(),
		State:   jobRunning,
		Total:   total,
		Format:  format,
		created: time.Now(),
	}
	s.mu.Lock()
	s.jobs[j.ID] = j
	s.mu.Unlock()
	return j
}

// snapshot renvoie une copie de l'état d'un job (accès concurrent sûr).
func (s *jobStore) snapshot(id string) (job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return job{}, false
	}
	return *j, true
}

func (s *jobStore) incProgress(id string) {
	s.mu.Lock()
	if j, ok := s.jobs[id]; ok {
		j.Done++
	}
	s.mu.Unlock()
}

func (s *jobStore) finish(id, zipPath string) {
	s.mu.Lock()
	if j, ok := s.jobs[id]; ok {
		j.State = jobDone
		j.zipPath = zipPath
	}
	s.mu.Unlock()
}

func (s *jobStore) fail(id, msg string) {
	s.mu.Lock()
	if j, ok := s.jobs[id]; ok {
		j.State = jobError
		j.Error = msg
	}
	s.mu.Unlock()
}

// cleanupLoop purge périodiquement les jobs expirés et leur zip temporaire.
func (s *jobStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, j := range s.jobs {
			if now.Sub(j.created) > s.ttl {
				if j.zipPath != "" {
					os.Remove(j.zipPath)
				}
				delete(s.jobs, id)
			}
		}
		s.mu.Unlock()
	}
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
