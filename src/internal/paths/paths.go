// Package paths centralise les emplacements d'Aria Vault.
//
// Depuis la v0.8, ce qui compte (le coffre, les certificats, les réglages, le
// journal) vit dans %USERPROFILE%\.Aria-Vault, et ce qui se recrée tout seul
// (Vaultwarden extrait, cache de l'interface, fichier de relève) reste dans
// %LOCALAPPDATA%\AriaVault. Les deux sont propres au compte Windows : Aria
// tourne avec les droits de l'utilisateur, jamais en administrateur.
//
// Migrations automatiques au lancement :
//   - v0.3 et avant : %LOCALAPPDATA%\AriaVaultwarden → %LOCALAPPDATA%\AriaVault ;
//   - v0.7 et avant : données de %LOCALAPPDATA%\AriaVault → %USERPROFILE%\.Aria-Vault.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Paths regroupe les dossiers et fichiers utilisés par Aria.
type Paths struct {
	Base        string // %USERPROFILE%\.Aria-Vault : données de l'utilisateur
	Cache       string // %LOCALAPPDATA%\AriaVault : ce qui se recrée tout seul
	Moteur      string // vaultwarden.exe et web-vault, réextraits à chaque démarrage
	Donnees     string // le coffre : base SQLite, clés RSA, pièces jointes
	Certificats string // certificat Let's Encrypt et compte ACME
	Config      string // réglages d'Aria (jeton DuckDNS chiffré)
}

const (
	Dossier        = "AriaVault"       // cache, dans %LOCALAPPDATA%
	AncienDossier  = "AriaVaultwarden" // v0.3 et avant
	DossierDonnees = ".Aria-Vault"     // données, dans %USERPROFILE% (v0.8)
)

// Racine renvoie le dossier parent (%LOCALAPPDATA% sous Windows).
func Racine() (string, error) {
	// Sous Windows, UserCacheDir correspond à %LOCALAPPDATA%.
	return os.UserCacheDir()
}

// Migrer renvoie les emplacements d'Aria, en renommant au besoin l'ancien
// dossier AriaVaultwarden en AriaVault. Si le renommage échoue (un fichier
// encore ouvert), Aria continue sur l'ancien dossier et réessaiera au
// prochain lancement. msg décrit la migration effectuée, s'il y en a eu une.
func Migrer(racine string) (p Paths, msg string, err error) {
	return migrer(racine, 20, 500*time.Millisecond)
}

func migrer(racine string, essais int, pause time.Duration) (Paths, string, error) {
	nouveau := filepath.Join(racine, Dossier)
	ancien := filepath.Join(racine, AncienDossier)
	if existe(nouveau) || !existe(ancien) {
		return In(nouveau), "", nil
	}
	var err error
	for i := 0; i < essais; i++ {
		if err = os.Rename(ancien, nouveau); err == nil {
			return In(nouveau), fmt.Sprintf("Dossier d'Aria renommé : %s → %s", ancien, nouveau), nil
		}
		time.Sleep(pause) // l'ancienne version libère encore ses fichiers
	}
	return In(ancien), "", fmt.Errorf("renommage de %s impossible, Aria continue dessus pour cette fois : %w", ancien, err)
}

func existe(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Documents renvoie le dossier Documents de l'utilisateur (redirection
// OneDrive comprise) : point de départ des boîtes de dialogue.
func Documents() string { return documents() }

// SurOneDrive indique si p est dans un dossier synchronisé par OneDrive.
func SurOneDrive(p string) bool {
	for _, v := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		racine := os.Getenv(v)
		if racine == "" {
			continue
		}
		if rel, err := filepath.Rel(racine, p); err == nil && filepath.IsLocal(rel) || filepath.Clean(p) == filepath.Clean(racine) {
			return true
		}
	}
	return false
}

