// Package config lit et enregistre les réglages d'Aria Vault.
// Les identifiants du fournisseur DNS (et la phrase des sauvegardes chiffrées)
// sont chiffrés avec DPAPI : seul le compte Windows qui les a enregistrés peut
// les relire.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aria-vault/internal/dnsprov"
)

// Config est le contenu de config.json.
type Config struct {
	Domaine              string `json:"domaine"`
	Fournisseur          string `json:"fournisseur,omitempty"`  // vide = DuckDNS (réglages d'avant la v0.6)
	IdentifiantsChiffres []byte `json:"identifiants,omitempty"` // DPAPI d'un objet JSON, encodé en base64
	JetonChiffre         []byte `json:"jetonDuckDNS,omitempty"` // v0.5 et avant : jeton DuckDNS seul (DPAPI)
	Email                string `json:"email,omitempty"`        // facultatif : avertissements Let's Encrypt
	Inscriptions         bool   `json:"inscriptions,omitempty"` // création de comptes autorisée

	// IP publiée automatiquement (activé par défaut)
	SansIPAuto bool `json:"sansIPAuto,omitempty"`

	// Sauvegardes du coffre
	DossierSauvegarde    string    `json:"dossierSauvegarde,omitempty"`  // vide = pas encore choisi : aucune sauvegarde
	SansSauvegardeAuto   bool      `json:"sansSauvegardeAuto,omitempty"` // la quotidienne est active par défaut
	DerniereSauvegarde   time.Time `json:"derniereSauvegarde,omitzero"`
	SauvegardesChiffrees bool      `json:"sauvegardesChiffrees,omitempty"`
	PhraseChiffree       []byte    `json:"phraseSauvegardes,omitempty"` // DPAPI

	// Services de Vaultwarden (v0.8)
	Push        *Push `json:"push,omitempty"` // synchronisation instantanée des téléphones
	SMTP        *SMTP `json:"smtp,omitempty"` // e-mails
	Invitations bool  `json:"invitations,omitempty"`

	// Divers
	SansVerifMaj   bool `json:"sansVerifMaj,omitempty"`   // vérification quotidienne des nouvelles versions
	EmpecherVeille bool `json:"empecherVeille,omitempty"` // le PC ne se met pas en veille tant que le serveur tourne
}

// Load lit config.json. Un fichier absent donne une configuration vide.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config.json illisible : %w", err)
	}
	return &c, nil
}

// Save enregistre config.json (écriture atomique).
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// FournisseurDNS renvoie le fournisseur choisi (DuckDNS pour les anciens réglages).
func (c *Config) FournisseurDNS() string {
	if c.Fournisseur == "" {
		return dnsprov.DuckDNS
	}
	return c.Fournisseur
}

// AIdentifiants indique si des identifiants DNS sont enregistrés.
func (c *Config) AIdentifiants() bool {
	return len(c.IdentifiantsChiffres) > 0 || len(c.JetonChiffre) > 0
}

// SetIdentifiants valide, chiffre et mémorise le fournisseur et ses identifiants.
// Les erreurs sont renvoyées par champ.
func (c *Config) SetIdentifiants(fournisseur string, ids map[string]string) map[string]string {
	propres, erreurs := dnsprov.Valider(fournisseur, ids)
	if len(erreurs) > 0 {
		return erreurs
	}
	data, err := json.Marshal(propres)
	if err != nil {
		return map[string]string{"fournisseur": err.Error()}
	}
	enc, err := protect(data)
	if err != nil {
		return map[string]string{"fournisseur": "chiffrement des identifiants : " + err.Error()}
	}
	c.Fournisseur, c.IdentifiantsChiffres, c.JetonChiffre = fournisseur, enc, nil
	return nil
}

// Identifiants déchiffre les identifiants du fournisseur DNS.
func (c *Config) Identifiants() (map[string]string, error) {
	const autre = " (enregistrés par un autre compte Windows ?)"
	switch {
	case len(c.IdentifiantsChiffres) > 0:
		plain, err := unprotect(c.IdentifiantsChiffres)
		if err != nil {
			return nil, fmt.Errorf("identifiants DNS indéchiffrables%s : %w", autre, err)
		}
		ids := map[string]string{}
		if err := json.Unmarshal(plain, &ids); err != nil {
			return nil, errors.New("identifiants DNS illisibles")
		}
		return ids, nil
	case len(c.JetonChiffre) > 0:
		plain, err := unprotect(c.JetonChiffre)
		if err != nil {
			return nil, fmt.Errorf("jeton DuckDNS indéchiffrable%s : %w", autre, err)
		}
		return map[string]string{"jeton": string(plain)}, nil
	}
	return nil, errors.New("aucun identifiant DNS enregistré")
}

// SetPhraseSauvegarde mémorise (chiffrée avec DPAPI) la phrase des sauvegardes
// chiffrées ; une phrase vide l'efface.
func (c *Config) SetPhraseSauvegarde(phrase string) error {
	if phrase == "" {
		c.PhraseChiffree = nil
		return nil
	}
	enc, err := protect([]byte(phrase))
	if err != nil {
		return fmt.Errorf("chiffrement de la phrase : %w", err)
	}
	c.PhraseChiffree = enc
	return nil
}

// PhraseSauvegarde renvoie la phrase des sauvegardes chiffrées ("" si aucune).
func (c *Config) PhraseSauvegarde() (string, error) {
	if len(c.PhraseChiffree) == 0 {
		return "", nil
	}
	plain, err := unprotect(c.PhraseChiffree)
	if err != nil {
		return "", fmt.Errorf("phrase des sauvegardes indéchiffrable (enregistrée par un autre compte Windows ?) : %w", err)
	}
	return string(plain), nil
}

// Valider vérifie que tout ce qu'il faut pour démarrer est présent.
func (c *Config) Valider() error {
	if _, err := dnsprov.NormaliserDomaine(c.FournisseurDNS(), c.Domaine); err != nil {
		return err
	}
	ids, err := c.Identifiants()
	if err != nil {
		return err
	}
	if _, erreurs := dnsprov.Valider(c.FournisseurDNS(), ids); len(erreurs) > 0 {
		return errors.New("identifiants DNS incomplets : complète la page HTTPS")
	}
	return nil
}
