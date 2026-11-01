<script>
  import { onMount } from 'svelte'
  import Logo from '../lib/Logo.svelte'
  import Icon from '../lib/Icon.svelte'
  import {
    app, formatDate, formatSince, daysLeft, start, stop, openVault, loadBackups, sauvegarderMaintenant, refreshJournal,
  } from '../lib/state.svelte.js'

  onMount(() => {
    loadBackups()
    refreshJournal()
  })

  const st = $derived(app.status)
  const m = $derived(st?.moteur ?? { etat: 'arrete' })
  const etat = $derived(m.etat)
  const configure = $derived(st?.configure ?? false)

  const word = $derived(
    { actif: 'EN LIGNE', demarrage: 'DÉMARRAGE', erreur: 'ERREUR' }[etat] ?? 'HORS LIGNE',
  )
  const tone = $derived({ actif: 'ok', demarrage: 'start', erreur: 'warn' }[etat] ?? 'off')

  const certFin = $derived(formatDate(m.certificat))
  const certJours = $derived(daysLeft(m.certificat))

  const tags = { ok: '[ok]', wait: '[..]', off: '[--]', alert: '[!!]' }
  const running = $derived(etat === 'actif' || etat === 'demarrage')
  const derniere = $derived(app.backups?.liste?.[0])
  const extrait = $derived(app.journal.slice(-6))
  const marks = { ok: '✓', erreur: '✗', info: '›' }
  const lines = $derived([
    {
      id: 'vaultwarden',
      page: 'journal',
      tag: etat === 'actif' ? 'ok' : etat === 'demarrage' ? 'wait' : 'off',
      name: 'vaultwarden',
      detail: `${st?.vaultwarden ? `v${st.vaultwarden} · ` : ''}127.0.0.1:8000, invisible du réseau`,
    },
    {
      id: 'https',
      page: 'https',
      tag: etat === 'actif' ? 'ok' : etat === 'demarrage' ? 'wait' : !configure ? 'alert' : 'off',
      name: 'https',
      detail:
        etat === 'actif' && certFin
          ? `certificat Let's Encrypt jusqu'au ${certFin}`
          : etat === 'demarrage'
            ? m.detail
            : configure
              ? 'port 443, certificat Let\'s Encrypt'
              : 'à configurer',
    },
    ...(configure
      ? [
          {
            id: 'ip',
            page: 'https',
            tag: !st?.ip?.publiee ? 'wait' : st.ip.publiee === st.ip.locale?.ip ? 'ok' : 'alert',
            name: 'adresse ip',
            detail: !st?.ip?.publiee
              ? 'vérification en cours'
              : st.ip.publiee === st.ip.locale?.ip
                ? `${st.ip.publiee}, celle de ce PC`
                : `le domaine pointe vers ${st.ip.publiee}, ce PC est en ${st.ip.locale?.ip || '?'}`,
          },
        ]
      : []),
    {
      id: 'version',
      page: 'reglages',
      tag: !st?.verifMaj ? 'off' : st?.maj?.disponible ? 'alert' : st?.maj?.version ? 'ok' : 'wait',
      name: 'version',
      detail: !st?.verifMaj
        ? `${st?.version ?? ''}, vérification désactivée`
        : st?.maj?.disponible
          ? `${st.version} · la ${st.maj.version} est disponible`
          : st?.maj?.version
            ? `${st.version} · à jour`
            : st?.maj?.erreur
              ? `${st?.version ?? ''} · GitHub injoignable pour l'instant`
              : `${st?.version ?? ''} · vérification en cours`,
    },
    {
      id: 'autostart',
      page: 'reglages',
      tag: st?.autostart ? 'ok' : 'off',
      name: 'démarrage',
      detail: st?.autostart ? 'à l\'ouverture de session, en silence' : 'manuel',
    },
    ...(st?.inscriptions
      ? [{ id: 'inscriptions', page: 'reglages', tag: 'alert', name: 'inscriptions', detail: 'ouvertes : referme-les une fois ton compte créé' }]
      : []),
  ])
</script>

