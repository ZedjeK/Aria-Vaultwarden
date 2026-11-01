// Package vaultwarden embarque Vaultwarden (compilé depuis le source officiel)
// et son interface web, et le fait tourner comme processus enfant d'Aria :
// rien n'est installé sur la machine.
package vaultwarden

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"aria-vault/internal/ports"
)

// Les deux archives sont produites par build.ps1 (ou la CI) et ne sont pas
// versionnées : assets/LISEZMOI.txt garde le dossier présent dans git.
//
//go:embed assets
var assets embed.FS

const (
	exeAsset      = "assets/vaultwarden.exe.gz"
	webVaultAsset = "assets/web-vault.tar.gz"
	versionAsset  = "assets/version.txt" // écrit par build.ps1
	startTimeout  = 60 * time.Second     // la 1re ouverture d'un gros coffre peut migrer la base
)

// Version renvoie la version de Vaultwarden embarquée, si build.ps1 l'a indiquée.
func Version() string {
	b, err := assets.ReadFile(versionAsset)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Options règle le lancement de Vaultwarden.
type Options struct {
	Domaine      string // vault.exemple.duckdns.org
	Port         int    // port local, sur 127.0.0.1 uniquement
	Inscriptions bool   // autorise la création de comptes
	Invitations  bool   // les organisations peuvent inviter de nouveaux comptes
	Push         *Push  // notifications push (nil = désactivées)
	SMTP         *SMTP  // e-mails (nil = désactivés)
}

// Push : relais de notifications de Bitwarden.
type Push struct {
	ID, Cle               string
	RelayURI, IdentityURI string // vides = région US (valeurs par défaut)
}

// SMTP : serveur d'envoi des e-mails.
type SMTP struct {
	Hote, Securite, Utilisateur, MotDePasse, De string
	Port                                        int
}

// Server pilote le processus vaultwarden.exe.
type Server struct {
	moteur  string // vaultwarden.exe et web-vault
	donnees string // le coffre

	mu   sync.Mutex
	cmd  *exec.Cmd
	done chan struct{}
}

// New prépare le pilote ; rien n'est lancé avant Start.
func New(moteurDir, donneesDir string) *Server {
	return &Server{moteur: moteurDir, donnees: donneesDir}
}

// Done est fermé quand vaultwarden.exe s'arrête.
func (s *Server) Done() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

// Start extrait Vaultwarden, le lance et attend qu'il réponde.
// logf reçoit les étapes à afficher.
func (s *Server) Start(opts Options, logf func(string, ...any)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done != nil {
		select {
		case <-s.done:
		default:
			return errors.New("Vaultwarden tourne déjà")
		}
	}

	addr := "127.0.0.1:" + strconv.Itoa(opts.Port)
	if err := ports.Check(addr); err != nil {
		return err
	}

	for _, d := range []string{s.moteur, s.donnees} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}

	// Toujours réextraits : on ne lance jamais un fichier laissé sur le disque.
	exe := filepath.Join(s.moteur, "vaultwarden.exe")
	sum, err := extractExe(exe)
	if err != nil {
		return fmt.Errorf("extraction de vaultwarden.exe : %w", err)
	}
	logf("vaultwarden.exe extrait (SHA-256 %s…)", sum[:16])

	webVault := filepath.Join(s.moteur, "web-vault")
	if err := extractWebVault(webVault); err != nil {
		return fmt.Errorf("extraction de l'interface web : %w", err)
	}
	if v := webVaultVersion(webVault); v != "" {
		logf("Interface web %s extraite", v)
	} else {
		logf("Interface web extraite")
	}

	// data\config.json est écrit par le panneau /admin : ses valeurs priment
	// sur celles qu'Aria transmet. Mieux vaut le savoir.
	if _, err := os.Stat(filepath.Join(s.donnees, "config.json")); err == nil {
		logf("Attention : data\\config.json existe, ses réglages priment sur ceux d'Aria")
	}

	cmd := exec.Command(exe)
	cmd.Dir = s.moteur // aucun .env ici : toute la config passe par l'environnement
	cmd.Env = environment(s.donnees, webVault, opts)
	logPath := filepath.Join(s.donnees, "vaultwarden-console.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	prepareChild(cmd)
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("lancement de vaultwarden.exe : %w", err)
	}
	// Rattaché à Aria : si Aria se ferme ou plante, Windows arrête Vaultwarden aussi.
	if err := bindToParent(cmd); err != nil {
		logf("Attention : liaison au processus Aria impossible (%v)", err)
	}

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		logFile.Close()
		close(done)
	}()
	s.cmd, s.done = cmd, done

	deadline := time.Now().Add(startTimeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case <-done:
			return fmt.Errorf("Vaultwarden s'est arrêté au démarrage : %s", lastLine(logPath))
		default:
		}
		if resp, err := client.Get("http://" + addr + "/alive"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				logf("Vaultwarden prêt sur %s (PID %d)", addr, cmd.Process.Pid)
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	s.stopLocked()
	return fmt.Errorf("Vaultwarden ne répond pas après %s : voir %s", startTimeout, logPath)
}

