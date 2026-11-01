package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"

	"aria-vault/internal/config"
	"aria-vault/internal/dnsprov"
	"aria-vault/internal/eveil"
	"aria-vault/internal/ipdyn"
	"aria-vault/internal/journal"
	"aria-vault/internal/maj"
	"aria-vault/internal/moteur"
	"aria-vault/internal/paths"
	"aria-vault/internal/vaultwarden"
)

// VaultService expose au frontend les actions d'Aria Vault.
type VaultService struct {
	p      paths.Paths
	j      *journal.Journal
	moteur *moteur.Moteur
	app    *application.App // renseigné après application.New
	ip     *ipdyn.Veilleur
	maj    *maj.Verificateur

	cfgMu sync.Mutex // protège config.json (lecture-modification-écriture)
	svgMu sync.Mutex // une seule sauvegarde ou restauration à la fois

	verifierIP     chan struct{} // demande une vérification de l'IP tout de suite
	verifierMaj    chan struct{}
	empecherVeille atomic.Bool

	installMu sync.Mutex // une seule installation de mise à jour à la fois
	onMaj     func()     // appelé quand l'état des mises à jour change (icône)
}

func nouveauService(p paths.Paths, j *journal.Journal) *VaultService {
	return &VaultService{
		p: p, j: j,
		ip:          ipdyn.NouveauVeilleur(),
		maj:         maj.Nouveau(version),
		verifierIP:  make(chan struct{}, 1),
		verifierMaj: make(chan struct{}, 1),
	}
}

// View est l'état complet affiché par l'interface.
type View struct {
	Moteur         moteur.Status `json:"moteur"`
	Configure      bool          `json:"configure"`
	Fournisseur    string        `json:"fournisseur"`
	Domaine        string        `json:"domaine"`
	Email          string        `json:"email"`
	Identifiants   bool          `json:"identifiants"` // identifiants DNS enregistrés
	Inscriptions   bool          `json:"inscriptions"` // réglage enregistré
	Autostart      bool          `json:"autostart"`
	IPAuto         bool          `json:"ipAuto"`
	IP             ipdyn.Etat    `json:"ip"`
	VerifMaj       bool          `json:"verifMaj"`
	Maj            maj.Info      `json:"maj"`
	EmpecherVeille bool          `json:"empecherVeille"`
	VeilleBloquee  bool          `json:"veilleBloquee"` // en ce moment
	Version        string        `json:"version"`
	Vaultwarden    string        `json:"vaultwarden"`
	Donnees        string        `json:"donnees"`
	Base           string        `json:"base"`
	Erreur         string        `json:"erreur"` // config.json illisible
}

// Status renvoie l'état de l'appli.
func (s *VaultService) Status() View {
	v := View{
		Moteur:        s.moteur.Status(),
		IP:            s.ip.Etat(),
		Maj:           s.maj.Info(),
		VeilleBloquee: eveil.Actif(),
		Version:       version,
		Vaultwarden:   vaultwarden.Version(),
		Donnees:       s.p.Donnees,
		Base:          s.p.Base,
	}
	cfg, err := s.lireConfig()
	if err != nil {
		v.Erreur = err.Error()
	} else {
		v.Configure = cfg.Valider() == nil
		v.Fournisseur, v.Domaine, v.Email, v.Inscriptions = cfg.FournisseurDNS(), cfg.Domaine, cfg.Email, cfg.Inscriptions
		v.Identifiants = cfg.AIdentifiants()
		v.IPAuto, v.VerifMaj, v.EmpecherVeille = !cfg.SansIPAuto, !cfg.SansVerifMaj, cfg.EmpecherVeille
	}
	if s.app != nil {
		v.Autostart, _ = s.app.Autostart.IsEnabled()
	}
	return v
}

// Fournisseurs renvoie les fournisseurs DNS et leurs tutoriels.
func (s *VaultService) Fournisseurs() []dnsprov.Fournisseur { return dnsprov.Liste() }

// Journal renvoie les derniers événements.
func (s *VaultService) Journal() []journal.Entry { return s.j.Entries() }

// ConfigInput est le formulaire de la page HTTPS.
type ConfigInput struct {
	Fournisseur  string            `json:"fournisseur"`
	Domaine      string            `json:"domaine"`
	Identifiants map[string]string `json:"identifiants"` // champ vide = garder la valeur enregistrée
	Email        string            `json:"email"`
}

