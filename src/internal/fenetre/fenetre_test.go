package fenetre

import (
	"path/filepath"
	"testing"
)

func TestDefaut(t *testing.T) {
	// Écran 1920×1080 (barre des tâches de 40 px)
	r := Defaut(Rect{0, 0, 1920, 1040})
	if r.Largeur != 1280 || r.Hauteur != 820 || r.X != 320 || r.Y != 110 {
		t.Errorf("1080p : %+v", r)
	}
	// Petit portable 1366×768
	r = Defaut(Rect{0, 0, 1366, 728})
	if r.Largeur != 1256 || r.Hauteur != 655 {
		t.Errorf("1366 : %+v", r)
	}
}

func TestPlacer(t *testing.T) {
	principal := Rect{0, 0, 1920, 1040}
	second := Rect{1920, 0, 2560, 1400}
	enr := Etat{Rect: Rect{2100, 100, 1500, 900}, Agrandie: true}

	if got := Placer(enr, true, []Rect{principal, second}, principal); got != enr {
		t.Errorf("sur le second écran : %+v", got)
	}
	// Second écran débranché : retour au centre de l'écran principal, toujours agrandie.
	got := Placer(enr, true, []Rect{principal}, principal)
	if got.Rect != Defaut(principal) || !got.Agrandie {
		t.Errorf("écran débranché : %+v", got)
	}
	// Barre de titre au-dessus de l'écran : redescendue.
	if got := Placer(Etat{Rect: Rect{100, -50, 1000, 700}}, true, []Rect{principal}, principal); got.Y != 0 {
		t.Errorf("au-dessus : %+v", got)
	}
	// Premier lancement.
	if got := Placer(Etat{}, false, []Rect{principal}, principal); got.Rect != Defaut(principal) || got.Agrandie {
		t.Errorf("premier lancement : %+v", got)
	}
}

func TestChargerEnregistrer(t *testing.T) {
	p := filepath.Join(t.TempDir(), "fenetre.json")
	if _, ok := Charger(p); ok {
		t.Fatal("fichier absent lu")
	}
	e := Etat{Rect: Rect{10, 20, 1300, 800}, Agrandie: true}
	if err := Enregistrer(p, e); err != nil {
		t.Fatal(err)
	}
	if got, ok := Charger(p); !ok || got != e {
		t.Errorf("%+v %v", got, ok)
	}
	// Fenêtre réduite (0×0) : l'état précédent est gardé.
	Enregistrer(p, Etat{})
	if got, _ := Charger(p); got != e {
		t.Errorf("écrasé par une fenêtre réduite : %+v", got)
	}
}
