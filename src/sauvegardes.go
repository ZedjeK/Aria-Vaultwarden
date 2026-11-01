package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aria-vault/internal/chiffre"
	"aria-vault/internal/config"
	"aria-vault/internal/journal"
	"aria-vault/internal/moteur"
	"aria-vault/internal/paths"
	"aria-vault/internal/sauvegarde"
)

// ErrDossierNonChoisi : aucune sauvegarde tant qu'aucun dossier n'est choisi.
var ErrDossierNonChoisi = errors.New("choisis d'abord le dossier des sauvegardes")

// BackupView est l'état de la page Sauvegardes.
type BackupView struct {
	Dossier    string            `json:"dossier"`    // vide tant qu'il n'est pas choisi
	MemeDisque bool              `json:"memeDisque"` // même lecteur que le coffre, hors OneDrive
	Auto       bool              `json:"auto"`
	Derniere   time.Time         `json:"derniere"`
	Liste      []sauvegarde.Info `json:"liste"`
	Garder     int               `json:"garder"`
	Chiffrees  bool              `json:"chiffrees"` // les nouvelles sauvegardes sont chiffrées
	Erreur     string            `json:"erreur"`
}

// Backups renvoie la configuration et la liste des sauvegardes.
func (s *VaultService) Backups() BackupView {
	cfg, err := s.lireConfig()
	v := BackupView{Garder: sauvegarde.Garder}
	if err != nil {
		v.Erreur = err.Error()
		return v
	}
	v.Dossier = cfg.DossierSauvegarde
	v.Auto, v.Derniere = !cfg.SansSauvegardeAuto, cfg.DerniereSauvegarde
	v.Chiffrees = cfg.SauvegardesChiffrees && len(cfg.PhraseChiffree) > 0
	if v.Dossier == "" {
		return v
	}
	v.MemeDisque = paths.MemeDisque(v.Dossier, s.p.Donnees) && !paths.SurOneDrive(v.Dossier)
	if v.Liste, err = sauvegarde.Lister(v.Dossier); err != nil {
		v.Erreur = err.Error()
	}
	return v
}

// Backup sauvegarde le coffre maintenant. Si le serveur tourne, il est
// arrêté le temps de la copie (quelques secondes) puis relancé.
func (s *VaultService) Backup() (sauvegarde.Info, error) {
	s.svgMu.Lock()
	defer s.svgMu.Unlock()

	cfg, err := s.lireConfig()
	if err != nil {
		return sauvegarde.Info{}, err
	}
	dest := cfg.DossierSauvegarde
	if dest == "" {
		return sauvegarde.Info{}, ErrDossierNonChoisi
	}

	phrase := ""
	if cfg.SauvegardesChiffrees {
		if phrase, err = cfg.PhraseSauvegarde(); err != nil {
			s.j.Add(journal.Erreur, "Sauvegarde impossible : %v", err)
			return sauvegarde.Info{}, err
		}
		if phrase == "" {
			err := errors.New("sauvegardes chiffrées activées, mais aucune phrase enregistrée : choisis-la de nouveau")
			s.j.Add(journal.Erreur, "Sauvegarde impossible : %v", err)
			return sauvegarde.Info{}, err
		}
	}

	enLigne := s.pauseServeur("sauvegarde")
	info, err := sauvegarde.Creer(s.p.Donnees, dest, time.Now(), phrase)
	s.repriseServeur(enLigne)
	if err != nil {
		s.j.Add(journal.Erreur, "Sauvegarde impossible : %v", err)
		return sauvegarde.Info{}, err
	}

	if info.Chiffree {
		s.j.Add(journal.OK, "Sauvegarde chiffrée créée : %s (%s)", info.Nom, taille(info.Taille))
	} else {
		s.j.Add(journal.OK, "Sauvegarde créée : %s (%s)", info.Nom, taille(info.Taille))
	}
	if supprimees, err := sauvegarde.Purger(dest, sauvegarde.Garder); err != nil {
		s.j.Add(journal.Erreur, "Rotation des sauvegardes : %v", err)
	} else if len(supprimees) > 0 {
		s.j.Add(journal.Info, "Rotation : %d ancienne(s) sauvegarde(s) supprimée(s)", len(supprimees))
	}
	if err := s.majConfig(func(c *config.Config) { c.DerniereSauvegarde = info.Date }); err != nil {
		s.j.Add(journal.Erreur, "Date de sauvegarde non enregistrée : %v", err)
	}
	return info, nil
}