// SaveResult renvoie les erreurs par champ, ou une erreur générale.
type SaveResult struct {
	Fields map[string]string `json:"fields"`
	Error  string            `json:"error"`
}

// SaveConfig enregistre la configuration HTTPS et relance le coffre s'il tournait.
func (s *VaultService) SaveConfig(in ConfigInput) SaveResult {
	res := SaveResult{Fields: map[string]string{}}
	s.cfgMu.Lock()
	enregistre := false
	defer func() {
		if !enregistre {
			s.cfgMu.Unlock()
		}
	}()
	cfg, err := config.Load(s.p.Config)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	f, ok := dnsprov.Trouver(in.Fournisseur)
	if !ok {
		res.Fields["fournisseur"] = "choisis un fournisseur DNS"
		res.Error = "vérifie les champs signalés"
		return res
	}
	if d, err := dnsprov.NormaliserDomaine(f.ID, in.Domaine); err != nil {
		res.Fields["domaine"] = err.Error()
	} else {
		cfg.Domaine = d
	}

	// Les champs laissés vides gardent leur valeur, si le fournisseur ne change pas.
	ids := map[string]string{}
	var anciens map[string]string
	if cfg.AIdentifiants() && cfg.FournisseurDNS() == f.ID {
		anciens, _ = cfg.Identifiants()
	}
	for _, c := range f.Champs {
		v := strings.TrimSpace(in.Identifiants[c.Cle])
		if v == "" {
			v = anciens[c.Cle]
		}
		ids[c.Cle] = v
	}
	for k, v := range cfg.SetIdentifiants(f.ID, ids) {
		res.Fields[k] = v
	}

	email := strings.TrimSpace(in.Email)
	if email != "" && (!strings.Contains(email, "@") || strings.ContainsAny(email, " \t")) {
		res.Fields["email"] = "adresse e-mail invalide"
	}
	cfg.Email = email
	if len(res.Fields) > 0 {
		res.Error = "vérifie les champs signalés"
		return res
	}

	err = cfg.Save(s.p.Config)
	enregistre = true
	s.cfgMu.Unlock()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	s.j.Add(journal.OK, "Configuration enregistrée pour %s chez %s (identifiants chiffrés pour ton compte Windows)", cfg.Domaine, f.Nom)
	s.demander(s.verifierIP)
	if err := s.relancerSiActif(); err != nil {
		res.Error = err.Error()
	}
	return res
}

// Start démarre le coffre.
func (s *VaultService) Start() error { return s.moteur.Start() }

// Stop arrête le coffre.
func (s *VaultService) Stop() { s.moteur.Stop() }

// Restart relance le coffre.
func (s *VaultService) Restart() error { return s.moteur.Restart() }

// SetInscriptions ouvre ou ferme la création de comptes.
func (s *VaultService) SetInscriptions(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.Inscriptions = on }); err != nil {
		return err
	}
	if on {
		s.j.Add(journal.Info, "Inscriptions ouvertes")
	} else {
		s.j.Add(journal.Info, "Inscriptions fermées")
	}
	return s.relancerSiActif()
}

// SetAutostart active ou non le lancement silencieux à l'ouverture de session.
func (s *VaultService) SetAutostart(on bool) error {
	if on {
		if err := s.app.Autostart.EnableWithOptions(application.AutostartOptions{Arguments: []string{argSilencieux}}); err != nil {
			return err
		}
		s.j.Add(journal.Info, "Lancement à l'ouverture de session activé")
		return nil
	}
	if err := s.app.Autostart.Disable(); err != nil {
		return err
	}
	s.j.Add(journal.Info, "Lancement à l'ouverture de session désactivé")
	return nil
}

// OpenVault ouvre le coffre web dans le navigateur.
func (s *VaultService) OpenVault() error {
	cfg, err := s.lireConfig()
	if err != nil {
		return err
	}
	if cfg.Domaine == "" {
		return fmt.Errorf("aucun domaine configuré")
	}
	return s.app.Browser.OpenURL("https://" + cfg.Domaine)
}

// OpenDataFolder ouvre le dossier du coffre dans l'Explorateur.
func (s *VaultService) OpenDataFolder() error { return s.app.Browser.OpenFile(s.p.Donnees) }

