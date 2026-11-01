// Package courriel envoie l'e-mail de test des réglages : il vérifie, avec les
// mêmes paramètres que Vaultwarden, que le serveur SMTP accepte la connexion,
// l'identification et l'envoi.
package courriel

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Serveur décrit le serveur d'envoi.
type Serveur struct {
	Hote, Securite, Utilisateur, MotDePasse, De string
	Port                                        int
	// pour les tests : configuration TLS (nil = vérification normale)
	TLS *tls.Config
}

// Tester envoie un e-mail de test à destinataire.
func Tester(ctx context.Context, s Serveur, destinataire string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	adresse := net.JoinHostPort(s.Hote, strconv.Itoa(s.Port))
	tlsCfg := s.TLS
	if tlsCfg == nil {
		tlsCfg = &tls.Config{ServerName: s.Hote, MinVersion: tls.VersionTLS12}
	}

	var d net.Dialer
	var conn net.Conn
	var err error
	if s.Securite == "force_tls" {
		conn, err = (&tls.Dialer{NetDialer: &d, Config: tlsCfg}).DialContext(ctx, "tcp", adresse)
	} else {
		conn, err = d.DialContext(ctx, "tcp", adresse)
	}
	if err != nil {
		return fmt.Errorf("connexion à %s impossible : %w", adresse, err)
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, s.Hote)
	if err != nil {
		conn.Close()
		return fmt.Errorf("le serveur ne répond pas comme un serveur SMTP : %w", err)
	}
	defer c.Close()
	if err := c.Hello("aria-vault.local"); err != nil {
		return err
	}
	if s.Securite == "starttls" {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("le serveur ne propose pas STARTTLS : essaie la sécurité « TLS » (port 465)")
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("STARTTLS refusé : %w", err)
		}
	}
	if s.Utilisateur != "" {
		if err := c.Auth(choisirAuth(c, s)); err != nil {
			return fmt.Errorf("identification refusée (identifiant ou mot de passe) : %w", err)
		}
	}
	if err := c.Mail(s.De); err != nil {
		return fmt.Errorf("expéditeur refusé : %w", err)
	}
	if err := c.Rcpt(destinataire); err != nil {
		return fmt.Errorf("destinataire refusé : %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(Message(s.De, destinataire, time.Now())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("envoi refusé : %w", err)
	}
	return c.Quit()
}

// choisirAuth prend PLAIN si le serveur le propose, sinon LOGIN (Outlook,
// certains hébergeurs).
func choisirAuth(c *smtp.Client, s Serveur) smtp.Auth {
	_, params := c.Extension("AUTH")
	mecas := strings.Fields(strings.ToUpper(params))
	for _, m := range mecas {
		if m == "PLAIN" {
			return plainAuth{s.Utilisateur, s.MotDePasse}
		}
	}
	for _, m := range mecas {
		if m == "LOGIN" {
			return &loginAuth{u: s.Utilisateur, p: s.MotDePasse}
		}
	}
	return plainAuth{s.Utilisateur, s.MotDePasse}
}

// plainAuth : comme smtp.PlainAuth, mais la connexion a déjà été chiffrée
// (STARTTLS ou TLS) avant, ce que vérifie SetSMTP côté réglages.
type plainAuth struct{ u, p string }

func (a plainAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.u + "\x00" + a.p), nil
}
func (a plainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("réponse inattendue du serveur")
	}
	return nil, nil
}

type loginAuth struct {
	u, p string
	n    int
}

func (a *loginAuth) Start(*smtp.ServerInfo) (string, []byte, error) { return "LOGIN", nil, nil }
func (a *loginAuth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	a.n++
	switch a.n {
	case 1:
		return []byte(a.u), nil
	case 2:
		return []byte(a.p), nil
	}
	return nil, errors.New("réponse inattendue du serveur")
}

// Message construit l'e-mail de test (texte, UTF-8).
func Message(de, a string, quand time.Time) []byte {
	id := make([]byte, 12)
	_, _ = rand.Read(id)
	domaine := de[strings.LastIndex(de, "@")+1:]
	entetes := []string{
		"From: " + mime.QEncoding.Encode("utf-8", "Aria Vault") + " <" + de + ">",
		"To: <" + a + ">",
		"Subject: " + mime.QEncoding.Encode("utf-8", "Aria Vault : e-mail de test ✓"),
		"Date: " + quand.Format(time.RFC1123Z),
		"Message-ID: <" + hex.EncodeToString(id) + "@" + domaine + ">",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
	}
	corps := "Bonjour,\r\n\r\nCet e-mail confirme que ton coffre Aria Vault peut envoyer des e-mails :\r\n" +
		"alertes de connexion, double authentification par e-mail, invitations.\r\n\r\n" +
		"Aucune action n'est nécessaire.\r\n\r\n-- \r\nAria Vault\r\n"
	return []byte(strings.Join(entetes, "\r\n") + "\r\n\r\n" + corps)
}
