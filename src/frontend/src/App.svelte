<script>
  import { onMount } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import Icon from './lib/Icon.svelte'
  import Logo from './lib/Logo.svelte'
  import {
    app, refresh, refreshJournal, startPolling, openVault, openRelease, installUpdate, sauvegarderMaintenant, loadBackups,
  } from './lib/state.svelte.js'
  import Home from './pages/Home.svelte'
  import Https from './pages/Https.svelte'
  import Reglages from './pages/Reglages.svelte'
  import Sauvegardes from './pages/Sauvegardes.svelte'
  import Services from './pages/Services.svelte'
  import Journal from './pages/Journal.svelte'

  const nav = [
    { id: 'home', label: 'Accueil', cmd: 'accueil', icon: 'home' },
    { id: 'https', label: 'HTTPS', cmd: 'https', icon: 'lock' },
    { id: 'services', label: 'Services', cmd: 'services', icon: 'services' },
    { id: 'sauvegardes', label: 'Sauvegardes', cmd: 'sauvegardes', icon: 'backup' },
    { id: 'reglages', label: 'Réglages', cmd: 'réglages', icon: 'settings' },
    { id: 'journal', label: 'Journal', cmd: 'journal', icon: 'journal' },
  ]

  const etat = $derived(app.status?.moteur?.etat ?? 'arrete')
  const version = $derived(app.status?.version ?? '')

  // Barre latérale : repliée à la main (retenu sur ce PC) ou d'office si la fenêtre est étroite.
  const cle = 'aria.barre-repliee'
  let replieManuel = $state(lire())
  let etroit = $state(false)
  const replie = $derived(replieManuel || etroit)

  function lire() {
    try {
      return localStorage.getItem(cle) === '1'
    } catch {
      return false
    }
  }
  function basculer() {
    replieManuel = !replieManuel
    try {
      localStorage.setItem(cle, replieManuel ? '1' : '0')
    } catch {}
  }

  function aller(id) {
    app.page = id
    if (id === 'journal') refreshJournal()
    if (id === 'sauvegardes') loadBackups()
  }

  // Raccourcis clavier : Ctrl+1…6 pour les pages, Ctrl+S pour sauvegarder,
  // Ctrl+R ou F5 pour actualiser l'état (au lieu de recharger l'interface).
  function clavier(e) {
    const ctrl = e.ctrlKey || e.metaKey
    if (ctrl && !e.shiftKey && !e.altKey && /^[1-6]$/.test(e.key)) {
      e.preventDefault()
      aller(nav[Number(e.key) - 1].id)
    } else if (ctrl && e.key.toLowerCase() === 's') {
      e.preventDefault()
      if (!app.busy) sauvegarderMaintenant()
    } else if ((ctrl && e.key.toLowerCase() === 'r') || e.key === 'F5') {
      e.preventDefault()
      refresh()
      refreshJournal()
      if (app.page === 'sauvegardes' || app.page === 'home') loadBackups()
    }
  }

  onMount(() => {
    refresh()
    refreshJournal()
    startPolling()
    const mq = matchMedia('(max-width: 760px)')
    etroit = mq.matches
    const suivre = (e) => (etroit = e.matches)
    mq.addEventListener('change', suivre)
    // Le Go peut demander d'afficher une page (ex. sauvegarde lancée depuis l'icône sans dossier).
    const off1 = Events.On('page', (e) => {
      if (nav.some((n) => n.id === e.data)) app.page = e.data
    })
    // Progression du téléchargement d'une mise à jour
    const off2 = Events.On('maj-progression', (e) => {
      if (app.installation !== null) app.installation = Number(e.data) || 0
    })
    return () => {
      off1?.()
      off2?.()
      mq.removeEventListener('change', suivre)
    }
  })
</script>

<svelte:window onkeydown={clavier} />

