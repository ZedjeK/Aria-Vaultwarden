package maj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLire(t *testing.T) {
	ok := `{"tag_name":"v0.6.1","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/v0.6.1"}`
	info, err := Lire([]byte(ok), "0.6.0")
	if err != nil || !info.Disponible || info.Version != "0.6.1" {
		t.Fatalf("%+v %v", info, err)
	}
	if info, _ := Lire([]byte(ok), "0.6.1"); info.Disponible {
		t.Error("même version annoncée comme nouvelle")
	}
	if info, _ := Lire([]byte(ok), "dev"); !info.Disponible {
		t.Error("une version dev doit voir la version publiée")
	}
	for _, bad := range []string{
		`{"tag_name":"v0.6.1","html_url":"https://exemple.com/piege"}`,
		`{"tag_name":"latest","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/latest"}`,
		`{"tag_name":"v0.6.1","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/v0.6.1","draft":true}`,
		`pas du json`,
	} {
		if _, err := Lire([]byte(bad), "0.6.0"); err == nil {
			t.Errorf("accepté : %s", bad)
		}
	}
}

func TestVerifier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent manquant")
		}
		w.Write([]byte(`{"tag_name":"v1.0.0","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/v1.0.0"}`))
	}))
	defer srv.Close()
	v := Nouveau("0.6.0")
	v.adresse = srv.URL
	info, err := v.Verifier(context.Background())
	if err != nil || !info.Disponible || v.Info().Version != "1.0.0" {
		t.Fatalf("%+v %v", info, err)
	}
	// Erreur réseau : le dernier résultat connu est gardé, avec l'erreur.
	srv.Close()
	info, err = v.Verifier(context.Background())
	if err == nil || info.Version != "1.0.0" || info.Erreur == "" {
		t.Errorf("%+v %v", info, err)
	}
}

func TestLireInstallable(t *testing.T) {
	const zipSHA = "9082edb5cdd219b57c82400ecc350d907f4a95f90df542ace86a67e91ff6e84c"
	const exeSHA = "2b13f26d4e66bb781161467e49a888db6c48318b6f59561d8c231f7db17be714"
	corps := func(digest, url string) string {
		return `{"tag_name":"v0.6.1","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/v0.6.1",
		"body":"## Empreintes SHA-256\r\n\r\n` + "```" + `\r\n` + zipSHA + `  Aria-Vault-0.6.1.zip\r\n` + exeSHA + `  Aria-Vault.exe (dans le zip)\r\n` + "```" + `",
		"assets":[{"name":"Aria-Vault-0.6.1.zip","browser_download_url":"` + url + `","digest":"` + digest + `","size":58983288}]}`
	}
	bonne := "https://github.com/ZedjeK/Aria-Vault/releases/download/v0.6.1/Aria-Vault-0.6.1.zip"
	info, err := Lire([]byte(corps("sha256:"+zipSHA, bonne)), "0.6.0")
	if err != nil || !info.Installable || info.ZipSHA != zipSHA || info.ExeSHA != exeSHA || info.Zip != bonne {
		t.Fatalf("%+v %v", info, err)
	}
	// Empreinte GitHub et notes en désaccord : pas d'installation automatique.
	if info, _ := Lire([]byte(corps("sha256:"+exeSHA, bonne)), "0.6.0"); info.Installable {
		t.Error("empreintes discordantes acceptées")
	}
	// Zip hébergé ailleurs que dans les releases du dépôt : refusé.
	if info, _ := Lire([]byte(corps("", "https://exemple.com/Aria-Vault-0.6.1.zip")), "0.6.0"); info.Installable {
		t.Error("zip étranger accepté")
	}
}
