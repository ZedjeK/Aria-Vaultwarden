// Package fenetre retient la taille, la position et l'état « agrandie » de la
// fenêtre d'un lancement à l'autre, et choisit une taille adaptée à l'écran
// au premier lancement. Les coordonnées sont en pixels logiques (DIP).
package fenetre

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Rect est un rectangle à l'écran.
type Rect struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	Largeur int `json:"largeur"`
	Hauteur int `json:"hauteur"`
}

// Etat est enregistré dans fenetre.json.
type Etat struct {
	Rect
	Agrandie bool `json:"agrandie,omitempty"`
}

// Tailles de référence.
const (
	LargeurIdeale, HauteurIdeale = 1280, 820
	LargeurMin, HauteurMin       = 420, 520
	visibleMin                   = 120 // une fenêtre doit dépasser d'au moins ça sur un écran
)

// Charger lit fenetre.json (ok = false s'il n'existe pas ou est illisible).
func Charger(chemin string) (Etat, bool) {
	data, err := os.ReadFile(chemin)
	if err != nil {
		return Etat{}, false
	}
	var e Etat
	if json.Unmarshal(data, &e) != nil || e.Largeur <= 0 || e.Hauteur <= 0 {
		return Etat{}, false
	}
	return e, true
}

// Enregistrer écrit fenetre.json.
func Enregistrer(chemin string, e Etat) error {
	if e.Largeur < LargeurMin || e.Hauteur < HauteurMin {
		return nil // fenêtre réduite ou détruite : rien de fiable à retenir
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(chemin), 0o700); err != nil {
		return err
	}
	return os.WriteFile(chemin, data, 0o600)
}

// Defaut centre une fenêtre de taille idéale sur la zone de travail de
// l'écran principal, sans dépasser 92 % de sa largeur et 90 % de sa hauteur.
func Defaut(travail Rect) Rect {
	l := min(LargeurIdeale, travail.Largeur*92/100)
	h := min(HauteurIdeale, travail.Hauteur*90/100)
	l, h = max(l, LargeurMin), max(h, HauteurMin)
	return Rect{
		X:       travail.X + (travail.Largeur-l)/2,
		Y:       travail.Y + (travail.Hauteur-h)/2,
		Largeur: l,
		Hauteur: h,
	}
}

// Placer renvoie où ouvrir la fenêtre : l'état enregistré s'il est encore
// visible sur un des écrans (un écran débranché ne doit pas la perdre),
// sinon la taille par défaut sur l'écran principal.
func Placer(e Etat, ok bool, ecrans []Rect, principal Rect) Etat {
	if ok {
		for _, s := range ecrans {
			if visible(e.Rect, s) {
				r := e.Rect
				// Jamais plus grande que l'écran qui l'accueille.
				r.Largeur = min(max(r.Largeur, LargeurMin), max(s.Largeur, LargeurMin))
				r.Hauteur = min(max(r.Hauteur, HauteurMin), max(s.Hauteur, HauteurMin))
				// La barre de titre reste atteignable.
				r.Y = max(r.Y, s.Y)
				return Etat{Rect: r, Agrandie: e.Agrandie}
			}
		}
	}
	return Etat{Rect: Defaut(principal), Agrandie: ok && e.Agrandie}
}

func visible(r, s Rect) bool {
	l := min(r.X+r.Largeur, s.X+s.Largeur) - max(r.X, s.X)
	h := min(r.Y+r.Hauteur, s.Y+s.Hauteur) - max(r.Y, s.Y)
	return l >= visibleMin && h >= visibleMin/2
}
