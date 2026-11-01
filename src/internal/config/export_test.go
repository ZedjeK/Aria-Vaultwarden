package config

import (
	"bytes"
	"testing"

	"aria-vault/internal/dnsprov"
)

func TestExportImport(t *testing.T) {
	c := &Config{Domaine: "vault.exemple.duckdns.org", Email: "moi@exemple.fr"}
	if errs := c.SetIdentifiants(dnsprov.DuckDNS, map[string]string{"jeton": jeton}); errs != nil {
		t.Fatal(errs)
	}

	if _, err := c.Exporter("court"); err == nil {
		t.Error("phrase trop courte acceptée")
	}
	data, err := c.Exporter("ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	// Rien de lisible dans le fichier : ni jeton, ni domaine, ni e-mail.
	for _, secret := range []string{jeton, "vault.exemple", "duckdns", "moi@exemple.fr"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Errorf("%q apparaît en clair dans l'export", secret)
		}
	}

	got, err := LireExport(data, "ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	if got.Identifiants["jeton"] != jeton || got.Jeton != jeton || got.Fournisseur != dnsprov.DuckDNS || got.Domaine != c.Domaine || got.Email != c.Email {
		t.Errorf("import = %+v", got)
	}

	if _, err := LireExport(data, "mauvaise phrase"); err != ErrPhrase {
		t.Errorf("mauvaise phrase : %v", err)
	}
	if _, err := LireExport(data, ""); err == nil {
		t.Error("import sans phrase accepté")
	}

	// Un seul octet modifié, dans l'en-tête ou le chiffré : refusé.
	for _, i := range []int{12, len(data) - 1} {
		altere := bytes.Clone(data)
		altere[i] ^= 1
		if _, err := LireExport(altere, "ma phrase de passe"); err != ErrPhrase {
			t.Errorf("octet %d modifié : %v", i, err)
		}
	}

	// Paramètres scrypt démesurés : refusés avant tout calcul.
	piege := bytes.Clone(data)
	piege[8] = 30
	if _, err := LireExport(piege, "ma phrase de passe"); err == nil || err == ErrPhrase {
		t.Errorf("paramètres piégés : %v", err)
	}

	if _, err := LireExport([]byte(`{"format":"aria-vault-reglages"}`), "x"); err == nil {
		t.Error("ancien format ou fichier étranger accepté")
	}

	// Deux exports de la même config ne se ressemblent pas (sel et nonce aléatoires).
	data2, _ := c.Exporter("ma phrase de passe")
	if bytes.Equal(data, data2) {
		t.Error("deux exports identiques")
	}
}

func TestExportOVH(t *testing.T) {
	c := &Config{Domaine: "vault.exemple.fr"}
	ids := map[string]string{"applicationKey": "ak123456789", "applicationSecret": "as123456789", "consumerKey": "ck123456789"}
	if errs := c.SetIdentifiants(dnsprov.OVH, ids); errs != nil {
		t.Fatal(errs)
	}
	data, err := c.Exporter("ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	got, err := LireExport(data, "ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	if got.Fournisseur != dnsprov.OVH || got.Identifiants["endpoint"] != "ovh-eu" || got.Identifiants["consumerKey"] != "ck123456789" || got.Jeton != "" {
		t.Errorf("import = %+v", got)
	}
}

func TestExportServices(t *testing.T) {
	c := &Config{Domaine: "vault.exemple.duckdns.org", Invitations: true}
	c.SetIdentifiants(dnsprov.DuckDNS, map[string]string{"jeton": jeton})
	if errs := c.SetPush("eu", "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0", "Cl3Secrete42"); errs != nil {
		t.Fatal(errs)
	}
	if errs := c.SetSMTP(SMTPEntree{Hote: "smtp.gmail.com", Securite: "starttls", Utilisateur: "moi@gmail.com", MotDePasse: "mdpappli", De: "moi@gmail.com"}); errs != nil {
		t.Fatal(errs)
	}
	data, err := c.Exporter("ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("mdpappli")) || bytes.Contains(data, []byte("Cl3Secrete42")) {
		t.Fatal("secret lisible dans l'export")
	}
	imp, err := LireExport(data, "ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}

	// Sur un autre PC, avec déjà des réglages : tout est remplacé.
	var autre Config
	if err := autre.Appliquer(imp); err != nil {
		t.Fatal(err)
	}
	if k, _ := autre.PushCle(); k != "Cl3Secrete42" || autre.Push.Region != "eu" {
		t.Errorf("push : %+v %q", autre.Push, k)
	}
	if m, _ := autre.SMTPMotDePasse(); m != "mdpappli" || !autre.Invitations {
		t.Errorf("smtp : %q", m)
	}

	// Un fichier d'avant la v0.8 n'efface pas les services configurés ici.
	imp.Push, imp.SMTP = nil, nil
	if err := autre.Appliquer(imp); err != nil || autre.Push == nil || autre.SMTP == nil {
		t.Errorf("services effacés par un ancien fichier : %v", err)
	}
}
