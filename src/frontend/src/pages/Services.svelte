<script>
  import Icon from '../lib/Icon.svelte'
  import Switch from '../lib/Switch.svelte'
  import {
    app, savePush, disablePush, saveSMTP, disableSMTP, testSMTP, setInvitations, openLink,
  } from '../lib/state.svelte.js'

  const st = $derived(app.status)
  const push = $derived(st?.push ?? { actif: false })
  const smtp = $derived(st?.smtp ?? { actif: false })
  const enLigne = $derived(['actif', 'demarrage'].includes(st?.moteur?.etat))

  // ---------- Synchronisation instantanée ----------
  let editPush = $state(false)
  let voirCle = $state(false)
  let champsPush = $state({})
  let fPush = $state({ region: 'eu', id: '', cle: '' })
  const formPush = $derived(!push.actif || editPush)

  function modifierPush() {
    fPush = { region: push.region || 'eu', id: push.id || '', cle: '' }
    champsPush = {}
    editPush = true
  }
  async function enregistrerPush(e) {
    e.preventDefault()
    const res = await savePush({ ...fPush })
    champsPush = res?.fields ?? {}
    if (!res?.error) {
      editPush = voirCle = false
      fPush.cle = ''
    }
  }

  // ---------- E-mails ----------
  const modeles = {
    gmail: { hote: 'smtp.gmail.com', port: 587, securite: 'starttls' },
    autre: { hote: '', port: 0, securite: 'starttls' },
  }
  const securites = [
    { id: 'starttls', nom: 'STARTTLS', port: 587 },
    { id: 'force_tls', nom: 'TLS', port: 465 },
    { id: 'off', nom: 'Aucune', port: 25 },
  ]
  let editSmtp = $state(false)
  let voirMdp = $state(false)
  let champsSmtp = $state({})
  let modele = $state('gmail')
  let fSmtp = $state({ hote: 'smtp.gmail.com', port: 587, securite: 'starttls', utilisateur: '', motDePasse: '', de: '' })
  const formSmtp = $derived(!smtp.actif || editSmtp)
  let destTest = $state('')
  let testOk = $state(false)

  function choisirModele(id) {
    modele = id
    fSmtp = { ...fSmtp, ...modeles[id] }
    if (id === 'gmail' && fSmtp.utilisateur && !fSmtp.de) fSmtp.de = fSmtp.utilisateur
  }
  function choisirSecurite(id) {
    const ancienDefaut = securites.find((x) => x.id === fSmtp.securite)?.port
    fSmtp.securite = id
    if (!fSmtp.port || fSmtp.port === ancienDefaut) fSmtp.port = securites.find((x) => x.id === id).port
  }
  function modifierSmtp() {
    modele = smtp.hote === 'smtp.gmail.com' ? 'gmail' : 'autre'
    fSmtp = { hote: smtp.hote, port: smtp.port, securite: smtp.securite, utilisateur: smtp.utilisateur, motDePasse: '', de: smtp.de }
    champsSmtp = {}
    editSmtp = true
  }
  async function enregistrerSmtp(e) {
    e.preventDefault()
    // Gmail : l'expéditeur est l'adresse du compte
    if (modele === 'gmail' && !fSmtp.de) fSmtp.de = fSmtp.utilisateur
    const res = await saveSMTP({ ...fSmtp, port: Number(fSmtp.port) || 0 })
    champsSmtp = res?.fields ?? {}
    if (!res?.error) {
      editSmtp = voirMdp = false
      fSmtp.motDePasse = ''
    }
  }
  async function envoyerTest() {
    testOk = false
    testOk = (await testSMTP(destTest.trim())) === true
  }
  const tutoGmail = $derived(formSmtp ? modele === 'gmail' : smtp.hote === 'smtp.gmail.com')
  const gardeMdp = $derived(smtp.motDePasse && fSmtp.hote === smtp.hote && fSmtp.utilisateur === smtp.utilisateur)
</script>

