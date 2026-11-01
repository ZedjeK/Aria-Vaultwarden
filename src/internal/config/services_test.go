package config

import "testing"

func TestPush(t *testing.T) {
	var c Config
	if errs := c.SetPush("mars", "pas-un-id", ""); errs["region"] == "" || errs["id"] == "" || errs["cle"] == "" {
		t.Errorf("%v", errs)
	}
	const id = "0F1E2D3C-4B5A-6978-8796-A5B4C3D2E1F0"
	if errs := c.SetPush("EU", id, "Cl3Secrete42"); errs != nil {
		t.Fatal(errs)
	}
	if c.Push.Region != "eu" || c.Push.ID != "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0" {
		t.Errorf("%+v", c.Push)
	}
	// Clé vide : l'actuelle est gardée.
	if errs := c.SetPush("us", id, ""); errs != nil {
		t.Fatal(errs)
	}
	if k, err := c.PushCle(); err != nil || k != "Cl3Secrete42" || c.Push.Region != "us" {
		t.Errorf("%q %v", k, err)
	}
}

func TestSMTP(t *testing.T) {
	var c Config
	errs := c.SetSMTP(SMTPEntree{Hote: "pas un hote", Securite: "?", De: "moi"})
	for _, k := range []string{"hote", "securite", "de"} {
		if errs[k] == "" {
			t.Errorf("%s accepté", k)
		}
	}
	// Mot de passe en clair interdit.
	if errs := c.SetSMTP(SMTPEntree{Hote: "smtp.exemple.fr", Securite: "off", Utilisateur: "moi", MotDePasse: "x", De: "moi@exemple.fr"}); errs["securite"] == "" {
		t.Error("mot de passe sans chiffrement accepté")
	}
	// Gmail : espaces du mot de passe d'application retirés, port par défaut.
	in := SMTPEntree{Hote: "SMTP.gmail.com", Securite: "starttls", Utilisateur: "moi@gmail.com", MotDePasse: "abcd efgh ijkl mnop", De: "moi@gmail.com"}
	if errs := c.SetSMTP(in); errs != nil {
		t.Fatal(errs)
	}
	if c.SMTP.Port != 587 || c.SMTP.Hote != "smtp.gmail.com" {
		t.Errorf("%+v", c.SMTP)
	}
	if m, _ := c.SMTPMotDePasse(); m != "abcdefghijklmnop" {
		t.Errorf("mot de passe %q", m)
	}
	// Mot de passe vide, même serveur et même identifiant : gardé.
	in.MotDePasse = ""
	if errs := c.SetSMTP(in); errs != nil {
		t.Fatal(errs)
	}
	if m, _ := c.SMTPMotDePasse(); m != "abcdefghijklmnop" {
		t.Errorf("mot de passe perdu : %q", m)
	}
	// Autre identifiant : le mot de passe doit être ressaisi.
	in.Utilisateur = "autre@gmail.com"
	if errs := c.SetSMTP(in); errs["motDePasse"] == "" {
		t.Error("ancien mot de passe réutilisé pour un autre compte")
	}
}