// MemeDisque indique si a et b sont sur le même lecteur (C:, D:…).
func MemeDisque(a, b string) bool {
	va, vb := filepath.VolumeName(a), filepath.VolumeName(b)
	return va != "" && strings.EqualFold(va, vb)
}

// In construit les emplacements quand données et cache partagent un dossier
// (v0.7 et avant, ou repli si le dossier du profil est inutilisable).
func In(base string) Paths { return Avec(base, base) }

// Avec construit les emplacements à partir du dossier des données et du cache.
func Avec(base, cache string) Paths {
	return Paths{
		Base:        base,
		Cache:       cache,
		Moteur:      filepath.Join(cache, "moteur"),
		Donnees:     filepath.Join(base, "data"),
		Certificats: filepath.Join(base, "certificats"),
		Config:      filepath.Join(base, "config.json"),
	}
}

// Profil renvoie le dossier du profil Windows (%USERPROFILE%).
func Profil() (string, error) { return os.UserHomeDir() }

// aDeplacer : ce qui quitte le cache pour le dossier des données (v0.8).
var aDeplacer = []string{"data", "certificats", "config.json", "aria.log", "aria.log.1", "fenetre.json", "data.restauration"}

// Deplacer installe les données dans profil\.Aria-Vault, en y déplaçant celles
// d'une version précédente rangées dans le cache. Le déplacement passe par un
// dossier .Aria-Vault.migration renommé à la fin : une coupure en plein
// milieu est reprise au lancement suivant, et rien n'est jamais supprimé.
// En cas d'échec (autre disque, fichier verrouillé), les éléments déjà
// déplacés reviennent et Aria continue dans le cache.
func Deplacer(profil string, cache Paths) (Paths, string, error) {
	nouveau := filepath.Join(profil, DossierDonnees)
	if profil == "" || SurOneDrive(nouveau) {
		// Une base SQLite synchronisée pendant qu'elle tourne se corrompt.
		return cache, "", fmt.Errorf("%s est synchronisé par OneDrive : les données restent dans %s", nouveau, cache.Base)
	}
	if existe(nouveau) {
		return Avec(nouveau, cache.Base), "", nil
	}

	elements := append([]string(nil), aDeplacer...)
	anciens, _ := filepath.Glob(filepath.Join(cache.Base, "data.avant-restauration-*"))
	for _, a := range anciens {
		elements = append(elements, filepath.Base(a))
	}

	etape := nouveau + ".migration"
	if err := os.MkdirAll(etape, 0o700); err != nil {
		return cache, "", fmt.Errorf("création de %s impossible, les données restent dans %s : %w", nouveau, cache.Base, err)
	}
	var deplaces []string
	for _, e := range elements {
		src, dst := filepath.Join(cache.Base, e), filepath.Join(etape, e)
		if !existe(src) || existe(dst) {
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			for _, d := range deplaces { // retour en arrière
				_ = os.Rename(filepath.Join(etape, d), filepath.Join(cache.Base, d))
			}
			_ = os.Remove(etape) // seulement s'il est vide
			return cache, "", fmt.Errorf("déplacement de %s vers %s impossible, les données restent dans %s : %w", e, nouveau, cache.Base, err)
		}
		deplaces = append(deplaces, e)
	}
	if err := os.Rename(etape, nouveau); err != nil {
		return cache, "", fmt.Errorf("finalisation de %s impossible (reprise au prochain lancement) : %w", nouveau, err)
	}
	if len(deplaces) == 0 {
		return Avec(nouveau, cache.Base), "", nil // première installation
	}
	return Avec(nouveau, cache.Base), fmt.Sprintf("Données d'Aria déplacées : %s → %s", cache.Base, nouveau), nil
}

// Existant renvoie p, ou son plus proche parent qui existe ("" si aucun).
// Utile pour ouvrir une boîte de dialogue Windows, qui refuse un dossier absent.
func Existant(p string) string {
	for {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			return ""
		}
		p = parent
	}
}