// OpenLink ouvre un lien de tutoriel. Seuls les liens connus sont acceptés.
func (s *VaultService) OpenLink(url string) error {
	for _, f := range dnsprov.Liste() {
		for _, e := range f.Tutoriel {
			if e.Lien != "" && e.Lien == url {
				return s.app.Browser.OpenURL(url)
			}
		}
	}
	for _, l := range liensBox {
		if l == url {
			return s.app.Browser.OpenURL(url)
		}
	}
	return errors.New("lien inconnu")
}

// Interfaces d'administration des box, ouvertes depuis le tutoriel de réservation d'IP.
var liensBox = []string{
	"http://mafreebox.freebox.fr",
	"http://192.168.1.1",
	"https://mabbox.bytel.fr",
}

// relancerSiActif applique un réglage en relançant le coffre s'il tourne.
func (s *VaultService) relancerSiActif() error {
	switch s.moteur.Status().Etat {
	case moteur.Actif, moteur.Demarrage, moteur.Erreur:
		return s.moteur.Restart()
	}
	return nil
}

func (s *VaultService) demander(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

// ---------- IP publiée ----------

// SetIPAuto active ou non la publication automatique de l'IP de ce PC.
func (s *VaultService) SetIPAuto(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.SansIPAuto = !on }); err != nil {
		return err
	}
	if on {
		s.j.Add(journal.Info, "Publication automatique de l'IP de ce PC activée")
	} else {
		s.j.Add(journal.Info, "Publication automatique de l'IP de ce PC désactivée")
	}
	s.demander(s.verifierIP)
	return nil
}

// CheckIP vérifie l'adresse publiée tout de suite.
func (s *VaultService) CheckIP() error {
	_, err := s.passeIP(context.Background())
	return err
}

// passeIP fait une vérification ; le booléen indique si quelque chose a changé.
func (s *VaultService) passeIP(ctx context.Context) (bool, error) {
	cfg, err := s.lireConfig()
	if err != nil {
		return false, err
	}
	var pub *dnsprov.Publieur
	if cfg.Valider() == nil {
		ids, _ := cfg.Identifiants()
		client, err := dnsprov.Nouveau(cfg.FournisseurDNS(), ids)
		if err != nil {
			return false, err
		}
		pub = &dnsprov.Publieur{
			ID: cfg.FournisseurDNS(), Domaine: cfg.Domaine, Client: client,
			Zone: func(ctx context.Context, fqdn string) (string, error) {
				return certmagic.FindZoneByFQDN(ctx, zap.NewNop(), fqdn, nil)
			},
		}
	}
	msg, err := s.ip.Verifier(ctx, pub, !cfg.SansIPAuto)
	if msg != "" {
		s.j.Add(journal.OK, "%s", msg)
	}
	return msg != "", err
}

// boucleIP vérifie l'adresse publiée au démarrage puis toutes les 10 minutes.
func (s *VaultService) boucleIP(ctx context.Context) {
	derniere := ""
	t := time.NewTimer(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-s.verifierIP:
		}
		_, err := s.passeIP(ctx)
		// Une même erreur n'est notée qu'une fois, pas toutes les 10 minutes.
		if msg := fmt.Sprint(err); err != nil && msg != derniere {
			s.j.Add(journal.Erreur, "IP publiée : %v", err)
			derniere = msg
		} else if err == nil {
			derniere = ""
		}
		if !t.Stop() {
			select {
			case <-t.C:
			default:
			}
		}
		t.Reset(ipdyn.Intervalle)
	}
}

// ---------- Mises à jour ----------

// SetVerifMaj active ou non la vérification quotidienne des nouvelles versions.
func (s *VaultService) SetVerifMaj(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.SansVerifMaj = !on }); err != nil {
		return err
	}
	if on {
		s.j.Add(journal.Info, "Vérification des mises à jour activée")
		s.demander(s.verifierMaj)
	} else {
		s.maj.Oublier()
		s.signalerMaj()
		s.j.Add(journal.Info, "Vérification des mises à jour désactivée")
	}
	return nil
}

// CheckUpdate interroge GitHub tout de suite.
func (s *VaultService) CheckUpdate() (maj.Info, error) {
	info, err := s.maj.Verifier(context.Background())
	if err == nil {
		s.annoncerMaj(info)
	}
	s.signalerMaj()
	return info, err
}

func (s *VaultService) signalerMaj() {
	if s.onMaj != nil {
		s.onMaj()
	}
}

