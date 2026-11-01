package main

import (
	"context"
	"errors"

	"aria-vault/internal/config"
	"aria-vault/internal/courriel"
	"aria-vault/internal/journal"
)

// Synchronisation instantanée (push) et e-mails (SMTP) de Vaultwarden.

// PushView est l'état affiché de la synchronisation instantanée.
type PushView struct {
	Actif  bool   `json:"actif"`
	Region string `json:"region"`
	ID     string `json:"id"`
}

// SMTPView est l'état affiché des e-mails (sans le mot de passe).
type SMTPView struct {
	Actif       bool   `json:"actif"`
	Hote        string `json:"hote"`
	Port        int    `json:"port"`
	Securite    string `json:"securite"`
	Utilisateur string `json:"utilisateur"`
	MotDePasse  bool   `json:"motDePasse"` // un mot de passe est enregistré
	De          string `json:"de"`
}

func vuePush(c *config.Config) PushView {
	if c.Push == nil {
		return PushView{}
	}
	return PushView{Actif: true, Region: c.Push.Region, ID: c.Push.ID}
}

func vueSMTP(c *config.Config) SMTPView {
	if c.SMTP == nil {
		return SMTPView{}
	}
	s := c.SMTP
	return SMTPView{Actif: true, Hote: s.Hote, Port: s.Port, Securite: s.Securite, Utilisateur: s.Utilisateur, MotDePasse: len(s.MotDePasseChiffre) > 0, De: s.De}
}

// PushInput est le formulaire de la synchronisation instantanée.
type PushInput struct {
	Region string `json:"region"`
	ID     string `json:"id"`
	Cle    string `json:"cle"` // vide = garder l'actuelle
}

// SavePush active la synchronisation instantanée et relance le serveur.
func (s *VaultService) SavePush(in PushInput) SaveResult {
	return s.enregistrerService(func(c *config.Config) map[string]string {
		return c.SetPush(in.Region, in.ID, in.Cle)
	}, "Synchronisation instantanée activée : déconnecte puis reconnecte l'appli Bitwarden sur ton téléphone")
}

// DisablePush désactive la synchronisation instantanée.
func (s *VaultService) DisablePush() error {
	if err := s.majConfig(func(c *config.Config) { c.Push = nil }); err != nil {
		return err
	}
	s.j.Add(journal.Info, "Synchronisation instantanée désactivée")
	return s.relancerSiActif()
}

// SaveSMTP enregistre le serveur d'e-mails et relance le serveur.
func (s *VaultService) SaveSMTP(in config.SMTPEntree) SaveResult {
	return s.enregistrerService(func(c *config.Config) map[string]string {
		return c.SetSMTP(in)
	}, "E-mails activés (mot de passe chiffré pour ton compte Windows)")
}

// DisableSMTP désactive les e-mails.
func (s *VaultService) DisableSMTP() error {
	if err := s.majConfig(func(c *config.Config) { c.SMTP = nil }); err != nil {
		return err
	}
	s.j.Add(journal.Info, "E-mails désactivés")
	return s.relancerSiActif()
}

// TestSMTP envoie un e-mail de test avec les réglages enregistrés, à
// l'adresse donnée (vide = l'adresse d'expédition).
func (s *VaultService) TestSMTP(destinataire string) error {
	cfg, err := s.lireConfig()
	if err != nil {
		return err
	}
	if cfg.SMTP == nil {
		return errors.New("enregistre d'abord le serveur d'e-mails")
	}
	mdp, err := cfg.SMTPMotDePasse()
	if err != nil {
		return err
	}
	if destinataire == "" {
		destinataire = cfg.SMTP.De
	}
	m := cfg.SMTP
	err = courriel.Tester(context.Background(), courriel.Serveur{
		Hote: m.Hote, Port: m.Port, Securite: m.Securite, Utilisateur: m.Utilisateur, MotDePasse: mdp, De: m.De,
	}, destinataire)
	if err != nil {
		s.j.Add(journal.Erreur, "E-mail de test : %v", err)
		return err
	}
	s.j.Add(journal.OK, "E-mail de test envoyé à %s", destinataire)
	return nil
}

// SetInvitations autorise ou non les invitations (organisations, accès d'urgence).
func (s *VaultService) SetInvitations(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.Invitations = on }); err != nil {
		return err
	}
	if on {
		s.j.Add(journal.Info, "Invitations autorisées")
	} else {
		s.j.Add(journal.Info, "Invitations refusées")
	}
	return s.relancerSiActif()
}

// enregistrerService applique fn à config.json ; rien n'est écrit en cas d'erreur.
func (s *VaultService) enregistrerService(fn func(*config.Config) map[string]string, ok string) SaveResult {
	res := SaveResult{Fields: map[string]string{}}
	var champs map[string]string
	err := s.majConfigSi(func(c *config.Config) bool {
		champs = fn(c)
		return len(champs) == 0
	})
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if len(champs) > 0 {
		res.Fields, res.Error = champs, "vérifie les champs signalés"
		return res
	}
	s.j.Add(journal.OK, "%s", ok)
	if err := s.relancerSiActif(); err != nil {
		res.Error = err.Error()
	}
	return res
}

// majConfigSi n'enregistre config.json que si fn renvoie true.
func (s *VaultService) majConfigSi(fn func(*config.Config) bool) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	cfg, err := config.Load(s.p.Config)
	if err != nil {
		return err
	}
	if !fn(cfg) {
		return nil
	}
	return cfg.Save(s.p.Config)
}
