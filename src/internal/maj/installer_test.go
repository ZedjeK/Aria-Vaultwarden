package maj

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func fauxZip(t *testing.T, fichiers map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for nom, contenu := range fichiers {
		w, _ := zw.Create(nom)
		w.Write([]byte(contenu))
	}
	zw.Close()
	return buf.Bytes()
}

func TestTelechargerInstaller(t *testing.T) {
	const nouvelExe = "MZ nouvelle version"
	z := fauxZip(t, map[string]string{"Aria-Vault.exe": nouvelExe, "LICENSE": "MIT", "BUILDINFO.txt": "0.6.1", "../evasion.txt": "non"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write(z) }))
	defer srv.Close()

	v := Nouveau("0.6.0")
	info := Info{Version: "0.6.1", Installable: true, Zip: srv.URL, ZipSHA: sha(z), ExeSHA: sha([]byte(nouvelExe)), Taille: int64(len(z))}
	dl := t.TempDir()
	var vu int64
	chemin, err := v.Telecharger(context.Background(), info, dl, func(fait, _ int64) { vu = fait })
	if err != nil || vu != int64(len(z)) {
		t.Fatalf("%v (progression %d)", err, vu)
	}

	dir := t.TempDir()
	exe := filepath.Join(dir, "Aria-Vault (perso).exe")
	os.WriteFile(exe, []byte("MZ ancienne version"), 0o755)

	// Empreinte de l'exe fausse : rien ne bouge.
	if _, err := Installer(chemin, exe, sha([]byte("autre"))); err == nil {
		t.Fatal("exe à la mauvaise empreinte installé")
	}
	if b, _ := os.ReadFile(exe); string(b) != "MZ ancienne version" {
		t.Fatal("exe modifié malgré l'erreur")
	}

	annuler, err := Installer(chemin, exe, info.ExeSHA)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != nouvelExe {
		t.Errorf("exe = %q", b)
	}
	if b, _ := os.ReadFile(exe + SuffixeAncien); string(b) != "MZ ancienne version" {
		t.Errorf("ancien exe = %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "BUILDINFO.txt")); string(b) != "0.6.1" {
		t.Error("BUILDINFO.txt non mis à jour")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "evasion.txt")); err == nil {
		t.Error("chemin hors du dossier écrit")
	}

	// La nouvelle version ne démarre pas : retour en arrière.
	if err := annuler(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "MZ ancienne version" {
		t.Errorf("après annulation, exe = %q", b)
	}

	// Zip altéré : refusé au téléchargement.
	info.ZipSHA = sha([]byte("autre chose"))
	if _, err := v.Telecharger(context.Background(), info, dl, nil); err == nil {
		t.Error("zip à la mauvaise empreinte accepté")
	}
	// Après l'annulation, l'ancien exe a repris sa place : rien à nettoyer.
	if Nettoyer(exe) {
		t.Error("un .old restait après l'annulation")
	}
	os.WriteFile(exe+SuffixeAncien, []byte("x"), 0o644)
	if !Nettoyer(exe) {
		t.Error("ancien exe non supprimé")
	}
}
