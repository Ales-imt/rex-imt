package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// lire recupere le prochain lot d'evenements sans bloquer indefiniment le test.
func lire(t *testing.T, w *Watcher) []Event {
	t.Helper()
	type resultat struct {
		events []Event
		err    error
	}
	ch := make(chan resultat, 1)
	go func() {
		events, err := w.Read()
		ch <- resultat{events, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("Read : %v", r.err)
		}
		return r.events
	case <-time.After(5 * time.Second):
		t.Fatal("aucun événement reçu")
		return nil
	}
}

func TestWatcherCloseWrite(t *testing.T) {
	depot := t.TempDir()
	w, err := NewWatcher(depot, unix.IN_CLOSE_WRITE|unix.IN_MOVED_TO)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	depose := filepath.Join(depot, "20250707.zip")
	f, err := os.Create(depose)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("PK")); err != nil {
		t.Fatal(err)
	}
	// IN_CLOSE_WRITE n'arrive qu'a la fermeture : l'ecriture ci-dessus ne doit
	// declencher aucun evenement, c'est tout l'interet par rapport a IN_MODIFY.
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	events := lire(t, w)
	if len(events) != 1 {
		t.Fatalf("%d événements, attendu 1 : %+v", len(events), events)
	}
	if events[0].Path != depose {
		t.Fatalf("chemin %q, attendu %q", events[0].Path, depose)
	}
	if events[0].Mask&unix.IN_CLOSE_WRITE == 0 {
		t.Fatalf("masque %#x, IN_CLOSE_WRITE attendu", events[0].Mask)
	}
}

func TestWatcherMovedTo(t *testing.T) {
	depot := t.TempDir()
	source := filepath.Join(t.TempDir(), "base.zip")
	if err := os.WriteFile(source, []byte("PK"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(depot, unix.IN_CLOSE_WRITE|unix.IN_MOVED_TO)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Le renommage doit venir du meme systeme de fichiers pour produire un
	// IN_MOVED_TO ; les deux TempDir partagent /tmp.
	cible := filepath.Join(depot, "base.zip")
	if err := os.Rename(source, cible); err != nil {
		t.Skipf("renommage impossible (systèmes de fichiers différents) : %v", err)
	}

	events := lire(t, w)
	if len(events) != 1 || events[0].Path != cible {
		t.Fatalf("événements %+v, attendu un IN_MOVED_TO sur %s", events, cible)
	}
	if events[0].Mask&unix.IN_MOVED_TO == 0 {
		t.Fatalf("masque %#x, IN_MOVED_TO attendu", events[0].Mask)
	}
}

// Close doit debloquer un Read en cours, sinon l'arret du service serait
// impossible sans tuer le processus.
func TestWatcherCloseDebloqueRead(t *testing.T) {
	w, err := NewWatcher(t.TempDir(), unix.IN_CLOSE_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	erreurs := make(chan error, 1)
	go func() {
		_, err := w.Read()
		erreurs <- err
	}()

	time.Sleep(50 * time.Millisecond) // laisser le Read s'installer
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-erreurs:
		if !errors.Is(err, os.ErrClosed) {
			t.Fatalf("erreur %v, os.ErrClosed attendu", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Read toujours bloqué après Close")
	}
}

func TestNewWatcherRepertoireInexistant(t *testing.T) {
	if _, err := NewWatcher(filepath.Join(t.TempDir(), "absent"), unix.IN_CLOSE_WRITE); err == nil {
		t.Fatal("répertoire inexistant accepté")
	}
}