// Stop arrête Vaultwarden s'il tourne.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *Server) stopLocked() {
	if s.cmd == nil || s.done == nil {
		return
	}
	select {
	case <-s.done:
		return
	default:
	}
	// Arrêt brutal sans risque pour le coffre : SQLite est transactionnel,
	// au pire la dernière écriture en cours est annulée, jamais à moitié faite.
	_ = s.cmd.Process.Kill()
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}

// heritees liste les seules variables de la session transmises à Vaultwarden
// (noms en majuscules : Windows ignore la casse). Une liste blanche plutôt
// qu'une liste noire : Vaultwarden lit des dizaines de variables, et une seule
// oubliée (DISABLE_ADMIN_TOKEN ouvrirait /admin sans authentification,
// IP_HEADER tromperait le limiteur) suffirait à contourner les réglages d'Aria.
var heritees = map[string]bool{
	// Windows : sans SYSTEMROOT, Winsock refuse de démarrer.
	"SYSTEMROOT": true, "SYSTEMDRIVE": true, "WINDIR": true, "COMSPEC": true,
	"PATH": true, "PATHEXT": true, "TEMP": true, "TMP": true,
	"USERPROFILE": true, "USERNAME": true, "USERDOMAIN": true, "COMPUTERNAME": true,
	"HOMEDRIVE": true, "HOMEPATH": true, "APPDATA": true, "LOCALAPPDATA": true,
	"PROGRAMDATA": true, "PROGRAMFILES": true,
	"NUMBER_OF_PROCESSORS": true, "PROCESSOR_ARCHITECTURE": true,
	// Autres systèmes (tests et compilation hors Windows).
	"HOME": true, "TMPDIR": true, "LANG": true, "TZ": true,
	// Proxy de sortie (icônes des sites, vérification des fuites de mots de passe).
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true, "ALL_PROXY": true,
}

