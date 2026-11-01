<script>
  import Logo from '../lib/Logo.svelte'
  import { app, formatDate, formatSince, daysLeft } from '../lib/state.svelte.js'

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

  <div class="bottom" class:single={configure}>
    <section class="card" aria-label="Moteurs">
      <h2>Moteurs</h2>
      <div class="lines">
        {#each lines as l (l.id)}
          <button class="line" onclick={() => (app.page = l.page)}>
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
      <section class="control" aria-label="Premiers pas">
        <span class="eyebrow">Premiers pas</span>
        <button class="cta" onclick={() => (app.page = 'https')}>configurer https</button>
        <p class="hint">Un nom de domaine et ses identifiants DNS : un tuto te guide pas à pas.</p>
      </section>
    {/if}
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

  .bottom {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 16px;
  }
  @container (min-width: 640px) {
    .bottom:not(.single) {
      grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr);
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