// RestoreResult : BesoinPhrase demande la phrase d'une sauvegarde chiffrée.
type RestoreResult struct {
	BesoinPhrase bool   `json:"besoinPhrase"`
	Error        string `json:"error"`
}

// Restore remplace le coffre par une sauvegarde de la liste. Pour une
// sauvegarde chiffrée, phrase vide = la phrase enregistrée sur ce PC.
func (s *VaultService) Restore(nom, phrase string) RestoreResult {
	if _, ok := sauvegarde.DateDe(nom); !ok {
		return RestoreResult{Error: "nom de sauvegarde invalide"}
	}
	cfg, err := s.lireConfig()
	if err != nil {
		return RestoreResult{Error: err.Error()}
	}
	dest := cfg.DossierSauvegarde
	if dest == "" {
		return RestoreResult{Error: ErrDossierNonChoisi.Error()}
	}
	enregistree := false
	if sauvegarde.EstChiffree(nom) && phrase == "" {
		phrase, _ = cfg.PhraseSauvegarde()
		if phrase == "" {
			return RestoreResult{BesoinPhrase: true}
		}
		enregistree = true
	}
	err = s.remplacerCoffre("Restauration de "+nom, func(now time.Time) (string, error) {
		return sauvegarde.Restaurer(filepath.Join(dest, nom), s.p.Donnees, now, phrase)
	})
	switch {
	case errors.Is(err, sauvegarde.ErrPhrase) && enregistree:
		return RestoreResult{BesoinPhrase: true, Error: "cette sauvegarde a été chiffrée avec une autre phrase : saisis celle de l'époque"}
	case errors.Is(err, sauvegarde.ErrPhrase):
		return RestoreResult{BesoinPhrase: true, Error: "phrase incorrecte : rien n'a été modifié"}
	case err != nil:
		return RestoreResult{Error: err.Error()}
	}
	return RestoreResult{}
}

// SetBackupEncryption active (avec la phrase) ou désactive le chiffrement des
// nouvelles sauvegardes. Les sauvegardes existantes ne sont pas modifiées.
func (s *VaultService) SetBackupEncryption(on bool, phrase string) error {
	if on {
		if err := chiffre.VerifierPhrase(phrase); err != nil {
			return err
		}
	}
	var errPhrase error
	err := s.majConfig(func(c *config.Config) {
		if !on {
			c.SauvegardesChiffrees = false
			_ = c.SetPhraseSauvegarde("")
			return
		}
		if errPhrase = c.SetPhraseSauvegarde(phrase); errPhrase == nil {
			c.SauvegardesChiffrees = true
		}
	})
	if err == nil {
		err = errPhrase
	}
	if err != nil {
		return err
	}
	if on {
		s.j.Add(journal.OK, "Les prochaines sauvegardes seront chiffrées par ta phrase de passe")
	} else {
		s.j.Add(journal.Info, "Les prochaines sauvegardes ne seront plus chiffrées")
	}
	return nil
}

// ImportVault remplace le coffre par un ancien dossier data choisi par l'utilisateur.
// Renvoie false si l'utilisateur a annulé.
func (s *VaultService) ImportVault() (bool, error) {
	src, err := s.app.Dialog.OpenFile().
		SetTitle("Choisis l'ancien dossier data (celui qui contient db.sqlite3)").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		PromptForSingleSelection()
	if err != nil || src == "" {
		return false, err
	}
	return true, s.remplacerCoffre("Import de "+src, func(now time.Time) (string, error) {
		return sauvegarde.Importer(src, s.p.Donnees, now)
	})
}

