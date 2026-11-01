package config

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

// Push : notifications push par le relais de Bitwarden (bitwarden.com/host).
type Push struct {
	Region      string `json:"region"` // "us" ou "eu"
	ID          string `json:"id"`     // Installation ID
	CleChiffree []byte `json:"cle"`    // Installation Key, DPAPI
}

// SMTP : serveur d'envoi des e-mails de Vaultwarden.
type SMTP struct {
	Hote              string `json:"hote"`
	Port              int    `json:"port"`
	Securite          string `json:"securite"` // starttls, force_tls ou off
	Utilisateur       string `json:"utilisateur,omitempty"`
	MotDePasseChiffre []byte `json:"motDePasse,omitempty"` // DPAPI
	De                string `json:"de"`                   // adresse d'expédition
}

// Régions du relais push et leurs adresses (vides = valeurs par défaut de Vaultwarden, US).
var RegionsPush = map[string][2]string{
	"us": {"", ""},
	"eu": {"https://api.bitwarden.eu", "https://identity.bitwarden.eu"},
}

// Ports par défaut de chaque sécurité (ceux de Vaultwarden).
var PortsSMTP = map[string]int{"starttls": 587, "force_tls": 465, "off": 25}

var (
	uuidRe  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	cleRe   = regexp.MustCompile(`^[!-~]{8,200}$`)
	hoteRe  = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
	espaces = strings.NewReplacer(" ", "", " ", "")
)

// SetPush valide et enregistre la configuration push. cle vide = garder
// l'actuelle. Les erreurs sont renvoyées par champ.
func (c *Config) SetPush(region, id, cle string) map[string]string {
	errs := map[string]string{}
	region = strings.ToLower(strings.TrimSpace(region))
	if _, ok := RegionsPush[region]; !ok {
		errs["region"] = "choisis la région des données : États-Unis ou Union européenne"
	}
	id = strings.ToLower(strings.TrimSpace(id))
	if !uuidRe.MatchString(id) {
		errs["id"] = "l'Installation ID ressemble à xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	}
	cle = strings.TrimSpace(cle)
	var enc []byte
	switch {
	case cle == "" && c.Push != nil && len(c.Push.CleChiffree) > 0:
		enc = c.Push.CleChiffree
	case cle == "":
		errs["cle"] = "l'Installation Key est obligatoire"
	case !cleRe.MatchString(cle):
		errs["cle"] = "Installation Key invalide (vérifie le copier-coller, sans espace)"
	default:
		var err error
		if enc, err = protect([]byte(cle)); err != nil {
			errs["cle"] = "chiffrement de la clé : " + err.Error()
		}
	}
	if len(errs) > 0 {
		return errs
	}
	c.Push = &Push{Region: region, ID: id, CleChiffree: enc}
	return nil
}

// PushCle déchiffre l'Installation Key.
func (c *Config) PushCle() (string, error) {
	if c.Push == nil || len(c.Push.CleChiffree) == 0 {
		return "", errors.New("synchronisation instantanée non configurée")
	}
	plain, err := unprotect(c.Push.CleChiffree)
	if err != nil {
		return "", fmt.Errorf("clé push indéchiffrable (enregistrée par un autre compte Windows ?) : %w", err)
	}
	return string(plain), nil
}

// SMTPEntree est le formulaire des e-mails.
type SMTPEntree struct {
	Hote        string `json:"hote"`
	Port        int    `json:"port"` // 0 = celui de la sécurité choisie
	Securite    string `json:"securite"`
	Utilisateur string `json:"utilisateur"`
	MotDePasse  string `json:"motDePasse"` // vide = garder l'actuel
	De          string `json:"de"`
}

// SetSMTP valide et enregistre le serveur d'e-mails.
func (c *Config) SetSMTP(in SMTPEntree) map[string]string {
	errs := map[string]string{}
	s := SMTP{
		Hote:        strings.ToLower(strings.TrimSpace(in.Hote)),
		Port:        in.Port,
		Securite:    strings.TrimSpace(in.Securite),
		Utilisateur: strings.TrimSpace(in.Utilisateur),
		De:          strings.TrimSpace(in.De),
	}
	if !hoteRe.MatchString(s.Hote) {
		errs["hote"] = "nom de serveur invalide (exemple : smtp.gmail.com)"
	}
	def, ok := PortsSMTP[s.Securite]
	if !ok {
		errs["securite"] = "choisis STARTTLS, TLS ou aucune"
	}
	if s.Port == 0 {
		s.Port = def
	}
	if s.Port < 1 || s.Port > 65535 {
		errs["port"] = "port invalide"
	}
	if a, err := mail.ParseAddress(s.De); err != nil || a.Address != s.De {
		errs["de"] = "adresse d'expédition invalide (exemple : moi@gmail.com)"
	}

	mdp := in.MotDePasse
	if s.Hote == "smtp.gmail.com" {
		mdp = espaces.Replace(mdp) // Google affiche ses mots de passe d'application par blocs de 4
	}
	switch {
	case s.Utilisateur == "":
		// envoi sans authentification (relais local) : pas de mot de passe
	case s.Securite == "off":
		errs["securite"] = "sans chiffrement, le mot de passe circulerait en clair : choisis STARTTLS ou TLS"
	case mdp == "" && c.SMTP != nil && len(c.SMTP.MotDePasseChiffre) > 0 && c.SMTP.Utilisateur == s.Utilisateur && c.SMTP.Hote == s.Hote:
		s.MotDePasseChiffre = c.SMTP.MotDePasseChiffre
	case mdp == "":
		errs["motDePasse"] = "le mot de passe est obligatoire avec un identifiant"
	default:
		enc, err := protect([]byte(mdp))
		if err != nil {
			errs["motDePasse"] = "chiffrement du mot de passe : " + err.Error()
		}
		s.MotDePasseChiffre = enc
	}
	if len(errs) > 0 {
		return errs
	}
	c.SMTP = &s
	return nil
}

// SMTPMotDePasse déchiffre le mot de passe du serveur d'e-mails ("" si aucun).
func (c *Config) SMTPMotDePasse() (string, error) {
	if c.SMTP == nil || len(c.SMTP.MotDePasseChiffre) == 0 {
		return "", nil
	}
	plain, err := unprotect(c.SMTP.MotDePasseChiffre)
	if err != nil {
		return "", fmt.Errorf("mot de passe e-mail indéchiffrable (enregistré par un autre compte Windows ?) : %w", err)
	}
	return string(plain), nil
}
