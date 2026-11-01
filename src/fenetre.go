package main

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"aria-vault/internal/fenetre"
)

// suiviFenetre ouvre la fenêtre à la taille adaptée à l'écran, puis retient
// sa taille, sa position et son état « agrandie » (fenetre.json).
type suiviFenetre struct {
	w      *application.WebviewWindow
	chemin string

	mu       sync.Mutex
	normale  fenetre.Rect // dernière position hors agrandissement
	agrandir bool         // à agrandir au premier affichage (lancement silencieux)
	minuteur *time.Timer
}

func nouveauSuivi(base string) *suiviFenetre {
	return &suiviFenetre{chemin: filepath.Join(base, "fenetre.json")}
}

// placer applique l'état enregistré (ou la taille par défaut) ; à appeler une
// fois l'appli démarrée, fenêtre encore cachée.
func (f *suiviFenetre) placer(app *application.App, afficher bool) {
	var ecrans []fenetre.Rect
	var principal fenetre.Rect
	for _, s := range app.Screen.GetAll() {
		r := versRect(s.WorkArea)
		ecrans = append(ecrans, r)
		if s.IsPrimary || principal.Largeur == 0 {
			principal = r
		}
	}
	if principal.Largeur == 0 {
		principal = fenetre.Rect{Largeur: fenetre.LargeurIdeale, Hauteur: fenetre.HauteurIdeale}
	}
	enr, ok := fenetre.Charger(f.chemin)
	e := fenetre.Placer(enr, ok, ecrans, principal)

	f.mu.Lock()
	f.normale, f.agrandir = e.Rect, e.Agrandie
	f.mu.Unlock()
	f.w.SetBounds(application.Rect{X: e.X, Y: e.Y, Width: e.Largeur, Height: e.Hauteur})

	// Les changements sont retenus au fil de l'eau (la fermeture d'Aria peut
	// survenir alors que la fenêtre n'existe déjà plus).
	for _, ev := range []events.WindowEventType{
		events.Common.WindowDidResize, events.Common.WindowDidMove,
		events.Common.WindowMaximise, events.Common.WindowUnMaximise,
	} {
		f.w.OnWindowEvent(ev, func(*application.WindowEvent) { f.planifier() })
	}
	if afficher {
		f.montrer()
	}
}

// montrer affiche la fenêtre au premier plan, sans défaire un agrandissement.
func (f *suiviFenetre) montrer() {
	f.w.Show()
	if f.w.IsMinimised() {
		f.w.Restore()
	}
	f.mu.Lock()
	agrandir := f.agrandir
	f.agrandir = false
	f.mu.Unlock()
	if agrandir {
		f.w.Maximise()
	}
	f.w.Focus()
}

// planifier enregistre l'état une demi-seconde après le dernier changement.
func (f *suiviFenetre) planifier() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.minuteur != nil {
		f.minuteur.Stop()
	}
	f.minuteur = time.AfterFunc(500*time.Millisecond, f.enregistrer)
}

func (f *suiviFenetre) enregistrer() {
	if !f.w.IsVisible() || f.w.IsMinimised() {
		return
	}
	agrandie := f.w.IsMaximised()
	b := f.w.Bounds()
	f.mu.Lock()
	if !agrandie && b.Width > 0 {
		f.normale = fenetre.Rect{X: b.X, Y: b.Y, Largeur: b.Width, Hauteur: b.Height}
	}
	e := fenetre.Etat{Rect: f.normale, Agrandie: agrandie}
	f.mu.Unlock()
	_ = fenetre.Enregistrer(f.chemin, e)
}

func versRect(r application.Rect) fenetre.Rect {
	return fenetre.Rect{X: r.X, Y: r.Y, Largeur: r.Width, Hauteur: r.Height}
}
