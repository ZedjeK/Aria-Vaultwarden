// Package moteur orchestre le coffre : Vaultwarden en local, puis le relais
// HTTPS. Le démarrage tourne en arrière-plan (le premier certificat peut
// prendre quelques minutes) ; l'interface suit l'état avec Status.
package moteur

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"aria-vault/internal/config"
	"aria-vault/internal/dnsprov"
	"aria-vault/internal/journal"
	"aria-vault/internal/paths"
	"aria-vault/internal/tlsproxy"
	"aria-vault/internal/vaultwarden"
)

// Etat du coffre.
type Etat string

const (
	Arrete    Etat = "arrete"
	Demarrage Etat = "demarrage"
	Actif     Etat = "actif"
	Erreur    Etat = "erreur"
)

// PortLocal est le port de Vaultwarden, joignable uniquement depuis le PC.
const PortLocal = 8000

// Status est l'état affiché par l'interface et le systray.
type Status struct {
	Etat         Etat      `json:"etat"`
	Detail       string    `json:"detail"` // étape en cours ou cause de l'erreur
	Domaine      string    `json:"domaine"`
	Depuis       time.Time `json:"depuis"`       // en ligne depuis
	Certificat   time.Time `json:"certificat"`   // fin de validité du certificat
	Inscriptions bool      `json:"inscriptions"` // valeur en vigueur
}

// Moteur pilote Vaultwarden et le relais HTTPS.
type Moteur struct {
	p        paths.Paths
	j        *journal.Journal
	onChange func()

	mu           sync.Mutex
	gen          int // incrémenté à chaque démarrage ou arrêt : les tâches périmées se taisent
	etat         Etat
	detail       string
	domaine      string
	depuis       time.Time
	inscriptions bool
	cancel       context.CancelFunc
	finished     chan struct{} // fermé quand la tâche en cours a tout arrêté
	proxy        *tlsproxy.Proxy
}

// New prépare le moteur ; onChange est appelé à chaque changement d'état.
func New(p paths.Paths, j *journal.Journal, onChange func()) *Moteur {
	if onChange == nil {
		onChange = func() {}
	}
	return &Moteur{p: p, j: j, onChange: onChange, etat: Arrete}
}

// Status renvoie l'état courant.
func (m *Moteur) Status() Status {
	m.mu.Lock()
	s := Status{
		Etat:         m.etat,
		Detail:       m.detail,
		Domaine:      m.domaine,
		Depuis:       m.depuis,
		Inscriptions: m.inscriptions,
	}
	proxy := m.proxy
	m.mu.Unlock()
	if proxy != nil {
		s.Certificat = proxy.Expiry()
	}
	return s
}

// Start lance le coffre en arrière-plan. Une configuration incomplète est
// refusée tout de suite ; les autres erreurs arrivent par Status.
func (m *Moteur) Start() error {
	cfg, err := config.Load(m.p.Config)
	if err != nil {
		return err
	}
	if err := cfg.Valider(); err != nil {
		return fmt.Errorf("configuration HTTPS incomplète : %w", err)
	}
	ids, err := cfg.Identifiants()
	if err != nil {
		return err
	}
	dns, err := dnsprov.Nouveau(cfg.FournisseurDNS(), ids)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if m.etat == Demarrage || m.etat == Actif {
		m.mu.Unlock()
		return nil
	}
	m.gen++
	gen := m.gen
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	m.cancel, m.finished = cancel, finished
	m.etat, m.detail = Demarrage, "lancement de Vaultwarden"
	m.domaine, m.inscriptions, m.depuis = cfg.Domaine, cfg.Inscriptions, time.Time{}
	m.mu.Unlock()
	m.onChange()

	go m.run(ctx, gen, cfg, dns, finished)
	return nil
}