<div class="page">
 <div class="dash">
  <section class="hero" aria-label="État du coffre">
    <div class="hero-text">
      <span class="eyebrow">État du coffre</span>
      <p class="word {tone}">{word}</p>
      <p class="sub">
        {#if etat === 'actif'}
          Ton coffre répond sur https://{m.domaine} · {formatSince(m.depuis)}. Ton téléphone peut se synchroniser.
        {:else if etat === 'demarrage'}
          En cours : {m.detail}…
        {:else if etat === 'erreur'}
          {m.detail}. Détails dans le journal ; relance le serveur depuis Réglages.
        {:else if !configure}
          Commence par la page HTTPS : ton nom de domaine et les identifiants de ton fournisseur DNS suffisent, un tuto te guide.
        {:else}
          Le serveur est arrêté : ton téléphone garde sa copie hors ligne, mais ne peut plus se synchroniser. Démarre-le
          depuis Réglages, ou d'un clic droit sur l'icône Aria dans la zone de notification.
        {/if}
      </p>
    </div>
    <div class="hero-logo" class:on={etat === 'actif'}>
      <Logo size={150} dim={etat !== 'actif'} label="Logo Aria Vault" />
    </div>
  </section>

  <section class="card moteurs" aria-label="Moteurs">
    <h2>Moteurs</h2>
    <div class="lines">
      {#each lines as l (l.id)}
        <button class="line" onclick={() => (app.page = l.page)} title="Ouvrir la page correspondante">
          <span class="tag {l.tag}">{tags[l.tag]}</span>
          <span class="name">{l.name}</span>
          <span class="detail">{l.detail}</span>
        </button>
      {/each}
    </div>
    <div class="facts">
      <span>adresse <strong>{st?.domaine ? `https://${st.domaine}` : '—'}</strong>{#if etat === 'actif'}<span class="cursor" aria-hidden="true"></span>{/if}</span>
      {#if certJours !== null && certJours < 20}
        <span class="warn-text">certificat : {certJours} j restants, le renouvellement devrait déjà avoir eu lieu</span>
      {/if}
    </div>
  </section>

  {#if !configure}
    <section class="card actions-card control" aria-label="Premiers pas">
      <span class="eyebrow">Premiers pas</span>
      <button class="cta" onclick={() => (app.page = 'https')}>configurer https</button>
      <p class="hint">Un nom de domaine et ses identifiants DNS : un tuto te guide pas à pas.</p>
    </section>
  {:else}
    <section class="card actions-card" aria-label="Actions rapides">
      <h2>Actions rapides</h2>
      <div class="quick">
        {#if running}
          <button class="btn-ghost" onclick={stop} disabled={app.busy} title="Arrêter le serveur (ton téléphone ne pourra plus se synchroniser)">
            <span class="q-ico stop" aria-hidden="true">■</span> Arrêter le serveur
          </button>
        {:else}
          <button class="btn-primary" onclick={start} disabled={app.busy} title="Démarrer le serveur">
            <span class="q-ico" aria-hidden="true">▶</span> Démarrer le serveur
          </button>
        {/if}
        <button class="btn-ghost" onclick={sauvegarderMaintenant} disabled={app.busy} title="Sauvegarder le coffre maintenant (Ctrl+S)">
          <Icon name="backup" size={16} /> Sauvegarder <kbd>Ctrl S</kbd>
        </button>
        <button class="btn-ghost" onclick={openVault} disabled={app.busy || etat !== 'actif'} title={etat === 'actif' ? `https://${st.domaine}` : 'Le serveur doit être en ligne'}>
          <Icon name="vault" size={16} /> Ouvrir le coffre ↗
        </button>
      </div>
      <div class="kv">
        <button class="kv-row" onclick={() => (app.page = 'sauvegardes')} title="Page Sauvegardes">
          <span>dernière sauvegarde</span>
          <strong class:warn-text={!derniere}>{derniere ? formatSince(derniere.date).replace('depuis', 'il y a') : app.backups?.dossier ? 'aucune' : 'dossier à choisir'}</strong>
        </button>
        <button class="kv-row" onclick={() => (app.page = 'https')} title="Page HTTPS">
          <span>certificat</span>
          <strong class:warn-text={certJours !== null && certJours < 20}>{certJours !== null ? `${certJours} j restants` : '—'}</strong>
        </button>
        <button class="kv-row" onclick={() => (app.page = 'https')} title="Page HTTPS">
          <span>IP de ce PC</span>
          <strong>{st?.ip?.locale?.ip || '—'}</strong>
        </button>
      </div>
    </section>
  {/if}

  <section class="card journal-card" aria-label="Derniers événements">
    <div class="card-head">
      <h2>Derniers événements</h2>
      <button class="btn-ghost small" onclick={() => (app.page = 'journal')} title="Ouvrir le journal (Ctrl+5)">Tout voir →</button>
    </div>
    {#if extrait.length}
      <ol class="log mini">
        {#each extrait as e, i (i)}
          <li class:fail={e.niveau === 'erreur'}>
            <span>{marks[e.niveau] ?? '›'}</span>
            <span class="time">{e.heure}</span>
            <span>{e.texte}</span>
          </li>
        {/each}
      </ol>
    {:else}
      <p class="card-text">Rien pour l'instant.</p>
    {/if}
  </section>
 </div>
</div>

<style>
  .hero {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 20px;
    align-items: center;
    padding: 22px 26px;
    background: var(--surface);
    border: 1px solid var(--line-strong);
    border-radius: 18px;
    corner-shape: bevel;
  }
  .hero-text {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-width: 0;
  }
  .word {
    margin: 0;
    font-family: var(--font-display);
    font-size: 52px;
    font-weight: 700;
    line-height: 1;
    color: #ffffff;
    overflow-wrap: anywhere;
  }
  .word.ok {
    text-shadow: -3px 0 0 var(--accent), 3px 0 0 var(--accent-2);
  }
  .word.start {
    text-shadow: -3px 0 0 var(--accent-2), 3px 0 0 var(--accent);
    animation: flicker 1.6s steps(1) infinite;
  }
  .word.off {
    text-shadow: -3px 0 0 var(--action), 3px 0 0 var(--accent-2);
  }
  .word.warn {
    color: var(--warn);
    text-shadow: -3px 0 0 var(--danger), 3px 0 0 var(--accent-2);
  }
  @keyframes flicker {
    50% {
      opacity: 0.6;
    }
  }
  .sub {
    margin: 0;
    font-size: 13px;
    line-height: 1.6;
    color: var(--muted);
    max-width: 520px;
    overflow-wrap: anywhere;
  }
  .hero-logo {
    border-radius: 50%;
    padding: 6px;
  }
  .hero-logo.on {
    background: radial-gradient(circle, rgba(125, 249, 255, 0.14) 0%, rgba(125, 249, 255, 0) 70%);
  }

  /* Tableau de bord : 1, 2 puis 3 zones selon la largeur de la fenêtre */
  .dash {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: 'hero' 'actions' 'moteurs' 'journal';
    gap: 16px;
    align-items: start;
  }
  .hero {
    grid-area: hero;
  }
  .moteurs {
    grid-area: moteurs;
  }
  .actions-card {
    grid-area: actions;
  }
  .journal-card {
    grid-area: journal;
  }
  @container (min-width: 720px) {
    .dash {
      grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
      grid-template-areas: 'hero hero' 'moteurs actions' 'journal journal';
    }
  }
  @container (min-width: 1180px) {
    .dash {
      grid-template-columns: minmax(0, 1.5fr) minmax(320px, 1fr);
      grid-template-areas: 'hero actions' 'moteurs journal';
    }
    .hero,
    .actions-card {
      align-self: stretch;
    }
  }
  @container (max-width: 520px) {
    .hero {
      grid-template-columns: minmax(0, 1fr);
    }
    .hero-logo {
      display: none;
    }
    .word {
      font-size: 38px;
    }
  }

  .quick {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .quick button {
    display: flex;
    align-items: center;
    gap: 10px;
    text-align: left;
  }
  .q-ico {
    width: 16px;
    text-align: center;
    font-size: 12px;
  }
  .q-ico.stop {
    color: var(--danger);
  }
  kbd {
    margin-left: auto;
    padding: 1px 6px;
    border: 1px solid var(--line-strong);
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 10px;
    font-weight: 500;
    color: var(--faint);
  }
  .kv {
    display: flex;
    flex-direction: column;
    border-top: 1px solid var(--line-soft);
    padding-top: 6px;
  }
  .kv-row {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    padding: 7px 6px;
    margin: 0 -6px;
    border: none;
    border-radius: 6px 0;
    corner-shape: bevel;
    background: none;
    color: var(--faint);
    font-family: var(--font-mono);
    font-size: 12px;
    text-align: left;
  }
  .kv-row:hover {
    background: var(--hover);
  }
  .kv-row strong {
    color: var(--text);
    font-weight: 500;
    text-align: right;
  }
  .kv-row strong.warn-text {
    color: var(--warn);
  }
  .log.mini {
    padding: 10px;
    font-size: 11.5px;
    gap: 5px;
  }
  .log.mini .time {
    color: var(--faint);
    flex: none;
  }

  .lines {
    display: flex;
    flex-direction: column;
  }
  .line {
    display: grid;
    grid-template-columns: 40px 104px minmax(0, 1fr);
    gap: 8px;
    align-items: baseline;
    min-height: 36px;
    padding: 8px;
    margin: 0 -8px;
    border: none;
    border-radius: 6px 0;
    corner-shape: bevel;
    background: transparent;
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 13px;
    text-align: left;
  }
  .line:hover {
    background: var(--hover);
  }
  .tag {
    white-space: pre;
    color: var(--faint);
  }
  .tag.ok {
    color: var(--accent);
  }
  .tag.wait {
    color: var(--accent-2);
  }
  .tag.alert {
    color: var(--warn);
  }
  .name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .detail {
    color: var(--muted);
    font-size: 12px;
    overflow-wrap: anywhere;
  }
  .facts {
    margin-top: auto;
    padding-top: 10px;
    border-top: 1px solid var(--line-soft);
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 13px;
    color: var(--faint);
  }
  .facts strong {
    margin-left: 6px;
    font-weight: 500;
    color: var(--text);
    overflow-wrap: anywhere;
  }
  .warn-text {
    color: var(--warn);
    font-size: 12px;
  }
  .cursor {
    display: inline-block;
    width: 8px;
    height: 15px;
    margin-left: 6px;
    vertical-align: -2px;
    background: var(--action);
    animation: blink 1.1s steps(1) infinite;
  }
  @keyframes blink {
    50% {
      opacity: 0;
    }
  }

  .control {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 12px;
    min-width: 0;
  }
  .cta {
    width: 100%;
    min-height: 52px;
    border: none;
    border-radius: 12px 0;
    corner-shape: bevel;
    background: var(--action);
    color: var(--on-action);
    font-family: var(--font-mono);
    font-size: 15px;
    font-weight: 600;
  }
  .cta:hover:not(:disabled) {
    background: #ff9d42;
  }
  .cta:disabled {
    opacity: 0.45;
  }
</style>
