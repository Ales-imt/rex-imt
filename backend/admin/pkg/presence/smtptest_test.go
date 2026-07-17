package presence

// Mini-serveur SMTP local pour les tests du témoin : accepte une session
// basique (EHLO/MAIL/RCPT/DATA/QUIT), sans TLS ni auth, et capture les
// messages reçus. Aucun trafic ne sort de 127.0.0.1.

import (
	"back-rex-common/pkg/services"
	"bufio"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

type smtpTestServer struct {
	addr net.Addr
	mu   sync.Mutex
	msgs []string
}

// startSMTPServer démarre le serveur sur un port éphémère et l'arrête à la
// fin du test.
func startSMTPServer(t *testing.T) *smtpTestServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("smtp test listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	srv := &smtpTestServer{addr: ln.Addr()}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener fermé
			}
			go srv.handle(conn)
		}
	}()
	return srv
}

// config retourne une SMTPConfig pointant sur ce serveur (ni TLS ni auth).
func (s *smtpTestServer) config() services.SMTPConfig {
	tcp := s.addr.(*net.TCPAddr)
	return services.SMTPConfig{
		Host:    tcp.IP.String(),
		Port:    tcp.Port,
		From:    "noreply@exemple.fr",
		Timeout: 5 * time.Second,
	}
}

func (s *smtpTestServer) messages() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.msgs...)
}

func (s *smtpTestServer) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	write := func(line string) { conn.Write([]byte(line + "\r\n")) }

	write("220 smtp-test ready")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			write("250-smtp-test")
			write("250 OK")
		case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
			write("250 OK")
		case strings.HasPrefix(cmd, "DATA"):
			write("354 end with .")
			var body strings.Builder
			for {
				dataLine, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
				body.WriteString(dataLine)
			}
			s.mu.Lock()
			s.msgs = append(s.msgs, body.String())
			s.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(cmd, "QUIT"):
			write("221 bye")
			return
		default:
			write("250 OK")
		}
	}
}
