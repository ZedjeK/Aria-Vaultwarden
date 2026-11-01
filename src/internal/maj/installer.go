package maj

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// TailleMax borne le téléchargement (la release fait environ 60 Mo).
const TailleMax = 300 << 20

// Suffixes des fichiers laissés à côté de l'exe pendant une mise à jour.
const (
	SuffixeAncien  = ".old"
	SuffixeNouveau = ".nouveau"
)

// Progression reçoit les octets téléchargés et le total attendu.
type Progression func(fait, total int64)

// Telecharger récupère le zip de la release dans dir et vérifie son SHA-256.
// Renvoie le chemin du zip vérifié.
func (v *Verificateur) Telecharger(ctx context.Context, info Info, dir string, prog Progression) (string, error) {
	if !info.Installable {
		return "", errors.New("cette version ne peut pas être installée automatiquement : télécharge-la depuis sa page")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.Zip, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Aria-Vault/"+v.actuel)
	client := &http.Client{} // pas de délai global : le téléchargement peut être long
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("téléchargement impossible : %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("téléchargement refusé par GitHub : %s", resp.Status)
	}

	final := filepath.Join(dir, "Aria-Vault-"+info.Version+".zip")
	part := final + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	total := info.Taille
	var fait int64
	buf := make([]byte, 256<<10)
	src := io.LimitReader(resp.Body, TailleMax+1)
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				f.Close()
				os.Remove(part)
				return "", err
			}
			h.Write(buf[:n])
			fait += int64(n)
			if fait > TailleMax {
				f.Close()
				os.Remove(part)
				return "", errors.New("fichier téléchargé trop volumineux")
			}
			if prog != nil {
				prog(fait, total)
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(part)
			return "", fmt.Errorf("téléchargement interrompu : %w", rerr)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(part)
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != info.ZipSHA {
		os.Remove(part)
		return "", fmt.Errorf("empreinte du zip incorrecte (attendue %.12s…, obtenue %.12s…) : rien n'a été installé", info.ZipSHA, got)
	}
	if err := os.Rename(part, final); err != nil {
		os.Remove(part)
		return "", err
	}
	return final, nil
}

// Installer remplace exe par celui du zip (vérifié par son SHA-256) et
// rafraîchit les fichiers de licence posés à côté. L'exe en cours est renommé
// en <exe>.old (Windows autorise le renommage d'un programme lancé) ; la
// fonction annuler remet tout en place si la nouvelle version ne démarre pas.
func Installer(zipPath, exe, exeSHA string) (annuler func() error, err error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("zip illisible : %w", err)
	}
	defer zr.Close()

	var entreeExe *zip.File
	var annexes []*zip.File
	for _, f := range zr.File {
		nom := f.Name
		if strings.ContainsAny(nom, `/\:`) || nom == "" || f.FileInfo().IsDir() {
			continue // seulement des fichiers à plat, sans chemin
		}
		switch {
		case nom == "Aria-Vault.exe":
			entreeExe = f
		case nom == "LICENSE" || strings.HasSuffix(strings.ToLower(nom), ".txt"):
			annexes = append(annexes, f)
		}
	}
	if entreeExe == nil {
		return nil, errors.New("Aria-Vault.exe absent du zip")
	}

	dir := filepath.Dir(exe)
	nouveau := exe + SuffixeNouveau
	if err := extraire(entreeExe, nouveau, exeSHA); err != nil {
		os.Remove(nouveau)
		return nil, err
	}

	ancien := exe + SuffixeAncien
	_ = os.Remove(ancien) // reste d'une mise à jour précédente
	if err := os.Rename(exe, ancien); err != nil {
		os.Remove(nouveau)
		return nil, fmt.Errorf("impossible de mettre l'exe actuel de côté (dossier protégé en écriture ?) : %w", err)
	}
	if err := os.Rename(nouveau, exe); err != nil {
		_ = os.Rename(ancien, exe)
		os.Remove(nouveau)
		return nil, err
	}
	annuler = func() error {
		if err := os.Rename(exe, nouveau); err != nil {
			return err
		}
		return os.Rename(ancien, exe)
	}

	// Licences et BUILDINFO : utiles mais pas indispensables.
	for _, f := range annexes {
		_ = extraire(f, filepath.Join(dir, f.Name), "")
	}
	return annuler, nil
}

// extraire écrit une entrée du zip ; si sha n'est pas vide, il doit correspondre.
func extraire(f *zip.File, dst, sha string) error {
	if f.UncompressedSize64 > TailleMax {
		return fmt.Errorf("%s : trop volumineux", f.Name)
	}
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	tmp := dst + ".tmp"
	w, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(w, h), io.LimitReader(r, TailleMax))
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("%s : %w", f.Name, err)
	}
	if sha != "" {
		if got := hex.EncodeToString(h.Sum(nil)); got != sha {
			os.Remove(tmp)
			return fmt.Errorf("empreinte de %s incorrecte : rien n'a été installé", f.Name)
		}
	}
	return os.Rename(tmp, dst)
}

// Nettoyer supprime l'ancien exe laissé par la dernière mise à jour.
// Renvoie true s'il y en avait un.
func Nettoyer(exe string) bool {
	return os.Remove(exe+SuffixeAncien) == nil
}
