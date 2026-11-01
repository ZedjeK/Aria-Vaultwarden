package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistant(t *testing.T) {
	racine := t.TempDir()
	docs := filepath.Join(racine, "Documents")
	os.MkdirAll(docs, 0o700)

	if got := Existant(filepath.Join(docs, "Aria Vault", "Sauvegardes")); got != docs {
		t.Errorf("dossier absent : %q, attendu le parent existant %q", got, docs)
	}
	if got := Existant(docs); got != docs {
		t.Errorf("dossier existant : %q", got)
	}
	if _, err := os.Stat(filepath.Join(docs, "Aria Vault")); err == nil {
		t.Error("aucun dossier ne doit être créé")
	}
}

func TestSurOneDrive(t *testing.T) {
	racine := t.TempDir()
	t.Setenv("OneDrive", filepath.Join(racine, "OneDrive"))
	t.Setenv("OneDriveConsumer", "")
	t.Setenv("OneDriveCommercial", "")
	if !SurOneDrive(filepath.Join(racine, "OneDrive", "Aria Vault")) {
		t.Error("sous-dossier OneDrive non reconnu")
	}
	if SurOneDrive(filepath.Join(racine, "OneDriveFaux", "x")) || SurOneDrive(filepath.Join(racine, "Documents")) {
		t.Error("faux positif OneDrive")
	}
}

func TestMigrer(t *testing.T) {
	racine := t.TempDir()
	ancien := filepath.Join(racine, AncienDossier)
	os.MkdirAll(filepath.Join(ancien, "data"), 0o700)
	os.WriteFile(filepath.Join(ancien, "data", "db.sqlite3"), []byte("coffre"), 0o600)

	p, msg, err := migrer(racine, 1, 0)
	if err != nil || msg == "" {
		t.Fatalf("migration : %v (msg %q)", err, msg)
	}
	if p.Base != filepath.Join(racine, Dossier) {
		t.Errorf("base = %q", p.Base)
	}
	if b, _ := os.ReadFile(filepath.Join(p.Donnees, "db.sqlite3")); string(b) != "coffre" {
		t.Errorf("coffre migré = %q", b)
	}
	if existe(ancien) {
		t.Error("l'ancien dossier doit avoir été renommé")
	}

	// Deuxième lancement : rien à faire.
	if p2, msg2, err := migrer(racine, 1, 0); err != nil || msg2 != "" || p2.Base != p.Base {
		t.Errorf("relance : %+v %q %v", p2, msg2, err)
	}
}

func TestMigrerNeToucheJamaisAuNouveau(t *testing.T) {
	racine := t.TempDir()
	os.MkdirAll(filepath.Join(racine, AncienDossier), 0o700)
	os.MkdirAll(filepath.Join(racine, Dossier), 0o700)
	p, msg, err := migrer(racine, 1, 0)
	if err != nil || msg != "" || p.Base != filepath.Join(racine, Dossier) {
		t.Errorf("%+v %q %v", p, msg, err)
	}
	if !existe(filepath.Join(racine, AncienDossier)) {
		t.Error("l'ancien dossier ne doit pas être supprimé quand le nouveau existe déjà")
	}
}

func TestMigrerPremierLancement(t *testing.T) {
	racine := t.TempDir()
	if p, msg, err := migrer(racine, 1, 0); err != nil || msg != "" || p.Base != filepath.Join(racine, Dossier) {
		t.Errorf("%+v %q %v", p, msg, err)
	}
}

