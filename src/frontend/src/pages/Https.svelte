<script>
  import { onMount } from 'svelte'
  import Icon from '../lib/Icon.svelte'
  import Switch from '../lib/Switch.svelte'
  import TutoBox from '../lib/TutoBox.svelte'
  import {
    app, saveConfig, loadFournisseurs, setIPAuto, checkIP, openLink, formatDate, formatHeure,
  } from '../lib/state.svelte.js'

  const st = $derived(app.status)
  const m = $derived(st?.moteur ?? { etat: 'arrete' })
  const configure = $derived(st?.configure ?? false)
  const ip = $derived(st?.ip ?? {})

  let editing = $state(false)
  let visibles = $state({}) // champs secrets affichés en clair
  let fields = $state({})
  let form = $state({ fournisseur: 'duckdns', domaine: '', email: '', identifiants: {} })

  const showForm = $derived(!configure || editing)
  // Le tutoriel suit le fournisseur choisi dans le formulaire, sinon celui enregistré.
  const fid = $derived(showForm ? form.fournisseur : (st?.fournisseur ?? 'duckdns'))
  const f = $derived(app.fournisseurs.find((x) => x.id === fid))
  const fEnregistre = $derived(app.fournisseurs.find((x) => x.id === st?.fournisseur))
  // Les secrets enregistrés ne sont gardés que pour le même fournisseur.
  const garde = $derived(st?.identifiants && st?.fournisseur === form.fournisseur)

  const ipOK = $derived(!!ip.publiee && ip.publiee === ip.locale?.ip)

  onMount(loadFournisseurs)

  function quand(iso) {
    const h = formatHeure(iso)
    if (!h) return '—'
    return formatDate(iso) === formatDate(new Date().toISOString()) ? `aujourd'hui à ${h}` : `${formatDate(iso)} à ${h}`
  }

  // Premier passage sans configuration : le formulaire part de l'état enregistré.
  let initialise = false
  $effect(() => {
    if (!initialise && st) {
      initialise = true
      form.fournisseur = st.fournisseur || 'duckdns'
      form.domaine = st.domaine ?? ''
      form.email = st.email ?? ''
    }
  })

  function startEdit() {
    form = { fournisseur: st?.fournisseur || 'duckdns', domaine: st?.domaine ?? '', email: st?.email ?? '', identifiants: {} }
    fields = {}
    editing = true
  }

  function choisir(id) {
    if (form.fournisseur === id) return
    form.fournisseur = id
    form.identifiants = {}
    fields = {}
  }

  async function submit(event) {
    event.preventDefault()
    const res = await saveConfig({ ...form, identifiants: { ...form.identifiants } })
    fields = res?.fields ?? {}
    if (!res?.error) {
      editing = false
      visibles = {}
      form.identifiants = {}
    }
  }
</script>