<div class="shell" class:replie>
  <nav class="sidebar" aria-label="Sections">
    <div class="brand">
      <Logo size={56} dim={etat !== 'actif'} />
      <span class="brand-name label"><span class="brand-top">ARIA</span><span class="brand-sub">VAULT</span></span>
    </div>

    <ul class="nav">
      {#each nav as item, i (item.id)}
        {@const lit = (item.id === 'home' && etat === 'actif') || (item.id === 'journal' && etat === 'erreur')}
        <li>
          <button
            class="nav-item"
            class:active={app.page === item.id}
            aria-current={app.page === item.id ? 'page' : undefined}
            aria-label={item.label}
            title={`${item.label} (Ctrl+${i + 1})`}
            onclick={() => aller(item.id)}
          >
            <span class="nav-icon">
              <Icon name={item.icon} />
              {#if lit}<span class="dot icon-dot" class:alert={etat === 'erreur'} aria-hidden="true"></span>{/if}
            </span>
            <span class="prompt label" aria-hidden="true">&gt;</span>
            <span class="label">{item.cmd}</span>
            {#if lit}<span class="dot label" class:alert={etat === 'erreur'} aria-hidden="true"></span>{/if}
          </button>
        </li>
      {/each}
    </ul>

    <div class="sidebar-foot">
      <button
        class="open"
        class:live={etat === 'actif'}
        disabled={etat !== 'actif' || app.busy}
        aria-label="Ouvrir le coffre dans le navigateur"
        title={etat === 'actif' ? `https://${app.status.domaine}` : 'Le serveur doit être en ligne'}
        onclick={openVault}
      >
        <span class="open-icon"><Icon name="vault" size={22} /></span>
        <span class="open-text label">
          <span class="open-title">Ouvrir le coffre</span>
          <span class="open-sub">{etat === 'actif' ? app.status.domaine : 'serveur hors ligne'}</span>
        </span>
        <span class="open-arrow label" aria-hidden="true">↗</span>
      </button>
      <div class="foot-row">
        <span class="version label">aria-vault {version ? `v${version}` : ''}</span>
        {#if !etroit}
          <button class="plier" onclick={basculer} title={replieManuel ? 'Déplier la barre latérale' : 'Replier la barre latérale'} aria-label={replieManuel ? 'Déplier la barre latérale' : 'Replier la barre latérale'}>
            <Icon name={replieManuel ? 'deplier' : 'replier'} size={16} />
          </button>
        {/if}
      </div>
    </div>
  </nav>

  <main class="content">
    {#if app.loadError}
      <p class="banner" role="alert">Impossible de lire l'état d'Aria : {app.loadError}</p>
    {/if}
    {#if app.status?.erreur}
      <p class="banner" role="alert">{app.status.erreur}</p>
    {/if}
    {#if app.status?.maj?.disponible}
      <div class="maj" role="status">
        {#if app.installation !== null}
          <span class="maj-texte">
            {app.installation < 100 ? `Téléchargement d'Aria Vault ${app.status.maj.version}… ${app.installation} %` : `Aria Vault ${app.status.maj.version} installé : il prend le relais, cette fenêtre va se fermer.`}
          </span>
          <span class="barre" aria-hidden="true"><span style:width="{app.installation}%"></span></span>
        {:else}
          <span><strong>Aria Vault {app.status.maj.version}</strong> est disponible · tu as la {app.status.version}</span>
          <span class="maj-btns">
            <button class="btn-ghost small" onclick={openRelease} disabled={app.busy}>Nouveautés ↗</button>
            {#if app.status.maj.installable}
              <button class="btn-primary small" onclick={installUpdate} disabled={app.busy}>Installer</button>
            {/if}
          </span>
        {/if}
      </div>
    {/if}

    {#if app.page === 'home'}
      <Home />
    {:else if app.page === 'https'}
      <Https />
    {:else if app.page === 'services'}
      <Services />
    {:else if app.page === 'sauvegardes'}
      <Sauvegardes />
    {:else if app.page === 'reglages'}
      <Reglages />
    {:else if app.page === 'journal'}
      <Journal />
    {/if}
  </main>
</div>

<style>
  .shell {
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr);
    height: 100vh;
  }

  .sidebar {
    background: var(--sidebar);
    border-right: 1px solid var(--line-soft);
    padding: 18px 12px;
    display: flex;
    flex-direction: column;
    gap: 20px;
    min-width: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 2px 4px 6px;
    min-height: 58px;
  }
  .brand-name {
    display: flex;
    flex-direction: column;
    line-height: 1.15;
  }
  .brand-top {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 26px;
    color: #ffffff;
    letter-spacing: 1px;
    text-shadow: -2px 0 0 color-mix(in srgb, var(--accent) 55%, transparent), 2px 0 0 color-mix(in srgb, var(--accent-2) 55%, transparent);
  }
  .brand-sub {
    font-size: 14px;
    letter-spacing: 5px;
    color: var(--accent);
  }

  .nav {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .nav-item {
    width: 100%;
    min-height: 42px;
    padding: 0 10px;
    display: flex;
    align-items: center;
    gap: 8px;
    border: none;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: transparent;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 14px;
    font-weight: 500;
    text-align: left;
  }
  .nav-item:hover:not(:disabled) {
    background: var(--hover);
    color: var(--text);
  }
  .nav-item.active {
    background: #0b1a30;
    color: var(--accent);
  }
  .prompt {
    color: var(--dim);
  }
  .nav-item.active .prompt {
    color: var(--action);
  }
  /* En grand, la console suffit : les icônes servent quand la barre est repliée */
  .nav-icon {
    position: relative;
    display: none;
    flex: none;
  }
  .dot {
    flex: none;
    margin-left: auto;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 8px var(--accent);
  }
  .dot.alert {
    background: var(--danger);
    box-shadow: 0 0 8px var(--danger);
  }
  .icon-dot {
    position: absolute;
    right: -3px;
    top: -3px;
    margin: 0;
    border: 2px solid var(--sidebar);
    width: 9px;
    height: 9px;
  }
  .label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sidebar-foot {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .open {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 58px;
    padding: 8px 12px;
    border: 1px solid var(--line-strong);
    border-radius: 12px 0;
    corner-shape: bevel;
    background: var(--surface);
    color: var(--muted);
    text-align: left;
    transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;
  }
  .open.live {
    border-color: color-mix(in srgb, var(--accent) 55%, transparent);
    background: linear-gradient(135deg, color-mix(in srgb, var(--accent) 12%, var(--surface)), color-mix(in srgb, var(--accent-2) 10%, var(--surface)));
    color: var(--text);
  }
  .open.live:hover {
    border-color: var(--accent);
    box-shadow: 0 0 18px color-mix(in srgb, var(--accent) 35%, transparent), inset 0 0 12px color-mix(in srgb, var(--accent) 12%, transparent);
  }
  .open:disabled {
    cursor: not-allowed;
  }
  .open-icon {
    flex: none;
    display: flex;
    color: var(--faint);
  }
  .open.live .open-icon {
    color: var(--accent);
    filter: drop-shadow(0 0 4px var(--accent));
  }
  .open-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .open-title {
    font-family: var(--font-display);
    font-size: 15px;
    font-weight: 600;
  }
  .open-sub {
    font-family: var(--font-mono);
    font-size: 10.5px;
    color: var(--faint);
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .open-arrow {
    flex: none;
    font-size: 18px;
    color: var(--dim);
    transition: transform 0.2s, color 0.2s;
  }
  .open.live .open-arrow {
    color: var(--action);
  }
  .open.live:hover .open-arrow {
    transform: translate(2px, -2px);
  }
  .version {
    padding: 0 4px;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--faint);
  }

  .maj {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 10px;
    width: 100%;
    max-width: 1600px;
    margin: 0 auto;
    padding: 10px 12px 10px 14px;
    border: 1px solid color-mix(in srgb, var(--accent) 45%, transparent);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--accent) 7%, transparent);
    font-size: 13px;
    color: var(--muted);
  }
  .maj strong {
    color: var(--accent);
    font-weight: 600;
  }
  .maj-btns {
    display: flex;
    gap: 8px;
  }
  .maj-texte {
    color: var(--text);
  }
  .barre {
    flex-basis: 100%;
    height: 4px;
    border-radius: 2px;
    background: var(--line-soft);
    overflow: hidden;
  }
  .barre span {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--accent-2), var(--accent));
    box-shadow: 0 0 8px var(--accent);
    transition: width 0.2s;
  }

  .content {
    overflow-y: auto;
    padding: 26px 30px 30px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .foot-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
  }
  .plier {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border: 1px solid transparent;
    border-radius: 6px 0;
    corner-shape: bevel;
    background: transparent;
    color: var(--faint);
  }
  .plier:hover {
    border-color: var(--line-strong);
    background: var(--hover);
    color: var(--accent);
  }

  /* Barre latérale repliée en icônes : à la main, ou d'office si la fenêtre est étroite */
  .shell {
    transition: grid-template-columns 0.18s ease;
  }
  .shell.replie {
    grid-template-columns: 68px minmax(0, 1fr);
  }
  .replie .sidebar {
    padding: 16px 8px;
    align-items: center;
  }
  .replie .brand {
    padding: 0;
    justify-content: center;
  }
  .replie .label {
    display: none;
  }
  .replie .nav-icon {
    display: flex;
  }
  .replie .nav-item,
  .replie .open {
    justify-content: center;
    padding: 0;
    width: 48px;
    min-height: 48px;
  }
  .replie .brand :global(svg) {
    width: 44px;
    height: 44px;
  }
  .replie .nav {
    align-items: center;
  }
  .replie .foot-row {
    justify-content: center;
  }
  @media (max-width: 760px) {
    .content {
      padding: 20px;
    }
  }
</style>