func (s *VaultService) remplacerCoffre(titre string, fn func(time.Time) (string, error)) error {
	s.svgMu.Lock()
	defer s.svgMu.Unlock()

	enLigne := s.pauseServeur("restauration")
	ancien, err := fn(time.Now())
	if err != nil {
		s.j.Add(journal.Erreur, "%s : %v", titre, err)
		s.repriseServeur(enLigne)
		return err
	}
	if ancien != "" {
		s.j.Add(journal.Info, "Ancien coffre mis de côté : %s", ancien)
	}
	s.j.Add(journal.OK, "%s terminée", titre)
	// Après une restauration, le serveur repart s'il est configuré, même s'il était arrêté.
	s.repriseServeur(enLigne || s.moteur.Configure())
	return nil
}

// pauseServeur arrête le serveur s'il tourne et indique s'il faudra le relancer.
func (s *VaultService) pauseServeur(motif string) bool {
	switch s.moteur.Status().Etat {
	case moteur.Actif, moteur.Demarrage:
		s.j.Add(journal.Info, "Serveur mis en pause pour la %s", motif)
		s.moteur.Stop()
		return true
	}
	return false
}

func (s *VaultService) repriseServeur(relancer bool) {
	if !relancer {
		return
	}
	if err := s.moteur.Start(); err != nil {
		s.j.Add(journal.Erreur, "Relance du serveur : %v", err)
	}
}

// ChooseBackupFolder choisit le dossier des sauvegardes. Renvoie false si annulé.
func (s *VaultService) ChooseBackupFolder() (bool, error) {
	cfg, err := s.lireConfig()
	if err != nil {
		return false, err
	}
	actuel := cfg.DossierSauvegarde
	if actuel == "" {
		actuel = paths.Documents()
	}
	dir, err := s.app.Dialog.OpenFile().
		SetTitle("Dossier des sauvegardes (idéalement OneDrive ou un autre disque)").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		SetDirectory(paths.Existant(actuel)).
		PromptForSingleSelection()
	if err != nil || dir == "" {
		return false, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}
	if err := s.majConfig(func(c *config.Config) { c.DossierSauvegarde = dir }); err != nil {
		return false, err
	}
	s.j.Add(journal.Info, "Dossier des sauvegardes : %s", dir)
	return true, nil
}

// OpenBackupFolder ouvre le dossier des sauvegardes dans l'Explorateur.
func (s *VaultService) OpenBackupFolder() error {
	cfg, err := s.lireConfig()
	if err != nil {
		return err
	}
	if cfg.DossierSauvegarde == "" {
		return ErrDossierNonChoisi
	}
	if err := os.MkdirAll(cfg.DossierSauvegarde, 0o700); err != nil {
		return err
	}
	return s.app.Browser.OpenFile(cfg.DossierSauvegarde)
}

// SetDailyBackup active ou non la sauvegarde quotidienne.
func (s *VaultService) SetDailyBackup(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.SansSauvegardeAuto = !on }); err != nil {
		return err
	}
	if on {
		s.j.Add(journal.Info, "Sauvegarde quotidienne activée")
	} else {
		s.j.Add(journal.Info, "Sauvegarde quotidienne désactivée")
	}
	return nil
}

// boucleQuotidienne sauvegarde le coffre une fois par 24 h tant qu'Aria tourne.
func (s *VaultService) boucleQuotidienne(ctx context.Context) {
	verifier := func() {
		cfg, err := s.lireConfig()
		if err != nil || cfg.SansSauvegardeAuto || cfg.DossierSauvegarde == "" || !s.moteur.Configure() {
			return // pas de dossier choisi : la page Sauvegardes le signale
		}
		if time.Since(cfg.DerniereSauvegarde) < 24*time.Hour {
			return
		}
		if _, err := os.Stat(filepath.Join(s.p.Donnees, "db.sqlite3")); err != nil {
			return // pas encore de coffre
		}
		s.j.Add(journal.Info, "Sauvegarde quotidienne")
		_, _ = s.Backup()
	}
	// Premier passage une fois le serveur en ligne, puis toutes les 30 min.
	t := time.NewTimer(2 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			verifier()
			t.Reset(30 * time.Minute)
		}
	}
}

