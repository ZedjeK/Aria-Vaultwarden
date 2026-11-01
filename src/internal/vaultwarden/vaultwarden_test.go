package vaultwarden

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func makeTgz(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, name := range names {
		body := entries[name]
		if strings.HasSuffix(name, "/") {
			tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o755})
			continue
		}
		tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))})
		tw.Write([]byte(body))
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

func TestUntar(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "web-vault")
	tgz := makeTgz(t, map[string]string{
		"web-vault/":                "",
		"web-vault/index.html":      "<html>",
		"web-vault/app/main.js":     "js",
		"web-vault/vw-version.json": `{"version":"v2025.1.0"}`,
	})
	if err := untar(tgz, dst, "web-vault"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "app", "main.js")); string(b) != "js" {
		t.Errorf("app/main.js = %q", b)
	}
	if v := webVaultVersion(dst); v != "v2025.1.0" {
		t.Errorf("version = %q", v)
	}
}

func TestUntarRefuseLesEvasions(t *testing.T) {
	for _, name := range []string{"web-vault/../evasion.txt", "autre/fichier.txt", "/web-vault/abs.txt"} {
		tgz := makeTgz(t, map[string]string{name: "x"})
		if err := untar(tgz, filepath.Join(t.TempDir(), "w"), "web-vault"); err == nil {
			t.Errorf("%s accepté, aurait dû être refusé", name)
		}
	}
}

func TestEnvironment(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "hérité")
	t.Setenv("SIGNUPS_ALLOWED", "true")
	t.Setenv("DISABLE_ADMIN_TOKEN", "true")
	t.Setenv("IP_HEADER", "X-Client-IP")
	t.Setenv("SystemRoot", `C:\Windows`)
	env := environment("D", "W", Options{Domaine: "vault.exemple.duckdns.org", Port: 8000})
	get := func(k string) (string, int) {
		var v string
		n := 0
		for _, kv := range env {
			if key, val, _ := strings.Cut(kv, "="); key == k {
				v, n = val, n+1
			}
		}
		return v, n
	}
	for _, k := range []string{"ADMIN_TOKEN", "DISABLE_ADMIN_TOKEN", "IP_HEADER"} {
		if _, n := get(k); n != 0 {
			t.Errorf("%s hérité de la session doit être écarté", k)
		}
	}
	if v, n := get("SystemRoot"); v != `C:\Windows` || n != 1 {
		t.Errorf("SystemRoot = %q (%d fois), indispensable à Vaultwarden sous Windows", v, n)
	}
	if v, n := get("SIGNUPS_ALLOWED"); v != "false" || n != 1 {
		t.Errorf("SIGNUPS_ALLOWED = %q (%d fois)", v, n)
	}
	if v, _ := get("DOMAIN"); v != "https://vault.exemple.duckdns.org" {
		t.Errorf("DOMAIN = %q", v)
	}
	if v, _ := get("ROCKET_ADDRESS"); v != "127.0.0.1" {
		t.Errorf("ROCKET_ADDRESS = %q, Vaultwarden ne doit écouter qu'en local", v)
	}
}

func TestEnvironmentServices(t *testing.T) {
	get := func(env []string, k string) string {
		for _, kv := range env {
			if key, val, _ := strings.Cut(kv, "="); key == k {
				return val
			}
		}
		return ""
	}
	t.Setenv("PUSH_ENABLED", "true") // hérité de la session : ignoré
	sans := environment("D", "W", Options{Domaine: "v.exemple.fr", Port: 8000})
	for _, k := range []string{"PUSH_ENABLED", "SMTP_HOST", "SMTP_PASSWORD"} {
		if get(sans, k) != "" {
			t.Errorf("%s présent sans service configuré", k)
		}
	}
	if get(sans, "INVITATIONS_ALLOWED") != "false" {
		t.Error("invitations ouvertes par défaut")
	}

	avec := environment("D", "W", Options{
		Domaine: "v.exemple.fr", Port: 8000, Invitations: true,
		Push: &Push{ID: "id", Cle: "cle", RelayURI: "https://api.bitwarden.eu", IdentityURI: "https://identity.bitwarden.eu"},
		SMTP: &SMTP{Hote: "smtp.gmail.com", Port: 587, Securite: "starttls", Utilisateur: "moi@gmail.com", MotDePasse: "mdp", De: "moi@gmail.com"},
	})
	for k, want := range map[string]string{
		"PUSH_ENABLED": "true", "PUSH_INSTALLATION_ID": "id", "PUSH_INSTALLATION_KEY": "cle",
		"PUSH_RELAY_URI": "https://api.bitwarden.eu", "PUSH_IDENTITY_URI": "https://identity.bitwarden.eu",
		"SMTP_HOST": "smtp.gmail.com", "SMTP_PORT": "587", "SMTP_SECURITY": "starttls", "SMTP_FROM": "moi@gmail.com",
		"SMTP_USERNAME": "moi@gmail.com", "SMTP_PASSWORD": "mdp", "INVITATIONS_ALLOWED": "true",
	} {
		if got := get(avec, k); got != want {
			t.Errorf("%s = %q, attendu %q", k, got, want)
		}
	}
	// Région US : adresses par défaut de Vaultwarden.
	us := environment("D", "W", Options{Domaine: "v.exemple.fr", Port: 8000, Push: &Push{ID: "id", Cle: "cle"}})
	if get(us, "PUSH_RELAY_URI") != "" {
		t.Error("adresse du relais forcée pour la région US")
	}
}