<div class="page">
  <div class="page-head">
    <h1>HTTPS</h1>
    <span class="pill" class:on={m.etat === 'actif'}>{m.etat === 'actif' ? 'Certificat actif' : 'Hors ligne'}</span>
  </div>
  <p class="page-intro">
    Aria obtient et renouvelle seule un vrai certificat Let's Encrypt pour ton nom de domaine, en prouvant qu'il t'appartient
    auprès de ton fournisseur DNS. L'appli Bitwarden pour Android refuse les certificats faits maison : c'est ce qui la rend
    compatible.
  </p>

  <div class="https-grid">
    <section class="card">
      <div class="card-head">
        <h2>Domaine</h2>
        {#if configure && !editing}<span class="pill on">Enregistré</span>{/if}
      </div>

      {#if showForm}
        <form class="form" onsubmit={submit}>
          <div class="field">
            <span class="field-label">Fournisseur DNS</span>
            <div class="seg" role="radiogroup" aria-label="Fournisseur DNS">
              {#each app.fournisseurs as x (x.id)}
                <button type="button" role="radio" aria-checked={form.fournisseur === x.id} class:on={form.fournisseur === x.id} onclick={() => choisir(x.id)}>
                  {x.nom}
                </button>
              {/each}
            </div>
            {#if f}<span class="row-detail">{f.resume}</span>{/if}
            {#if fields.fournisseur}<span class="field-error">{fields.fournisseur}</span>{/if}
          </div>

          <label class="field">
            <span class="field-label">Domaine du coffre</span>
            <input bind:value={form.domaine} placeholder={`Ex. ${f?.exemple ?? 'vault.exemple.fr'}`} autocomplete="off" spellcheck="false" />
            {#if fields.domaine}<span class="field-error">{fields.domaine}</span>{/if}
          </label>

          {#each f?.champs ?? [] as c (c.cle)}
            <label class="field">
              <span class="field-label">{c.libelle}</span>
              {#if c.choix?.length}
                <select bind:value={form.identifiants[c.cle]}>
                  {#each c.choix as o (o)}<option value={o} selected={o === c.defaut}>{o}</option>{/each}
                </select>
              {:else}
                <span class="with-btn">
                  <input
                    type={c.secret && !visibles[c.cle] ? 'password' : 'text'}
                    bind:value={form.identifiants[c.cle]}
                    placeholder={garde ? '•••••••• (laisse vide pour garder l\'actuel)' : c.placeholder}
                    autocomplete="off"
                    spellcheck="false"
                  />
                  {#if c.secret}
                    <button type="button" class="btn-ghost small" onclick={() => (visibles[c.cle] = !visibles[c.cle])}>
                      {visibles[c.cle] ? 'Masquer' : 'Afficher'}
                    </button>
                  {/if}
                </span>
              {/if}
              {#if fields[c.cle]}<span class="field-error">{fields[c.cle]}</span>{/if}
            </label>
          {/each}

          <label class="field">
            <span class="field-label">E-mail <span class="optional">(facultatif)</span></span>
            <input bind:value={form.email} placeholder="Pour les avis de Let's Encrypt" autocomplete="off" spellcheck="false" />
            {#if fields.email}<span class="field-error">{fields.email}</span>{/if}
          </label>

          <p class="hint">Les identifiants sont chiffrés avec DPAPI : seul ton compte Windows peut les relire.</p>

          {#if app.error && !Object.keys(fields).length}<p class="error" role="alert">{app.error}</p>{/if}

          <div class="actions">
            <button type="submit" class="btn-primary grow" disabled={app.busy || !f}>
              {app.busy ? 'Enregistrement…' : m.etat === 'arrete' ? 'Enregistrer' : 'Enregistrer et relancer'}
            </button>
            {#if editing}
              <button type="button" class="btn-ghost" onclick={() => (editing = false)} disabled={app.busy}>Annuler</button>
            {/if}
          </div>
        </form>
      {:else}
        <div class="row"><span class="row-name">Fournisseur</span><span class="value">{fEnregistre?.nom ?? st.fournisseur}</span></div>
        <div class="row"><span class="row-name">Domaine</span><span class="value mono">{st.domaine}</span></div>
        <div class="row"><span class="row-name">Identifiants</span><span class="value mono">•••••••• chiffrés</span></div>
        <div class="row"><span class="row-name">E-mail</span><span class="value">{st.email || '—'}</span></div>
        <div class="actions">
          <button class="btn-ghost grow" onclick={startEdit} disabled={app.busy}>Modifier</button>
        </div>
      {/if}
    </section>

    <section class="card tuto">
      <div class="card-head">
        <h2>Tuto {f?.nom ?? ''}</h2>
        <span class="chip">{f?.tutoriel?.length ?? 0} étapes</span>
      </div>
      {#if f}
        <ol class="steps">
          {#each f.tutoriel as e, i (i)}
            <li>
              {e.texte}
              {#if e.lien}
                <button class="link" onclick={() => openLink(e.lien)}>
                  {new URL(e.lien).host}<Icon name="external" size={13} />
                </button>
              {/if}
            </li>
          {/each}
        </ol>
        {#if f.note}<p class="hint">💡 {f.note}</p>{/if}
      {/if}
    </section>
    <div class="pile">
      <section class="card">
        <div class="card-head">
          <h2>Adresse publiée</h2>
          {#if ip.publiee}
            <span class="chip" class:ok={ipOK} class:warn={!ipOK}>{ipOK ? 'À jour' : 'Différente'}</span>
          {/if}
        </div>
        <div class="row">
          <div class="row-text">
            <span class="row-name">Publier l'IP de ce PC automatiquement</span>
            <span class="row-detail">Vérifiée au démarrage puis toutes les 10 minutes, mise à jour chez ton fournisseur si besoin.</span>
          </div>
          <Switch checked={st?.ipAuto ?? true} disabled={app.busy} label="Publier l'IP de ce PC automatiquement" onchange={setIPAuto} />
        </div>
        <div class="row"><span class="row-name">IP de ce PC</span><span class="value mono">{ip.locale?.ip || '—'}</span></div>
        <div class="row"><span class="row-name">IP de {st?.domaine || 'ton domaine'}</span><span class="value mono">{ip.publiee || '—'}</span></div>
        <div class="row">
          <span class="row-name">Dernière vérification</span>
          <span class="value">{quand(ip.verifie)}</span>
        </div>
        {#if ip.erreur}<p class="error" role="alert">{ip.erreur}</p>{/if}
        {#if ip.avertissement}<p class="hint warn-text">{ip.avertissement}</p>{/if}
        {#if !st?.ipAuto && ip.publiee && !ipOK}
          <p class="hint warn-text">Ton domaine ne pointe pas vers ce PC : active la publication automatique, ou corrige l'adresse chez ton fournisseur.</p>
        {/if}
        <div class="actions">
          <button class="btn-ghost grow" onclick={checkIP} disabled={app.busy || !configure}>{app.busy ? 'Vérification…' : 'Vérifier maintenant'}</button>
        </div>
      </section>

      <section class="card">
        <h2>Certificat</h2>
        <div class="row"><span class="row-name">État</span><span class="chip" class:ok={m.etat === 'actif'}>{m.etat === 'actif' ? 'Servi sur le port 443' : m.etat === 'demarrage' ? 'En cours' : 'Inactif'}</span></div>
        <div class="row"><span class="row-name">Valable jusqu'au</span><span class="value">{formatDate(m.certificat) || '—'}</span></div>
        <div class="row"><span class="row-name">Renouvellement</span><span class="value">auto, 30 j avant</span></div>

        <span class="eyebrow">Comment ça marche</span>
        <ol class="how">
          <li>Ton domaine pointe vers l'adresse <strong>privée</strong> de ce PC (192.168.x.x) : il n'est joignable que depuis chez toi.</li>
          <li>Aria prouve à Let's Encrypt qu'il contrôle le domaine par un enregistrement DNS (DNS-01) : aucun port à ouvrir sur la box.</li>
          <li>Hors de chez toi, ton téléphone passe par ton VPN.</li>
        </ol>
      </section>
    </div>

    <section class="card important cq" aria-labelledby="titre-box">
      <div class="card-head">
        <h2 id="titre-box">Étape indispensable : une adresse IP fixe pour ce PC</h2>
        <span class="chip warn">À faire une fois</span>
      </div>
      <p class="card-text">
        Ton domaine pointe vers l'adresse locale de ce PC. Par défaut, la box peut lui en donner une autre après un redémarrage :
        ton téléphone chercherait alors le coffre au mauvais endroit. Réserve-lui son adresse actuelle sur ta box, c'est
        l'affaire de deux minutes.
      </p>
      <TutoBox ip={ip.locale?.ip} mac={ip.locale?.mac} carte={ip.locale?.interface} />
    </section>
  </div>
</div>

<style>
  .optional {
    color: var(--faint);
    font-weight: 400;
  }
  .seg {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
  }
  .seg button {
    min-height: 40px;
    border: 1px solid var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 13px;
    font-weight: 500;
  }
  .seg button:hover {
    border-color: var(--accent-2);
    color: var(--text);
  }
  .seg button.on {
    border-color: var(--action);
    background: color-mix(in srgb, var(--action) 12%, transparent);
    color: var(--action);
  }
  select {
    width: 100%;
    min-height: 40px;
    padding: 0 10px;
    border-radius: 8px 0;
    corner-shape: bevel;
    border: 1px solid var(--line-strong);
    background: var(--surface-sunken);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 13px;
  }
  .steps {
    margin: 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    font-size: 13px;
    line-height: 1.55;
  }
  .steps li::marker {
    color: var(--action);
    font-family: var(--font-mono);
    font-weight: 600;
  }
  .link {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: 4px;
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 12px;
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  /* Grand écran : la colonne de droite empile adresse publiée et certificat ;
     sur deux colonnes, elle passe dessous, sur toute la largeur. */
  .https-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: 'domaine' 'tuto' 'box' 'pile';
    gap: 16px;
    align-items: start;
  }
  .https-grid > :nth-child(1) {
    grid-area: domaine;
  }
  .https-grid > :nth-child(2) {
    grid-area: tuto;
  }
  .pile {
    grid-area: pile;
    display: grid;
    gap: 16px;
    align-items: start;
  }
  .important {
    grid-area: box;
  }
  @container (min-width: 640px) {
    .https-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      grid-template-areas: 'domaine tuto' 'box box' 'pile pile';
    }
    .pile {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  @container (min-width: 1180px) {
    .https-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
      grid-template-areas: 'domaine tuto pile' 'box box pile';
    }
    .pile {
      grid-template-columns: minmax(0, 1fr);
    }
  }
  .important {
    border-color: color-mix(in srgb, var(--warn) 45%, var(--line));
    background: linear-gradient(160deg, color-mix(in srgb, var(--warn) 6%, var(--surface)), var(--surface) 55%);
  }
  .important h2 {
    color: var(--warn);
  }
  .warn-text {
    color: var(--warn);
  }
  .mono {
    font-family: var(--font-mono);
  }
  .how {
    margin: 0;
    padding-left: 18px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 12px;
    line-height: 1.6;
    color: var(--muted);
  }
  .how strong {
    color: var(--text);
    font-weight: 500;
  }
</style>