// ---------- Réglages d'Aria (fournisseur, domaine, e-mail, identifiants) ----------

// ExportSettings enregistre les réglages dans un fichier choisi par l'utilisateur.
// Renvoie false si annulé.
func (s *VaultService) ExportSettings(phrase string) (bool, error) {
	cfg, err := s.lireConfig()
	if err != nil {
		return false, err
	}
	data, err := cfg.Exporter(phrase)
	if err != nil {
		return false, err
	}
	dir := cfg.DossierSauvegarde
	if dir == "" {
		dir = paths.Documents()
	}
	dst, err := s.app.Dialog.SaveFile().
		SetMessage("Enregistrer les réglages d'Aria Vault").
		SetDirectory(paths.Existant(dir)).
		SetFilename("reglages"+config.ExtensionExport).
		AddFilter("Réglages Aria Vault", "*"+config.ExtensionExport).
		PromptForSingleSelection()
	if err != nil || dst == "" {
		return false, err
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return false, err
	}
	s.j.Add(journal.OK, "Réglages exportés : %s (fichier entièrement chiffré par ta phrase de passe)", dst)
	return true, nil
}

// ImportResult décrit le résultat d'un import de réglages.
type ImportResult struct {
	Annule  bool   `json:"annule"`
	Domaine string `json:"domaine"`
	Error   string `json:"error"`
}

// ImportSettings déchiffre un fichier de réglages avec sa phrase de passe
// (obligatoire) et l'applique. Le serveur est relancé s'il tournait.
func (s *VaultService) ImportSettings(phrase string) ImportResult {
	if phrase == "" {
		return ImportResult{Error: "la phrase de passe du fichier est obligatoire"}
	}
	src, err := s.app.Dialog.OpenFile().
		SetTitle("Fichier de réglages Aria Vault").
		AddFilter("Réglages Aria Vault", "*"+config.ExtensionExport).
		PromptForSingleSelection()
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	if src == "" {
		return ImportResult{Annule: true}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	imp, err := config.LireExport(data, phrase)
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	// Tout est validé et rechiffré (DPAPI, pour ce compte) avant d'être écrit :
	// rien n'est modifié en cas d'erreur.
	var errApp error
	err = s.majConfigSi(func(c *config.Config) bool {
		errApp = c.Appliquer(imp)
		return errApp == nil
	})
	if err == nil {
		err = errApp
	}
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	s.j.Add(journal.OK, "Réglages importés pour %s, identifiants DNS compris%s", imp.Domaine, servicesImportes(imp))
	s.demander(s.verifierIP)
	if err := s.relancerSiActif(); err != nil {
		return ImportResult{Domaine: imp.Domaine, Error: err.Error()}
	}
	return ImportResult{Domaine: imp.Domaine}
}

// ---------- Accès à config.json ----------

func (s *VaultService) lireConfig() (*config.Config, error) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	return config.Load(s.p.Config)
}

// majConfig lit, modifie et réenregistre config.json sans course entre appels.
func (s *VaultService) majConfig(fn func(*config.Config)) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	cfg, err := config.Load(s.p.Config)
	if err != nil {
		return err
	}
	fn(cfg)
	return cfg.Save(s.p.Config)
}

func taille(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f Mo", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f Ko", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d o", n)
}

func servicesImportes(imp config.Importe) string {
	switch {
	case imp.Push != nil && imp.SMTP != nil:
		return ", avec la synchronisation instantanée et les e-mails"
	case imp.Push != nil:
		return ", avec la synchronisation instantanée"
	case imp.SMTP != nil:
		return ", avec les e-mails"
	}
	return ""
}
