<script>
  import Switch from '../lib/Switch.svelte'
  import {
    app, start, stop, setAutostart, setInscriptions, openData, setEmpecherVeille, setVerifMaj, checkUpdate, openRelease,
    installUpdate, formatDateHeure, formatTaille,
  } from '../lib/state.svelte.js'

  const st = $derived(app.status)
  const etat = $derived(st?.moteur?.etat ?? 'arrete')
  const running = $derived(etat === 'actif' || etat === 'demarrage')
  const label = { actif: 'En ligne', demarrage: 'Démarrage…', erreur: 'Erreur', arrete: 'Arrêté' }
  const maj = $derived(st?.maj ?? {})
  let verifiee = $state(false)
  let confirmer = $state(false)

  async function installer() {
    if (!confirmer) {
      confirmer = true
      return
    }
    confirmer = false
    await installUpdate()
  }

  async function verifier() {
    await checkUpdate()
    verifiee = true
  }
</script>

<div class="page">
  <div class="page-head">
    <h1>Réglages</h1>
  </div>

  <div class="grid-3">
    <section class="card">
      <h2>Coffre</h2>
      <div class="row">
        <div class="row-text">
          <span class="row-name">Serveur</span>
          <span class="row-detail">
            <span class="chip" class:ok={etat === 'actif'} class:warn={etat === 'demarrage'} class:fail={etat === 'erreur'}>{label[etat]}</span>
          </span>
        </div>
        {#if running}
          <button class="btn-ghost small" onclick={stop} disabled={app.busy}>{app.busy ? 'Arrêt…' : 'Arrêter'}</button>
        {:else}
          <button class="btn-primary small" onclick={start} disabled={app.busy || !st?.configure}>
            {app.busy ? 'Démarrage…' : etat === 'erreur' ? 'Relancer' : 'Démarrer'}
          </button>
        {/if}
      </div>
      <p class="hint">
        Aussi possible d'un clic droit sur l'icône Aria, dans la zone de notification. Fermer la fenêtre n'arrête pas le
        serveur.
      </p>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Dossier du coffre</span>
          <span class="path">{st?.donnees ?? ''}</span>
        </div>
        <button class="btn-ghost small" onclick={openData} disabled={app.busy}>Ouvrir</button>
      </div>
      <p class="hint">
        Base, clés du serveur et pièces jointes : tout est là. Pour le sauvegarder, passe par la page Sauvegardes (ou
        Ctrl+S) : le serveur est mis en pause le temps de la copie.
      </p>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Autoriser les inscriptions</span>
          <span class="row-detail">À n'ouvrir que le temps de créer un compte.</span>
        </div>
        <Switch checked={st?.inscriptions ?? false} disabled={app.busy} label="Autoriser les inscriptions" onchange={setInscriptions} />
      </div>
      {#if running}
        <p class="hint">Changer ce réglage relance le serveur quelques secondes.</p>
      {/if}
    </section>

    <section class="card">
      <h2>Démarrage</h2>
      <div class="row">
        <div class="row-text">
          <span class="row-name">Lancer à l'ouverture de session</span>
          <span class="row-detail">Aria démarre en silence dans la zone de notification et sert ton coffre.</span>
        </div>
        <Switch checked={st?.autostart ?? false} disabled={app.busy} label="Lancer à l'ouverture de session" onchange={setAutostart} />
      </div>
      <p class="hint">
        Le serveur ne tourne que pendant ta session Windows. Session fermée ou PC en veille : ton téléphone garde sa copie
        hors ligne et se resynchronise au retour.
      </p>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Empêcher la mise en veille</span>
          <span class="row-detail">Tant que le serveur tourne, le PC ne s'endort pas tout seul.</span>
        </div>
        <Switch checked={st?.empecherVeille ?? false} disabled={app.busy} label="Empêcher la mise en veille" onchange={setEmpecherVeille} />
      </div>
      <p class="hint">
        L'écran peut toujours s'éteindre, et tu peux toujours mettre le PC en veille toi-même (menu Démarrer, capot fermé).
        {#if st?.empecherVeille}{st?.veilleBloquee ? 'Actif en ce moment.' : 'Inactif tant que le serveur est arrêté.'}{/if}
      </p>
    </section>

    <section class="card">
      <div class="card-head">
        <h2>Mises à jour</h2>
        {#if maj.disponible}<span class="pill on">{maj.version} disponible</span>{/if}
      </div>
      <div class="row">
        <div class="row-text">
          <span class="row-name">Vérifier les mises à jour</span>
          <span class="row-detail">Une fois par jour, Aria regarde sur GitHub si une nouvelle version est sortie. Rien n'est téléchargé tout seul.</span>
        </div>
        <Switch checked={st?.verifMaj ?? true} disabled={app.busy} label="Vérifier les mises à jour" onchange={setVerifMaj} />
      </div>
      <div class="row">
        <span class="row-name">Version installée</span>
        <span class="value mono">{st?.version ?? ''}</span>
      </div>
      {#if maj.verifie && new Date(maj.verifie).getFullYear() > 2000}
        <div class="row">
          <span class="row-name">Dernière vérification</span>
          <span class="value">{formatDateHeure(maj.verifie)}</span>
        </div>
      {/if}
      {#if maj.erreur}<p class="hint warn-text">{maj.erreur}</p>{/if}
      {#if maj.disponible && maj.installable}
        <p class="hint">
          « Installer » télécharge le zip de la release ({formatTaille(maj.taille)}) depuis le dépôt officiel, vérifie ses
          empreintes SHA-256, remplace l'exe puis lance la nouvelle version : elle prend le relais, ton coffre et tes réglages
          sont conservés. L'ancien exe est gardé de côté jusqu'au lancement suivant.
        </p>
      {:else if maj.disponible}
        <p class="hint">Télécharge le zip de la nouvelle version et lance-la : elle prend le relais toute seule, ton coffre et tes réglages sont conservés.</p>
      {:else if verifiee && !maj.erreur}
        <p class="hint">Tu as la dernière version.</p>
      {/if}
      {#if app.installation !== null}
        <p class="hint">{app.installation < 100 ? `Téléchargement… ${app.installation} %` : 'Installée : la nouvelle version prend le relais.'}</p>
      {/if}
      <div class="actions">
        {#if maj.disponible && maj.installable}
          <button class="btn-primary grow" onclick={installer} onblur={() => (confirmer = false)} disabled={app.busy}>
            {app.installation !== null ? 'Installation…' : confirmer ? `Confirmer l'installation ?` : `Installer la ${maj.version}`}
          </button>
        {/if}
        {#if maj.disponible}
          <button class="btn-ghost grow" onclick={openRelease} disabled={app.busy}>Nouveautés ↗</button>
        {/if}
        <button class="btn-ghost grow" onclick={verifier} disabled={app.busy}>{app.busy && app.installation === null ? 'Vérification…' : 'Vérifier'}</button>
      </div>
    </section>

    <section class="card">
      <h2>Raccourcis clavier</h2>
      <div class="raccourcis">
        <span><kbd>Ctrl</kbd> <kbd>1</kbd> … <kbd>6</kbd></span><span>Accueil, HTTPS, Services, Sauvegardes, Réglages, Journal</span>
        <span><kbd>Ctrl</kbd> <kbd>S</kbd></span><span>Sauvegarder le coffre maintenant</span>
        <span><kbd>Ctrl</kbd> <kbd>R</kbd> ou <kbd>F5</kbd></span><span>Actualiser l'état</span>
      </div>
      <p class="hint">La taille et la position de la fenêtre sont retenues d'un lancement à l'autre. Le bouton en bas de la barre latérale la replie en icônes.</p>
    </section>
  </div>

  {#if app.error}<p class="error" role="alert">{app.error}</p>{/if}
</div>

<style>
  .raccourcis {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: 10px 14px;
    align-items: center;
    font-size: 12px;
    color: var(--muted);
  }
  kbd {
    display: inline-block;
    min-width: 22px;
    padding: 2px 6px;
    border: 1px solid var(--line-strong);
    border-bottom-width: 2px;
    border-radius: 4px;
    background: var(--surface-sunken);
    font-family: var(--font-mono);
    font-size: 11px;
    text-align: center;
    color: var(--text);
  }
  .warn-text {
    color: var(--warn);
  }
  .mono {
    font-family: var(--font-mono);
  }
</style>