func fichier(t *testing.T, p, contenu string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o700)
	if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDeplacer(t *testing.T) {
	racine := t.TempDir()
	t.Setenv("OneDrive", "")
	t.Setenv("OneDriveConsumer", "")
	t.Setenv("OneDriveCommercial", "")
	profil := filepath.Join(racine, "zeman")
	os.MkdirAll(profil, 0o700)
	cache := In(filepath.Join(racine, "Local", Dossier))
	fichier(t, filepath.Join(cache.Base, "data", "db.sqlite3"), "coffre")
	fichier(t, filepath.Join(cache.Base, "config.json"), "{}")
	fichier(t, filepath.Join(cache.Base, "certificats", "cert.pem"), "cert")
	fichier(t, filepath.Join(cache.Base, "data.avant-restauration-2026-10-01_100000", "db.sqlite3"), "ancien")
	fichier(t, filepath.Join(cache.Base, "moteur", "vaultwarden.exe"), "exe")
	fichier(t, filepath.Join(cache.Base, "instance.json"), "{}")

	p, msg, err := Deplacer(profil, cache)
	if err != nil || msg == "" {
		t.Fatalf("%v %q", err, msg)
	}
	nouveau := filepath.Join(profil, DossierDonnees)
	if p.Base != nouveau || p.Cache != cache.Base || p.Moteur != filepath.Join(cache.Base, "moteur") {
		t.Fatalf("%+v", p)
	}
	if b, _ := os.ReadFile(filepath.Join(p.Donnees, "db.sqlite3")); string(b) != "coffre" {
		t.Error("coffre non déplacé")
	}
	for _, e := range []string{"config.json", "certificats/cert.pem", "data.avant-restauration-2026-10-01_100000/db.sqlite3"} {
		if !existe(filepath.Join(nouveau, filepath.FromSlash(e))) {
			t.Errorf("%s non déplacé", e)
		}
	}
	// Le cache garde ce qui se recrée tout seul, et la relève.
	for _, e := range []string{"moteur/vaultwarden.exe", "instance.json"} {
		if !existe(filepath.Join(cache.Base, filepath.FromSlash(e))) {
			t.Errorf("%s ne doit pas bouger", e)
		}
	}
	if existe(nouveau + ".migration") {
		t.Error("dossier d'étape restant")
	}

	// Lancement suivant : rien à faire.
	if p2, msg2, err := Deplacer(profil, cache); err != nil || msg2 != "" || p2 != p {
		t.Errorf("relance : %+v %q %v", p2, msg2, err)
	}
}

// Coupure en plein déplacement : la reprise termine le travail.
func TestDeplacerReprise(t *testing.T) {
	racine := t.TempDir()
	t.Setenv("OneDrive", "")
	profil := filepath.Join(racine, "zeman")
	cache := In(filepath.Join(racine, "Local", Dossier))
	fichier(t, filepath.Join(cache.Base, "config.json"), "{}")
	etape := filepath.Join(profil, DossierDonnees+".migration")
	fichier(t, filepath.Join(etape, "data", "db.sqlite3"), "coffre") // déjà déplacé avant la coupure

	p, _, err := Deplacer(profil, cache)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(p.Donnees, "db.sqlite3")); string(b) != "coffre" {
		t.Error("coffre perdu à la reprise")
	}
	if !existe(p.Config) {
		t.Error("config.json non déplacé à la reprise")
	}
}

func TestDeplacerPremiereInstallation(t *testing.T) {
	racine := t.TempDir()
	t.Setenv("OneDrive", "")
	profil := filepath.Join(racine, "zeman")
	cache := In(filepath.Join(racine, "Local", Dossier))
	p, msg, err := Deplacer(profil, cache)
	if err != nil || msg != "" || p.Base != filepath.Join(profil, DossierDonnees) || !existe(p.Base) {
		t.Errorf("%+v %q %v", p, msg, err)
	}
}

func TestDeplacerRefuseOneDrive(t *testing.T) {
	racine := t.TempDir()
	profil := filepath.Join(racine, "zeman")
	t.Setenv("OneDrive", profil)
	cache := In(filepath.Join(racine, "Local", Dossier))
	if p, _, err := Deplacer(profil, cache); err == nil || p != cache {
		t.Errorf("profil OneDrive accepté : %+v", p)
	}
}