func (m *Moteur) run(ctx context.Context, gen int, cfg *config.Config, dns dnsprov.Client, finished chan struct{}) {
	defer close(finished)

	m.j.Add(journal.Info, "Démarrage du coffre pour %s", cfg.Domaine)
	vw := vaultwarden.New(m.p.Moteur, m.p.Donnees)
	opts := m.optionsVW(cfg)
	if err := vw.Start(opts, m.j.Logf); err != nil {
		m.fail(gen, "Vaultwarden : %v", err)
		return
	}
	if cfg.Inscriptions {
		m.j.Add(journal.Info, "Inscriptions ouvertes : pense à les refermer une fois ton compte créé")
	}

	m.setDetail(gen, "certificat HTTPS (jusqu'à quelques minutes la première fois)")
	proxy, err := tlsproxy.Start(ctx, tlsproxy.Options{
		Domaine:     cfg.Domaine,
		DNS:         dns,
		Email:       cfg.Email,
		Certificats: m.p.Certificats,
		Ecoute:      ":443",
		Cible:       fmt.Sprintf("http://127.0.0.1:%d", PortLocal),
		Logger:      zap.New(m.j.Core()),
	})
	if err != nil {
		vw.Stop()
		if ctx.Err() == nil {
			m.fail(gen, "HTTPS : %v", err)
		}
		return
	}

	m.mu.Lock()
	if gen != m.gen {
		m.mu.Unlock()
		proxy.Stop()
		vw.Stop()
		return
	}
	m.proxy = proxy
	m.etat, m.detail, m.depuis = Actif, "", time.Now()
	m.mu.Unlock()
	m.j.Add(journal.OK, "Coffre en ligne : https://%s", cfg.Domaine)
	m.onChange()

	select {
	case <-ctx.Done():
	case <-vw.Done():
		m.fail(gen, "Vaultwarden s'est arrêté de lui-même (voir data\\vaultwarden.log)")
	case err := <-proxy.Done():
		m.fail(gen, "le serveur HTTPS s'est arrêté : %v", err)
	}
	proxy.Stop()
	vw.Stop()
}

// optionsVW traduit les réglages d'Aria pour Vaultwarden. Un service dont le
// secret est illisible est désactivé (et signalé) plutôt que d'empêcher le
// coffre de démarrer.
func (m *Moteur) optionsVW(cfg *config.Config) vaultwarden.Options {
	opts := vaultwarden.Options{Domaine: cfg.Domaine, Port: PortLocal, Inscriptions: cfg.Inscriptions, Invitations: cfg.Invitations}
	if cfg.Push != nil {
		if cle, err := cfg.PushCle(); err != nil {
			m.j.Add(journal.Erreur, "Synchronisation instantanée désactivée : %v", err)
		} else {
			uris := config.RegionsPush[cfg.Push.Region]
			opts.Push = &vaultwarden.Push{ID: cfg.Push.ID, Cle: cle, RelayURI: uris[0], IdentityURI: uris[1]}
		}
	}
	if s := cfg.SMTP; s != nil {
		if mdp, err := cfg.SMTPMotDePasse(); err != nil {
			m.j.Add(journal.Erreur, "E-mails désactivés : %v", err)
		} else {
			opts.SMTP = &vaultwarden.SMTP{Hote: s.Hote, Port: s.Port, Securite: s.Securite, Utilisateur: s.Utilisateur, MotDePasse: mdp, De: s.De}
		}
	}
	return opts
}

func (m *Moteur) setDetail(gen int, detail string) {
	m.mu.Lock()
	if gen != m.gen {
		m.mu.Unlock()
		return
	}
	m.detail = detail
	m.mu.Unlock()
	m.onChange()
}

func (m *Moteur) fail(gen int, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	m.mu.Lock()
	if gen != m.gen {
		m.mu.Unlock()
		return
	}
	m.etat, m.detail, m.proxy = Erreur, msg, nil
	m.mu.Unlock()
	m.j.Add(journal.Erreur, "%s", msg)
	m.onChange()
}

// Stop arrête le coffre et attend que les ports soient libérés.
func (m *Moteur) Stop() {
	m.mu.Lock()
	cancel, finished := m.cancel, m.finished
	wasRunning := m.etat == Demarrage || m.etat == Actif
	m.gen++
	m.cancel, m.finished, m.proxy = nil, nil, nil
	m.etat, m.detail, m.depuis = Arrete, "", time.Time{}
	m.mu.Unlock()

	if cancel != nil {
		cancel()
		select {
		case <-finished:
		case <-time.After(20 * time.Second):
		}
	}
	if wasRunning {
		m.j.Add(journal.Info, "Coffre arrêté")
	}
	m.onChange()
}

// Restart arrête puis relance le coffre (le certificat déjà obtenu est
// rechargé depuis le disque, sans nouvelle demande).
func (m *Moteur) Restart() error {
	m.Stop()
	return m.Start()
}

// Configure indique si la configuration permet de démarrer.
func (m *Moteur) Configure() bool {
	cfg, err := config.Load(m.p.Config)
	return err == nil && cfg.Valider() == nil
}