// InstallUpdate télécharge la nouvelle version, vérifie ses empreintes SHA-256,
// remplace l'exe puis la lance : elle prend le relais (paquet releve) et
// celle-ci se ferme. En cas d'échec, rien n'est modifié.
func (s *VaultService) InstallUpdate() error {
	if !s.installMu.TryLock() {
		return errors.New("installation déjà en cours")
	}
	defer s.installMu.Unlock()

	info := s.maj.Info()
	switch {
	case version == "dev":
		return errors.New("version de développement : installe la nouvelle version à la main")
	case !info.Disponible:
		return errors.New("aucune nouvelle version à installer")
	case !info.Installable:
		return errors.New("cette version ne s'installe pas automatiquement : ouvre sa page pour la télécharger")
	}
	exe, err := executable()
	if err != nil {
		return err
	}

	s.j.Add(journal.Info, "Téléchargement d'Aria Vault %s (%s)…", info.Version, taille(info.Taille))
	dernier := -1
	progression := func(fait, total int64) {
		if total <= 0 {
			return
		}
		if pct := int(fait * 100 / total); pct != dernier && s.app != nil {
			dernier = pct
			s.app.Event.Emit("maj-progression", pct)
		}
	}
	zip, err := s.maj.Telecharger(context.Background(), info, filepath.Join(s.p.Base, "maj"), progression)
	if err != nil {
		s.j.Add(journal.Erreur, "Mise à jour : %v", err)
		return err
	}
	defer os.Remove(zip)
	s.j.Add(journal.OK, "Zip téléchargé, empreinte SHA-256 vérifiée")

	annuler, err := maj.Installer(zip, exe, info.ExeSHA)
	if err != nil {
		s.j.Add(journal.Erreur, "Mise à jour : %v", err)
		return err
	}
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		if aerr := annuler(); aerr != nil {
			s.j.Add(journal.Erreur, "Retour à l'ancienne version impossible : %v", aerr)
		}
		s.j.Add(journal.Erreur, "La nouvelle version n'a pas démarré, l'ancienne est remise en place : %v", err)
		return err
	}
	_ = cmd.Process.Release()
	s.j.Add(journal.OK, "Aria Vault %s installé (empreinte de l'exe vérifiée) : il prend le relais", info.Version)
	return nil
}

// OpenRelease ouvre la page de la nouvelle version.
func (s *VaultService) OpenRelease() error {
	info := s.maj.Info()
	if info.URL == "" {
		return errors.New("aucune nouvelle version connue")
	}
	return s.app.Browser.OpenURL(info.URL) // lien vérifié par maj.Lire
}

var majAnnoncee atomic.Value // version déjà notée dans le journal

func (s *VaultService) annoncerMaj(info maj.Info) {
	if info.Disponible && majAnnoncee.Swap(info.Version) != info.Version {
		s.j.Add(journal.Info, "Aria Vault %s est disponible (tu as la %s)", info.Version, version)
	}
}

// boucleMaj regarde une fois par jour si une nouvelle version est publiée.
func (s *VaultService) boucleMaj(ctx context.Context) {
	t := time.NewTimer(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-s.verifierMaj:
		}
		if cfg, err := s.lireConfig(); err == nil && !cfg.SansVerifMaj {
			if info, err := s.maj.Verifier(ctx); err == nil {
				s.annoncerMaj(info)
			}
			s.signalerMaj()
		}
		if !t.Stop() {
			select {
			case <-t.C:
			default:
			}
		}
		t.Reset(maj.Intervalle)
	}
}

// ---------- Veille ----------

// SetEmpecherVeille empêche ou non la mise en veille tant que le serveur tourne.
func (s *VaultService) SetEmpecherVeille(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.EmpecherVeille = on }); err != nil {
		return err
	}
	s.empecherVeille.Store(on)
	if on {
		s.j.Add(journal.Info, "Mise en veille automatique empêchée tant que le serveur tourne")
	} else {
		s.j.Add(journal.Info, "Mise en veille automatique de nouveau autorisée")
	}
	s.appliquerVeille()
	return nil
}

// appliquerVeille suit l'état du serveur (appelé à chaque changement).
func (s *VaultService) appliquerVeille() {
	etat := s.moteur.Status().Etat
	tourne := etat == moteur.Actif || etat == moteur.Demarrage
	if err := eveil.Regler(s.empecherVeille.Load() && tourne); err != nil {
		s.j.Add(journal.Erreur, "Blocage de la mise en veille : %v", err)
	}
}