// environment construit la configuration de Vaultwarden.
func environment(donnees, webVault string, opts Options) []string {
	set := map[string]string{
		"DATA_FOLDER":         donnees,
		"WEB_VAULT_FOLDER":    webVault,
		"WEB_VAULT_ENABLED":   "true",
		"ROCKET_ADDRESS":      "127.0.0.1",
		"ROCKET_PORT":         strconv.Itoa(opts.Port),
		"DOMAIN":              "https://" + opts.Domaine,
		"SIGNUPS_ALLOWED":     strconv.FormatBool(opts.Inscriptions),
		"INVITATIONS_ALLOWED": strconv.FormatBool(opts.Invitations),
		"LOG_FILE":            filepath.Join(donnees, "vaultwarden.log"),
		"LOG_LEVEL":           "info",
		"EXTENDED_LOGGING":    "true",
	}
	if p := opts.Push; p != nil {
		set["PUSH_ENABLED"] = "true"
		set["PUSH_INSTALLATION_ID"] = p.ID
		set["PUSH_INSTALLATION_KEY"] = p.Cle
		if p.RelayURI != "" {
			set["PUSH_RELAY_URI"] = p.RelayURI
			set["PUSH_IDENTITY_URI"] = p.IdentityURI
		}
	}
	if m := opts.SMTP; m != nil {
		set["SMTP_HOST"] = m.Hote
		set["SMTP_PORT"] = strconv.Itoa(m.Port)
		set["SMTP_SECURITY"] = m.Securite
		set["SMTP_FROM"] = m.De
		set["SMTP_FROM_NAME"] = "Aria Vault"
		if m.Utilisateur != "" {
			set["SMTP_USERNAME"] = m.Utilisateur
			set["SMTP_PASSWORD"] = m.MotDePasse
		}
	}
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		// Seules les variables système de la liste blanche passent : aucune
		// config Vaultwarden héritée de la session, Aria reste la seule source.
		if heritees[strings.ToUpper(k)] {
			env = append(env, kv)
		}
	}
	for k, v := range set {
		env = append(env, k+"="+v)
	}
	return env
}

// extractExe décompresse vaultwarden.exe et renvoie son SHA-256.
func extractExe(dst string) (string, error) {
	gz, err := assets.ReadFile(exeAsset)
	if errors.Is(err, fs.ErrNotExist) {
		return "", errors.New("Vaultwarden n'est pas embarqué dans cette compilation : lance build.ps1")
	}
	if err != nil {
		return "", err
	}
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return "", err
	}
	defer zr.Close()

	tmp := dst + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	// gzip vérifie son CRC en fin de flux : une archive abîmée échoue ici.
	if _, err := io.Copy(io.MultiWriter(f, h), zr); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
		os.Remove(tmp)
		return "", fmt.Errorf("%s est verrouillé : une autre instance tourne-t-elle encore ? (%w)", dst, err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractWebVault remplace le dossier web-vault par le contenu embarqué.
func extractWebVault(dst string) error {
	tgz, err := assets.ReadFile(webVaultAsset)
	if errors.Is(err, fs.ErrNotExist) {
		return errors.New("l'interface web n'est pas embarquée dans cette compilation : lance build.ps1")
	}
	if err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if err := untar(tgz, tmp, "web-vault"); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	if _, err := os.Stat(filepath.Join(tmp, "index.html")); err != nil {
		os.RemoveAll(tmp)
		return errors.New("archive de l'interface web invalide (index.html absent)")
	}
	if err := os.RemoveAll(dst); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// untar extrait le dossier racine `root` de l'archive tar.gz dans dst.
// Les chemins sortant de dst et les liens sont refusés.
func untar(tgz []byte, dst, root string) error {
	zr, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	prefix := root + "/"
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(filepath.ToSlash(hdr.Name), "./")
		if name == root || name == prefix {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			return fmt.Errorf("entrée inattendue dans l'archive : %s", hdr.Name)
		}
		rel := filepath.FromSlash(strings.TrimPrefix(name, prefix))
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("chemin refusé dans l'archive : %s", hdr.Name)
		}
		target := filepath.Join(dst, rel)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeXGlobalHeader:
			// métadonnées pax, sans fichier
		default:
			return fmt.Errorf("type d'entrée refusé dans l'archive : %s", hdr.Name)
		}
	}
}

// webVaultVersion lit la version de l'interface web, si elle est indiquée.
func webVaultVersion(dir string) string {
	for _, name := range []string{"vw-version.json", "version.json"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var v struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &v) == nil && v.Version != "" {
			return v.Version
		}
	}
	return ""
}

// lastLine renvoie la dernière ligne non vide d'un journal.
func lastLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "voir " + path
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return "voir " + path
}
