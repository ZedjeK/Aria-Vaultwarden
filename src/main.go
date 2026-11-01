// Aria Vault : Vaultwarden et son HTTPS dans un seul exe Windows.
//
// Deux modes :
//   - silencieux (--silencieux, utilisé au démarrage de session) : le coffre
//     démarre, Aria reste en icône dans la zone de notification ;
//   - logiciel : la fenêtre s'ouvre. La fermer la renvoie dans la zone de
//     notification, le coffre continue de tourner.
package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"aria-vault/internal/config"
	"aria-vault/internal/demarrage"
	"aria-vault/internal/eveil"
	"aria-vault/internal/fenetre"
	"aria-vault/internal/journal"
	"aria-vault/internal/maj"
	"aria-vault/internal/moteur"
	"aria-vault/internal/paths"
	"aria-vault/internal/releve"
	"aria-vault/internal/sauvegarde"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/icon.ico
var appIcon []byte

//go:embed build/tray-on.ico
var trayOn []byte

//go:embed build/tray-off.ico
var trayOff []byte

// version est fixée à la compilation (-X main.version=…).
var version = "dev"

const argSilencieux = "--silencieux"

func main() {
	silencieux := slices.Contains(os.Args[1:], argSilencieux)

	racine, err := paths.Racine()
	if err != nil {
		log.Fatal(err)
	}
	// 1. Une version plus ancienne encore lancée laisse la place (avant tout
	//    accès au dossier, qu'elle peut encore tenir ouvert).
	releves := releve.Reprendre(filepath.Join(racine, paths.Dossier), version)
	// 2. Ancien dossier AriaVaultwarden renommé en AriaVault si besoin.
	p, migration, errMigration := paths.Migrer(racine)
	if err := os.MkdirAll(p.Base, 0o700); err != nil {
		log.Fatal(err)
	}
	j := journal.New(filepath.Join(p.Base, "aria.log"))
	if silencieux {
		j.Add(journal.Info, "Aria Vault %s lancé en mode silencieux", version)
	} else {
		j.Add(journal.Info, "Aria Vault %s lancé", version)
	}
	for _, m := range releves {
		j.Add(journal.Info, "%s", m)
	}
	if migration != "" {
		j.Add(journal.OK, "%s", migration)
	}
	if errMigration != nil {
		j.Add(journal.Erreur, "%v", errMigration)
	}
	// 3. Une restauration coupée en plein milieu (panne, plantage) est terminée.
	if msg, err := sauvegarde.Reparer(p.Donnees); err != nil {
		j.Add(journal.Erreur, "%v", err)
	} else if msg != "" {
		j.Add(journal.OK, "%s", msg)
	}

	var (
		window  *application.WebviewWindow
		tray    *application.SystemTray
		toggle  *application.MenuItem
		majItem *application.MenuItem
	)
	svc := nouveauService(p, j)
	if cfg, err := config.Load(p.Config); err == nil {
		svc.empecherVeille.Store(cfg.EmpecherVeille)
	}
	ctx, arreter := context.WithCancel(context.Background())

	suivi := nouveauSuivi(p.Base)
	montrer := func() {
		if window == nil {
			return
		}
		suivi.montrer()
	}

	app := application.New(application.Options{
		Name:        "Aria Vault",
		Description: "Ton coffre Vaultwarden en HTTPS, depuis la zone de notification",
		Icon:        appIcon,
		Services:    []application.Service{application.NewService(svc)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "io.github.zedjek.ariavault",
			// Relancer l'exe (double-clic, raccourci) montre la fenêtre déjà ouverte.
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				application.InvokeAsync(montrer)
			},
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
			// Profil WebView2 rangé avec le reste, quel que soit le nom de l'exe.
			WebviewUserDataPath: filepath.Join(p.Base, "webview"),
		},
		OnShutdown: func() {
			arreter()
			// Une sauvegarde ou une restauration en cours se termine d'abord.
			svc.svgMu.Lock()
			defer svc.svgMu.Unlock()
			svc.moteur.Stop()
			_ = eveil.Regler(false)
			releve.Effacer(p.Base)
		},
	})
	svc.app = app

	// Ici, on est l'instance principale (Wails a déjà arrêté les doublons).
	if exe, err := executable(); err == nil {
		if maj.Nettoyer(exe) {
			j.Add(journal.Info, "Ancien exe de la mise à jour précédente supprimé")
		}
		if msg, err := demarrage.Aligner(exe, argSilencieux); err != nil {
			j.Add(journal.Erreur, "Lancement à l'ouverture de session : %v", err)
		} else if msg != "" {
			j.Add(journal.Info, "%s", msg)
		}
	}
	if err := releve.Enregistrer(p.Base, version); err != nil {
		j.Add(journal.Erreur, "instance.json : %v", err)
	}
	if err := releve.EcouterFermeture(func() {
		j.Add(journal.Info, "Une version plus récente prend le relais : fermeture")
		app.Quit()
	}); err != nil {
		j.Add(journal.Erreur, "Relève par une nouvelle version indisponible : %v", err)
	}

	// Icône, info-bulle et menu suivent l'état du coffre.
	majTray := func() {
		if tray == nil {
			return
		}
		s := svc.moteur.Status()
		icon, tip, action := trayOff, "Aria Vault · serveur arrêté", "Démarrer le serveur"
		switch s.Etat {
		case moteur.Actif:
			icon, tip, action = trayOn, "Aria Vault · en ligne : "+s.Domaine, "Arrêter le serveur"
		case moteur.Demarrage:
			tip, action = "Aria Vault · démarrage…", "Arrêter le serveur"
		case moteur.Erreur:
			tip = "Aria Vault · erreur : " + s.Detail
		}
		if r := []rune(tip); len(r) > 120 { // limite de l'info-bulle Windows
			tip = string(r[:119]) + "…"
		}
		tray.SetIcon(icon)
		tray.SetTooltip(tip)
		toggle.SetLabel(action)

		// Mise à jour : l'entrée du menu dit si Aria est à jour.
		m := svc.maj.Info()
		switch {
		case m.Disponible:
			majItem.SetLabel("Installer Aria Vault " + m.Version + "…").SetEnabled(true)
		case m.Version != "" && m.Erreur == "":
			majItem.SetLabel("Aria Vault " + version + " est à jour").SetEnabled(false)
		default:
			majItem.SetLabel("Aria Vault " + version).SetEnabled(false)
		}
	}
	svc.onMaj = func() { application.InvokeAsync(majTray) }
	svc.moteur = moteur.New(p, j, func() {
		svc.appliquerVeille()
		application.InvokeAsync(majTray)
	})
	configure := svc.moteur.Configure()

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Aria Vault",
		Width:     fenetre.LargeurIdeale,
		Height:    fenetre.HauteurIdeale,
		MinWidth:  fenetre.LargeurMin,
		MinHeight: fenetre.HauteurMin,
		// Toujours créée cachée : elle est placée (taille retenue ou adaptée à
		// l'écran) puis affichée au démarrage de l'appli.
		Hidden:           true,
		BackgroundColour: application.NewRGB(3, 8, 18),
		URL:              "/",
		Windows:          application.WindowsWindow{Theme: application.Dark},
	})
	suivi.w = window
	// Fermer la fenêtre la cache : le coffre continue de servir le téléphone.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	menu := app.NewMenu()
	menu.Add("Ouvrir Aria Vault").OnClick(func(*application.Context) { montrer() })
	menu.Add("Ouvrir le coffre dans le navigateur").OnClick(func(*application.Context) {
		if err := svc.OpenVault(); err != nil {
			j.Add(journal.Erreur, "Ouverture du coffre : %v", err)
		}
	})
	menu.AddSeparator()
	toggle = menu.Add("Démarrer le serveur").OnClick(func(*application.Context) {
		go func() {
			switch svc.moteur.Status().Etat {
			case moteur.Actif, moteur.Demarrage:
				svc.moteur.Stop()
			default:
				if err := svc.moteur.Start(); err != nil {
					j.Add(journal.Erreur, "%v", err)
					application.InvokeAsync(montrer)
				}
			}
		}()
	})
	menu.Add("Sauvegarder maintenant").OnClick(func(*application.Context) {
		go func() {
			// Le résultat arrive dans le journal. Sans dossier choisi, on ouvre
			// la page Sauvegardes pour que l'utilisateur le choisisse.
			if _, err := svc.Backup(); errors.Is(err, ErrDossierNonChoisi) {
				application.InvokeAsync(func() {
					montrer()
					app.Event.Emit("page", "sauvegardes")
				})
			}
		}()
	})
	menu.AddSeparator()
	// Installer demande une confirmation : on ouvre la page Réglages.
	majItem = menu.Add("Aria Vault " + version).SetEnabled(false).OnClick(func(*application.Context) {
		montrer()
		app.Event.Emit("page", "reglages")
	})
	menu.Add("Quitter (arrête le serveur)").OnClick(func(*application.Context) { app.Quit() })

	tray = app.SystemTray.New()
	tray.SetMenu(menu)
	tray.OnClick(montrer)
	tray.SetIcon(trayOff)
	tray.SetTooltip("Aria Vault")

	// Le coffre démarre une fois l'appli lancée (l'icône doit exister pour suivre son état).
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		// Sans configuration, rien à servir : la fenêtre s'ouvre même en mode silencieux.
		suivi.placer(app, !(silencieux && configure))
		majTray()
		go svc.boucleQuotidienne(ctx)
		go svc.boucleIP(ctx)
		go svc.boucleMaj(ctx)
		if configure {
			go func() {
				if err := svc.moteur.Start(); err != nil {
					j.Add(journal.Erreur, "%v", err)
				}
			}()
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// executable renvoie le chemin de l'exe, comme le fait Wails pour le démarrage auto.
func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		return r, nil
	}
	return exe, nil
}
