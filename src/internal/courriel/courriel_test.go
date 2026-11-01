package courriel

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fauxServeur : un serveur SMTP minimal, sans TLS, qui exige AUTH PLAIN.
func fauxServeur(t *testing.T, recu chan<- string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		dire := func(s string) { conn.Write([]byte(s + "\r\n")) }
		dire("220 faux ESMTP")
		var donnees strings.Builder
		auth := false
		for {
			ligne, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(ligne))
			switch {
			case strings.HasPrefix(cmd, "EHLO"):
				dire("250-faux")
				dire("250 AUTH PLAIN LOGIN")
			case strings.HasPrefix(cmd, "AUTH PLAIN "):
				b, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(ligne)[11:])
				if string(b) == "\x00moi\x00secret" {
					auth = true
					dire("235 ok")
				} else {
					dire("535 refusé")
				}
			case strings.HasPrefix(cmd, "MAIL FROM"):
				if !auth {
					dire("530 authentification requise")
					continue
				}
				dire("250 ok")
			case strings.HasPrefix(cmd, "RCPT TO"):
				dire("250 ok")
			case cmd == "DATA":
				dire("354 vas-y")
				for {
					l, _ := r.ReadString('\n')
					if l == ".\r\n" {
						break
					}
					donnees.WriteString(l)
				}
				dire("250 envoyé")
				recu <- donnees.String()
			case cmd == "QUIT":
				dire("221 au revoir")
				return
			default:
				dire("250 ok")
			}
		}
	}()
	return ln.Addr().String()
}

func TestTester(t *testing.T) {
	recu := make(chan string, 1)
	addr := fauxServeur(t, recu)
	hote, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	s := Serveur{Hote: hote, Port: p, Securite: "off", Utilisateur: "moi", MotDePasse: "secret", De: "moi@exemple.fr"}
	if err := Tester(context.Background(), s, "moi@exemple.fr"); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-recu:
		if !strings.Contains(m, "Subject: =?utf-8?q?Aria_Vault") || !strings.Contains(m, "Content-Type: text/plain; charset=utf-8") {
			t.Errorf("message :\n%s", m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("rien reçu")
	}

	// Mauvais mot de passe : erreur claire.
	addr2 := fauxServeur(t, make(chan string, 1))
	_, port2, _ := net.SplitHostPort(addr2)
	p2, _ := strconv.Atoi(port2)
	s.Port, s.MotDePasse = p2, "faux"
	if err := Tester(context.Background(), s, "moi@exemple.fr"); err == nil || !strings.Contains(err.Error(), "identification refusée") {
		t.Errorf("mauvais mot de passe : %v", err)
	}

	// STARTTLS absent : conseil de passer en TLS.
	addr3 := fauxServeur(t, make(chan string, 1))
	_, port3, _ := net.SplitHostPort(addr3)
	p3, _ := strconv.Atoi(port3)
	s.Port, s.Securite = p3, "starttls"
	if err := Tester(context.Background(), s, "moi@exemple.fr"); err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Errorf("sans STARTTLS : %v", err)
	}
}
