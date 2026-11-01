package config

import (
	"encoding/json"
	"errors"
	"strings"

	"aria-vault/internal/chiffre"
	"aria-vault/internal/dnsprov"
)

// Fichier de réglages exportable (.ariavault), entièrement chiffré par une
// phrase de passe (scrypt + AES-256-GCM, voir le paquet chiffre) : rien n'y
// est lisible sans elle, ni le domaine, ni l'e-mail, ni les identifiants DNS.
// À l'import, les identifiants sont rechiffrés avec DPAPI pour le compte
// Windows qui l'importe.

const (
	ExtensionExport = ".ariavault"
	PhraseMin       = chiffre.PhraseMin
)

var magie = []byte("ARIAVLT2")

// ErrPhrase signale une phrase de passe incorrecte (ou un fichier altéré).
var ErrPhrase = chiffre.ErrPhrase

// Importe est le contenu d'un fichier de réglages déchiffré.
type Importe struct {
	Domaine      string            `json:"domaine"`
	Email        string            `json:"email,omitempty"`
	Fournisseur  string            `json:"fournisseur,omitempty"`  // absent avant la v0.6 : DuckDNS
	Identifiants map[string]string `json:"identifiants,omitempty"` // v0.6
	Jeton        string            `json:"jeton,omitempty"`        // DuckDNS, lu par les versions précédentes

	// v0.8 : services de Vaultwarden (absents des fichiers plus anciens)
	Push        *PushExport `json:"push,omitempty"`
	SMTP        *SMTPEntree `json:"smtp,omitempty"`
	Invitations bool        `json:"invitations,omitempty"`
}

// PushExport : la synchronisation instantanée, clé en clair dans le fichier chiffré.
type PushExport struct {
	Region string `json:"region"`
	ID     string `json:"id"`
	Cle    string `json:"cle"`
}

// Exporter produit le fichier de réglages chiffré par la phrase de passe.
func (c *Config) Exporter(phrase string) ([]byte, error) {
	if err := chiffre.VerifierPhrase(phrase); err != nil {
		return nil, err
	}
	if err := c.Valider(); err != nil {
		return nil, err
	}
	ids, err := c.Identifiants()
	if err != nil {
		return nil, err
	}
	imp := Importe{Domaine: c.Domaine, Email: c.Email, Fournisseur: c.FournisseurDNS(), Identifiants: ids}
	if imp.Fournisseur == dnsprov.DuckDNS {
		imp.Jeton = ids["jeton"]
	}
	imp.Invitations = c.Invitations
	if c.Push != nil {
		cle, err := c.PushCle()
		if err != nil {
			return nil, err
		}
		imp.Push = &PushExport{Region: c.Push.Region, ID: c.Push.ID, Cle: cle}
	}
	if m := c.SMTP; m != nil {
		mdp, err := c.SMTPMotDePasse()
		if err != nil {
			return nil, err
		}
		imp.SMTP = &SMTPEntree{Hote: m.Hote, Port: m.Port, Securite: m.Securite, Utilisateur: m.Utilisateur, MotDePasse: mdp, De: m.De}
	}
	clair, err := json.Marshal(imp)
	if err != nil {
		return nil, err
	}
	return chiffre.Sceller(phrase, magie, clair)
}

// LireExport déchiffre un fichier de réglages. La phrase est obligatoire.
func LireExport(data []byte, phrase string) (Importe, error) {
	if phrase == "" {
		return Importe{}, errors.New("la phrase de passe du fichier est obligatoire")
	}
	clair, err := chiffre.Ouvrir(phrase, magie, data)
	if errors.Is(err, chiffre.ErrFormat) {
		return Importe{}, errors.New("ce n'est pas un fichier de réglages Aria Vault (.ariavault)")
	}
	if err != nil {
		return Importe{}, err
	}

	var imp Importe
	if err := json.Unmarshal(clair, &imp); err != nil {
		return Importe{}, errors.New("contenu du fichier illisible")
	}
	if imp.Fournisseur == "" {
		imp.Fournisseur = dnsprov.DuckDNS
		imp.Identifiants = map[string]string{"jeton": imp.Jeton}
	}
	if imp.Domaine, err = dnsprov.NormaliserDomaine(imp.Fournisseur, imp.Domaine); err != nil {
		return Importe{}, err
	}
	imp.Email = strings.TrimSpace(imp.Email)
	propres, erreurs := dnsprov.Valider(imp.Fournisseur, imp.Identifiants)
	if len(erreurs) > 0 {
		return Importe{}, errors.New("identifiants DNS invalides dans le fichier")
	}
	imp.Identifiants = propres
	return imp, nil
}

// Appliquer recopie un fichier importé dans c : tout est validé et rechiffré
// avec DPAPI pour ce compte Windows avant d'être écrit. En cas d'erreur, c
// n'est pas modifié.
func (c *Config) Appliquer(imp Importe) error {
	var n Config
	if errs := n.SetIdentifiants(imp.Fournisseur, imp.Identifiants); len(errs) > 0 {
		return errors.New("identifiants DNS invalides dans le fichier")
	}
	if p := imp.Push; p != nil {
		if errs := n.SetPush(p.Region, p.ID, p.Cle); len(errs) > 0 {
			return errors.New("synchronisation instantanée invalide dans le fichier")
		}
	}
	if m := imp.SMTP; m != nil {
		if errs := n.SetSMTP(*m); len(errs) > 0 {
			return errors.New("réglages d'e-mails invalides dans le fichier")
		}
	}
	c.Domaine, c.Email = imp.Domaine, imp.Email
	c.Fournisseur, c.IdentifiantsChiffres, c.JetonChiffre = n.Fournisseur, n.IdentifiantsChiffres, nil
	// Les services ne sont remplacés que si le fichier les contient (un fichier
	// d'avant la v0.8 ne doit pas effacer ceux déjà configurés ici).
	if imp.Push != nil {
		c.Push = n.Push
	}
	if imp.SMTP != nil {
		c.SMTP = n.SMTP
		c.Invitations = imp.Invitations
	}
	return nil
}
