package watcher

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Event est un evenement inotify decode, avec le chemin absolu du fichier.
type Event struct {
	Path string
	Mask uint32
}

// Watcher encapsule un descripteur inotify sur un repertoire.
//
// Le descripteur est ouvert en IN_NONBLOCK puis confie a un *os.File : le
// runtime Go l'enregistre alors dans son poller, ce qui rend Read()
// interruptible par Close() depuis une autre goroutine. Un read bloquant, lui,
// ne serait pas interrompu par un signal (le runtime relance l'appel), et
// l'arret propre du service serait impossible.
type Watcher struct {
	file *os.File
	dir  string
	buf  []byte
}

// NewWatcher pose une surveillance sur dir pour les evenements de mask.
// inotify n'est pas recursif : seul dir lui-meme est surveille.
func NewWatcher(dir string, mask uint32) (*Watcher, error) {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, fmt.Errorf("inotify_init1 : %w", err)
	}
	if _, err := unix.InotifyAddWatch(fd, dir, mask); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("inotify_add_watch %s : %w", dir, err)
	}
	return &Watcher{
		file: os.NewFile(uintptr(fd), "inotify"),
		dir:  dir,
		// Un evenement peut porter un nom de NAME_MAX octets ; ce tampon en
		// absorbe une bonne dizaine par lecture.
		buf: make([]byte, 16*(unix.SizeofInotifyEvent+unix.NAME_MAX+1)),
	}, nil
}

// Read bloque jusqu'au prochain lot d'evenements et le decode.
// Renvoie os.ErrClosed apres un appel a Close.
func (w *Watcher) Read() ([]Event, error) {
	n, err := w.file.Read(w.buf)
	if err != nil {
		return nil, err
	}

	var events []Event
	for offset := 0; offset+unix.SizeofInotifyEvent <= n; {
		// Le noyau ecrit une suite de struct inotify_event de taille variable :
		// l'en-tete de taille fixe est suivie de Len octets de nom.
		raw := (*unix.InotifyEvent)(unsafe.Pointer(&w.buf[offset]))
		next := offset + unix.SizeofInotifyEvent + int(raw.Len)
		if next > n {
			return nil, fmt.Errorf("evenement inotify tronque (%d octets attendus, %d lus)", next, n)
		}

		if raw.Mask&unix.IN_Q_OVERFLOW != 0 {
			// La file du noyau a deborde : des depots ont ete perdus.
			// Cf. /proc/sys/fs/inotify/max_queued_events.
			return nil, fmt.Errorf("file inotify saturee : des evenements ont ete perdus")
		}

		if raw.Len > 0 {
			name := w.buf[offset+unix.SizeofInotifyEvent : next]
			// Le nom est complete par des octets nuls jusqu'a un alignement.
			if i := bytes.IndexByte(name, 0); i >= 0 {
				name = name[:i]
			}
			events = append(events, Event{
				Path: filepath.Join(w.dir, string(name)),
				Mask: raw.Mask,
			})
		}
		offset = next
	}
	return events, nil
}

// Close libere le descripteur et debloque un Read en cours.
func (w *Watcher) Close() error {
	return w.file.Close()
}
