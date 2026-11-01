// Package maj regarde, une fois par jour, si une nouvelle version d'Aria Vault
// a été publiée sur GitHub, et l'installe quand l'utilisateur le demande
// (installer.go). Rien n'est jamais téléchargé ni installé sans son clic.
package maj

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"aria-vault/internal/releve"
)

const (
	Depot      = "ZedjeK/Aria-Vault"
	api        = "https://api.github.com/repos/" + Depot + "/releases/latest"
	pagePrefix = "https://github.com/" + Depot + "/releases/"
	// Le zip ne peut venir que des releases du dépôt officiel.
	telechargementPrefix = pagePrefix + "download/"
	Intervalle           = 24 * time.Hour
)

// Info décrit la dernière version publiée.
type Info struct {
	Version    string    `json:"version"`    // « 0.6.1 »
	URL        string    `json:"url"`        // page de la release
	Disponible bool      `json:"disponible"` // plus récente que la version lancée
	Verifie    time.Time `json:"verifie"`
	Erreur     string    `json:"erreur"`

	// Installation en un clic : zip de la release et empreintes publiées.
	Installable bool   `json:"installable"`
	Taille      int64  `json:"taille"`
	Zip         string `json:"-"`
	ZipSHA      string `json:"-"`
	ExeSHA      string `json:"-"`
}

type release struct {
	TagName    string  `json:"tag_name"`
	HTMLURL    string  `json:"html_url"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Body       string  `json:"body"`
	Assets     []asset `json:"assets"`
}

type asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"` // « sha256:… », calculé par GitHub
	Size   int64  `json:"size"`
}

var shaRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// empreinte cherche dans les notes la ligne « <sha256>  <nom> ».
func empreinte(notes, nom string) string {
	re := regexp.MustCompile(`(?m)^([0-9a-f]{64})\s+` + regexp.QuoteMeta(nom) + `(\s|$)`)
	if m := re.FindStringSubmatch(notes); m != nil {
		return m[1]
	}
	return ""
}

// Lire interprète la réponse de l'API GitHub. Seule une page de release du
// dépôt officiel est acceptée comme lien.
func Lire(body []byte, actuelle string) (Info, error) {
	var r release
	if err := json.Unmarshal(body, &r); err != nil {
		return Info{}, errors.New("réponse de GitHub illisible")
	}
	if r.Draft || r.Prerelease {
		return Info{}, errors.New("la dernière release n'est pas une version publiée")
	}
	v := strings.TrimPrefix(r.TagName, "v")
	if !releve.PlusRecente(v, "0.0.0") {
		return Info{}, fmt.Errorf("numéro de version inattendu : %q", r.TagName)
	}
	if !strings.HasPrefix(r.HTMLURL, pagePrefix) || strings.ContainsAny(r.HTMLURL, " \"'<>") {
		return Info{}, errors.New("lien de release inattendu")
	}
	info := Info{Version: v, URL: r.HTMLURL, Disponible: releve.PlusRecente(v, actuelle)}

	// Installable seulement si le zip et les deux empreintes sont là et concordent.
	nomZip := "Aria-Vault-" + v + ".zip"
	for _, a := range r.Assets {
		if a.Name != nomZip || !strings.HasPrefix(a.URL, telechargementPrefix+"v"+v+"/") {
			continue
		}
		zipNotes := empreinte(r.Body, nomZip)
		zipGitHub := strings.TrimPrefix(a.Digest, "sha256:")
		zip := zipNotes
		if zip == "" {
			zip = zipGitHub
		}
		if zipGitHub != "" && zipNotes != "" && zipGitHub != zipNotes {
			break // empreintes discordantes : on n'installe pas
		}
		exe := empreinte(r.Body, "Aria-Vault.exe")
		if shaRe.MatchString(zip) && shaRe.MatchString(exe) && a.Size > 0 && a.Size <= TailleMax {
			info.Installable, info.Zip, info.ZipSHA, info.ExeSHA, info.Taille = true, a.URL, zip, exe, a.Size
		}
	}
	return info, nil
}

// Verificateur garde le dernier résultat.
type Verificateur struct {
	mu      sync.Mutex
	info    Info
	actuel  string
	client  *http.Client
	adresse string
}

// Nouveau prépare un vérificateur pour la version lancée.
func Nouveau(actuelle string) *Verificateur {
	return &Verificateur{actuel: actuelle, client: &http.Client{Timeout: 20 * time.Second}, adresse: api}
}

// Info renvoie le dernier résultat.
func (v *Verificateur) Info() Info {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.info
}

// Oublier efface le résultat (vérification désactivée).
func (v *Verificateur) Oublier() {
	v.mu.Lock()
	v.info = Info{}
	v.mu.Unlock()
}

// Verifier interroge GitHub.
func (v *Verificateur) Verifier(ctx context.Context) (Info, error) {
	info, err := v.interroger(ctx)
	info.Verifie = time.Now()
	if err != nil {
		v.mu.Lock()
		v.info.Erreur, v.info.Verifie = err.Error(), info.Verifie
		info = v.info
		v.mu.Unlock()
		return info, err
	}
	v.mu.Lock()
	v.info = info
	v.mu.Unlock()
	return info, nil
}

func (v *Verificateur) interroger(ctx context.Context) (Info, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.adresse, nil)
	if err != nil {
		return Info{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Aria-Vault/"+v.actuel)
	resp, err := v.client.Do(req)
	if err != nil {
		return Info{}, fmt.Errorf("GitHub injoignable : %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Info{}, errors.New("aucune version publiée")
	}
	if resp.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("GitHub a répondu %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Info{}, err
	}
	return Lire(body, v.actuel)
}