<div class="page">
  <div class="page-head">
    <h1>Services</h1>
  </div>
  <p class="page-intro">
    Deux services facultatifs de Vaultwarden. Ils passent par des serveurs extérieurs : le relais de Bitwarden pour la
    synchronisation instantanée, ta messagerie pour les e-mails. Tes mots de passe, eux, ne quittent jamais ton coffre.
  </p>

  {#if app.error}<p class="error" role="alert">{app.error}</p>{/if}

  <div class="grid-2">
    <!-- ======================= Synchronisation instantanée ======================= -->
    <section class="card">
      <div class="card-head">
        <h2>Synchronisation instantanée</h2>
        <span class="pill" class:on={push.actif}>{push.actif ? 'Active' : 'Inactive'}</span>
      </div>
      <p class="card-text">
        Sans elle, ton téléphone ne voit tes changements qu'à l'ouverture de l'appli ou après un moment. Avec elle, Vaultwarden le
        prévient aussitôt via le relais de notifications de Bitwarden, gratuit.
      </p>

      {#if formPush}
        <form class="form" onsubmit={enregistrerPush}>
          <div class="field">
            <span class="field-label">Région des données</span>
            <div class="seg" role="radiogroup" aria-label="Région des données">
              <button type="button" role="radio" aria-checked={fPush.region === 'eu'} class:on={fPush.region === 'eu'} onclick={() => (fPush.region = 'eu')}>Union européenne</button>
              <button type="button" role="radio" aria-checked={fPush.region === 'us'} class:on={fPush.region === 'us'} onclick={() => (fPush.region = 'us')}>États-Unis</button>
            </div>
            {#if champsPush.region}<span class="field-error">{champsPush.region}</span>{/if}
          </div>
          <label class="field">
            <span class="field-label">Installation ID</span>
            <input bind:value={fPush.id} placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" autocomplete="off" spellcheck="false" />
            {#if champsPush.id}<span class="field-error">{champsPush.id}</span>{/if}
          </label>
          <label class="field">
            <span class="field-label">Installation Key</span>
            <span class="with-btn">
              <input
                type={voirCle ? 'text' : 'password'}
                bind:value={fPush.cle}
                placeholder={push.actif ? '•••••••• (laisse vide pour garder l\'actuelle)' : 'Clé reçue sur bitwarden.com/host'}
                autocomplete="off"
                spellcheck="false"
              />
              <button type="button" class="btn-ghost small" onclick={() => (voirCle = !voirCle)}>{voirCle ? 'Masquer' : 'Afficher'}</button>
            </span>
            {#if champsPush.cle}<span class="field-error">{champsPush.cle}</span>{/if}
          </label>
          <div class="actions">
            <button type="submit" class="btn-primary grow" disabled={app.busy}>{enLigne ? 'Activer et relancer' : 'Activer'}</button>
            {#if editPush}<button type="button" class="btn-ghost" onclick={() => (editPush = false)} disabled={app.busy}>Annuler</button>{/if}
          </div>
        </form>
      {:else}
        <div class="row"><span class="row-name">Région</span><span class="value">{push.region === 'eu' ? 'Union européenne' : 'États-Unis'}</span></div>
        <div class="row"><span class="row-name">Installation ID</span><span class="value mono">{push.id}</span></div>
        <div class="row"><span class="row-name">Installation Key</span><span class="value mono">•••••••• chiffrée</span></div>
        <div class="actions">
          <button class="btn-ghost grow" onclick={modifierPush} disabled={app.busy}>Modifier</button>
          <button class="btn-ghost" onclick={disablePush} disabled={app.busy}>Désactiver</button>
        </div>
      {/if}

      <span class="eyebrow">Tuto</span>
      <ol class="steps">
        <li>
          Ouvre la page d'hébergement de Bitwarden et saisis ton adresse e-mail : tu reçois un <em>Installation ID</em> et une
          <em>Installation Key</em>, gratuitement.
          <button class="link" onclick={() => openLink('https://bitwarden.com/host/')}>bitwarden.com/host<Icon name="external" size={13} /></button>
        </li>
        <li>Choisis la même région des données ici et là-bas (Union européenne ou États-Unis).</li>
        <li>Colle l'ID et la clé ci-dessus, puis active : le serveur redémarre quelques secondes.</li>
        <li>Sur ton téléphone, déconnecte-toi puis reconnecte-toi dans l'appli Bitwarden pour qu'elle s'inscrive aux notifications.</li>
        <li>Pour vérifier : renomme un dossier dans le coffre web, il change sur le téléphone en quelques secondes.</li>
      </ol>
      <p class="hint">
        💡 Ça marche avec l'appli Bitwarden du Play Store ou de l'App Store, pas avec celle de F-Droid. Si ton téléphone ne reçoit
        rien après la reconnexion, vider les données de l'appli (ou la réinstaller) puis se reconnecter règle le problème.
      </p>
    </section>

    <!-- ======================= E-mails ======================= -->
    <section class="card">
      <div class="card-head">
        <h2>E-mails</h2>
        <span class="pill" class:on={smtp.actif}>{smtp.actif ? 'Actifs' : 'Inactifs'}</span>
      </div>
      <p class="card-text">
        Vaultwarden peut alors t'alerter d'une connexion depuis un nouvel appareil, proposer la double authentification par
        e-mail et envoyer les invitations (partage en famille, accès d'urgence).
      </p>

      {#if formSmtp}
        <form class="form" onsubmit={enregistrerSmtp}>
          <div class="field">
            <span class="field-label">Messagerie</span>
            <div class="seg" role="radiogroup" aria-label="Messagerie">
              <button type="button" role="radio" aria-checked={modele === 'gmail'} class:on={modele === 'gmail'} onclick={() => choisirModele('gmail')}>Gmail</button>
              <button type="button" role="radio" aria-checked={modele === 'autre'} class:on={modele === 'autre'} onclick={() => choisirModele('autre')}>Autre</button>
            </div>
          </div>
          {#if modele === 'autre'}
            <div class="deux">
              <label class="field">
                <span class="field-label">Serveur SMTP</span>
                <input bind:value={fSmtp.hote} placeholder="smtp.exemple.fr" autocomplete="off" spellcheck="false" />
                {#if champsSmtp.hote}<span class="field-error">{champsSmtp.hote}</span>{/if}
              </label>
              <label class="field port">
                <span class="field-label">Port</span>
                <input type="number" min="1" max="65535" bind:value={fSmtp.port} />
                {#if champsSmtp.port}<span class="field-error">{champsSmtp.port}</span>{/if}
              </label>
            </div>
            <div class="field">
              <span class="field-label">Sécurité</span>
              <div class="seg trois" role="radiogroup" aria-label="Sécurité">
                {#each securites as x (x.id)}
                  <button type="button" role="radio" aria-checked={fSmtp.securite === x.id} class:on={fSmtp.securite === x.id} onclick={() => choisirSecurite(x.id)}>{x.nom}</button>
                {/each}
              </div>
              {#if champsSmtp.securite}<span class="field-error">{champsSmtp.securite}</span>{/if}
            </div>
          {/if}
          <label class="field">
            <span class="field-label">{modele === 'gmail' ? 'Adresse Gmail' : 'Identifiant'}</span>
            <input bind:value={fSmtp.utilisateur} placeholder={modele === 'gmail' ? 'moi@gmail.com' : 'Souvent ton adresse e-mail'} autocomplete="off" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">{modele === 'gmail' ? 'Mot de passe d\'application' : 'Mot de passe'}</span>
            <span class="with-btn">
              <input
                type={voirMdp ? 'text' : 'password'}
                bind:value={fSmtp.motDePasse}
                placeholder={gardeMdp ? '•••••••• (laisse vide pour garder l\'actuel)' : modele === 'gmail' ? '16 lettres, espaces compris ou non' : ''}
                autocomplete="new-password"
              />
              <button type="button" class="btn-ghost small" onclick={() => (voirMdp = !voirMdp)}>{voirMdp ? 'Masquer' : 'Afficher'}</button>
            </span>
            {#if champsSmtp.motDePasse}<span class="field-error">{champsSmtp.motDePasse}</span>{/if}
          </label>
          {#if modele === 'autre'}
            <label class="field">
              <span class="field-label">Adresse d'expédition</span>
              <input bind:value={fSmtp.de} placeholder="moi@exemple.fr" autocomplete="off" spellcheck="false" />
              {#if champsSmtp.de}<span class="field-error">{champsSmtp.de}</span>{/if}
            </label>
          {:else if champsSmtp.de}
            <span class="field-error">{champsSmtp.de}</span>
          {/if}
          <p class="hint">Le mot de passe est chiffré avec DPAPI : seul ton compte Windows peut le relire.</p>
          <div class="actions">
            <button type="submit" class="btn-primary grow" disabled={app.busy}>{enLigne ? 'Enregistrer et relancer' : 'Enregistrer'}</button>
            {#if editSmtp}<button type="button" class="btn-ghost" onclick={() => (editSmtp = false)} disabled={app.busy}>Annuler</button>{/if}
          </div>
        </form>
      {:else}
        <div class="row"><span class="row-name">Serveur</span><span class="value mono">{smtp.hote}:{smtp.port} · {securites.find((x) => x.id === smtp.securite)?.nom}</span></div>
        <div class="row"><span class="row-name">Expéditeur</span><span class="value mono">{smtp.de}</span></div>
        <div class="row"><span class="row-name">Identifiant</span><span class="value mono">{smtp.utilisateur || '—'}</span></div>
        <div class="actions">
          <button class="btn-ghost grow" onclick={modifierSmtp} disabled={app.busy}>Modifier</button>
          <button class="btn-ghost" onclick={disableSMTP} disabled={app.busy}>Désactiver</button>
        </div>

        <form class="form test" onsubmit={(e) => { e.preventDefault(); envoyerTest() }}>
          <span class="field-label">Envoyer un e-mail de test</span>
          <span class="with-btn">
            <input bind:value={destTest} placeholder={smtp.de} autocomplete="off" spellcheck="false" />
            <button type="submit" class="btn-primary small" disabled={app.busy}>{app.busy ? 'Envoi…' : 'Envoyer'}</button>
          </span>
          {#if testOk}<p class="hint ok-text">✓ E-mail envoyé : regarde ta boîte (et les indésirables).</p>{/if}
        </form>

        <div class="row">
          <div class="row-text">
            <span class="row-name">Autoriser les invitations</span>
            <span class="row-detail">Inviter un proche dans une organisation ou pour l'accès d'urgence, même inscriptions fermées.</span>
          </div>
          <Switch checked={st?.invitations ?? false} disabled={app.busy} label="Autoriser les invitations" onchange={setInvitations} />
        </div>
      {/if}

      <span class="eyebrow">Tuto {tutoGmail ? 'Gmail' : ''}</span>
      {#if tutoGmail}
        <ol class="steps">
          <li>Active la validation en deux étapes sur ton compte Google, si ce n'est pas déjà fait.</li>
          <li>
            Crée un mot de passe d'application nommé « Aria Vault » et recopie ses 16 lettres.
            <button class="link" onclick={() => openLink('https://myaccount.google.com/apppasswords')}>myaccount.google.com/apppasswords<Icon name="external" size={13} /></button>
          </li>
          <li>Saisis ton adresse Gmail et ce mot de passe d'application (pas celui de ton compte), puis enregistre.</li>
          <li>Envoie-toi un e-mail de test.</li>
        </ol>
        <p class="hint">
          💡 Un mot de passe d'application est révoqué si tu changes le mot de passe de ton compte Google : il faudra en créer un
          autre. Tu ne trouves pas l'option ? Elle n'existe pas sur les comptes professionnels ou scolaires, ni avec la Protection
          Avancée.
        </p>
      {:else}
        <ol class="steps">
          <li>Cherche « paramètres SMTP » dans l'aide de ta messagerie : serveur, port et sécurité y sont indiqués.</li>
          <li>STARTTLS va avec le port 587, TLS avec le port 465. « Aucune » n'est permis que sans mot de passe (relais local).</li>
          <li>Enregistre, puis envoie-toi un e-mail de test.</li>
        </ol>
        <p class="hint">
          💡 Outlook.com et Hotmail n'acceptent plus de mot de passe pour envoyer depuis une autre appli (Microsoft exige OAuth) :
          utilise plutôt Gmail ou la messagerie de ton fournisseur d'accès.
        </p>
      {/if}
    </section>
  </div>
</div>

<style>
  .seg {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
  }
  .seg.trois {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .seg button {
    min-height: 36px;
    border: 1px solid var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 13px;
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
  .deux {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 96px;
    gap: 10px;
  }
  .steps {
    margin: 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 13px;
    line-height: 1.55;
  }
  .steps li::marker {
    color: var(--action);
    font-family: var(--font-mono);
    font-weight: 600;
  }
  .steps em {
    font-style: normal;
    color: var(--accent);
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
  .test {
    padding: 12px;
    gap: 8px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
  }
  .ok-text {
    color: var(--accent);
  }
  .mono {
    font-family: var(--font-mono);
  }
</style>
