<script>
  import { onMount } from 'svelte'
  import Switch from '../lib/Switch.svelte'
  import {
    app, loadBackups, backupNow, restoreBackup, importVault, chooseBackupFolder, openBackupFolder,
    setDailyBackup, setBackupEncryption, exportSettings, importSettings, formatDateHeure, formatSince, formatTaille,
  } from '../lib/state.svelte.js'

  const b = $derived(app.backups)
  const liste = $derived(b?.liste ?? [])
  const choisi = $derived(!!b?.dossier)
  const enLigne = $derived(['actif', 'demarrage'].includes(app.status?.moteur?.etat))

  let aConfirmer = $state('') // sauvegarde en attente de confirmation de restauration
  let message = $state('')

  // Restauration d'une sauvegarde chiffrée dont la phrase est demandée
  let phraseDe = $state('')
  let phraseRestau = $state('')

  // Chiffrement des sauvegardes
  let reglerChiffre = $state(false)
  let phraseSvg = $state('')
  let phraseSvg2 = $state('')
  let noteeSvg = $state(false)
  const phraseSvgOk = $derived(phraseSvg.length >= 8 && phraseSvg === phraseSvg2 && noteeSvg)

  let phraseExport = $state('')
  let phraseExport2 = $state('')
  let phraseImport = $state('')
  let montrerPhrases = $state(false)
  const phraseOk = $derived(phraseExport.length >= 8 && phraseExport === phraseExport2)

  onMount(loadBackups)

  async function sauvegarder() {
    message = ''
    // Premier clic sans dossier : on le choisit d'abord, puis on sauvegarde.
    if (!choisi) {
      const ok = await chooseBackupFolder()
      if (!ok || app.error) return
    }
    const info = await backupNow()
    if (info?.nom) message = `Sauvegarde créée : ${info.nom} (${formatTaille(info.taille)})`
  }

  async function restaurer(nom, phrase = '') {
    if (!phrase && aConfirmer !== nom) {
      aConfirmer = nom
      return
    }
    aConfirmer = ''
    message = ''
    const res = await restoreBackup(nom, phrase)
    if (res?.besoinPhrase) {
      phraseDe = nom
      return
    }
    phraseDe = phraseRestau = ''
    if (!res?.error) message = 'Coffre restauré. L\'ancien a été mis de côté à côté du dossier data, rien n\'est supprimé.'
  }

  async function activerChiffrement(event) {
    event.preventDefault()
    message = ''
    await setBackupEncryption(true, phraseSvg)
    if (!app.error) {
      message = 'Les prochaines sauvegardes seront chiffrées. Garde ta phrase en lieu sûr : sans elle, elles sont irrécupérables.'
      reglerChiffre = noteeSvg = false
      phraseSvg = phraseSvg2 = ''
    }
  }

  async function basculerChiffrement(on) {
    if (on) {
      reglerChiffre = true
      return
    }
    reglerChiffre = false
    await setBackupEncryption(false)
  }

  async function importer() {
    message = ''
    const ok = await importVault()
    if (ok && !app.error) message = 'Ancien coffre importé. Le précédent a été mis de côté, rien n\'est supprimé.'
  }

  async function exporter() {
    message = ''
    const ok = await exportSettings(phraseExport)
    if (ok && !app.error) {
      message = 'Réglages exportés dans un fichier entièrement chiffré. Garde ta phrase de passe en lieu sûr : sans elle, le fichier est illisible.'
      phraseExport = phraseExport2 = ''
    }
  }

  async function importerReglages() {
    message = ''
    const res = await importSettings(phraseImport)
    if (res && !res.annule && !res.error) {
      message = `Réglages importés pour ${res.domaine}, identifiants DNS compris.`
      phraseImport = ''
    }
  }
</script>

<div class="page">
  <div class="page-head">
    <h1>Sauvegardes</h1>
    <span class="pill" class:on={liste.length > 0}>
      {liste.length ? `Dernière : ${formatSince(liste[0].date).replace('depuis', 'il y a')}` : choisi ? 'Aucune sauvegarde' : 'Dossier à choisir'}
    </span>
  </div>
  <p class="page-intro">
    Tout ton coffre tient dans un petit zip : base, clés du serveur, pièces jointes et Sends. Tes mots de passe y restent
    chiffrés par ton mot de passe maître, et le zip lui-même peut l'être par une phrase de passe.
  </p>

  {#if message}<p class="banner ok" role="status">{message}</p>{/if}
  {#if app.error}<p class="error" role="alert">{app.error}</p>{/if}

  <div class="grid-3">
    <section class="card">
      <h2>Coffre</h2>
      <button class="btn-primary big" onclick={sauvegarder} disabled={app.busy || !app.status?.configure}>
        {app.busy ? 'En cours…' : choisi ? 'Sauvegarder maintenant' : 'Choisir et sauvegarder'}
      </button>
      <p class="hint">
        {enLigne ? 'Le serveur se met en pause quelques secondes, ton téléphone réessaie tout seul.' : 'Aussi possible d\'un clic droit sur l\'icône Aria.'}
      </p>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Sauvegarde quotidienne</span>
          <span class="row-detail">Une par jour tant qu'Aria tourne. Les {b?.garder ?? 10} plus récentes sont gardées.</span>
        </div>
        <Switch checked={b?.auto ?? true} disabled={app.busy} label="Sauvegarde quotidienne" onchange={setDailyBackup} />
      </div>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Chiffrer les sauvegardes</span>
          <span class="row-detail">
            {b?.chiffrees ? 'Activé : les nouvelles sauvegardes sont illisibles sans ta phrase.' : 'Recommandé si le dossier est dans OneDrive ou sur une clé USB.'}
          </span>
        </div>
        <Switch checked={b?.chiffrees || reglerChiffre} disabled={app.busy} label="Chiffrer les sauvegardes" onchange={basculerChiffrement} />
      </div>
      {#if reglerChiffre}
        <form class="form chiffre" onsubmit={activerChiffrement}>
          <p class="hint">
            Le zip contient déjà tes mots de passe chiffrés par ton mot de passe maître, mais aussi les clés du serveur, les
            adresses e-mail des comptes et les métadonnées. Avec cette option, tout le fichier est chiffré (scrypt + AES-256-GCM).
          </p>
          <input type="password" bind:value={phraseSvg} placeholder="Phrase de passe (8 caractères min.)" autocomplete="new-password" />
          <input type="password" bind:value={phraseSvg2} placeholder="La même, pour confirmer" autocomplete="new-password" />
          {#if phraseSvg2 && phraseSvg !== phraseSvg2}<span class="field-error">Les deux phrases sont différentes.</span>{/if}
          <p class="alerte">
            ⚠ Aucune récupération possible : si tu perds cette phrase et ce PC, tes sauvegardes chiffrées sont perdues. Aria la
            garde pour ce compte Windows (DPAPI) afin de faire la quotidienne, mais il te la faudra pour restaurer sur un autre PC.
          </p>
          <label class="check">
            <input type="checkbox" bind:checked={noteeSvg} />
            <span>J'ai noté ma phrase en lieu sûr (gestionnaire de mots de passe d'un autre appareil, papier…)</span>
          </label>
          <div class="actions">
            <button type="submit" class="btn-primary grow" disabled={app.busy || !phraseSvgOk}>Chiffrer les prochaines sauvegardes</button>
            <button type="button" class="btn-ghost" onclick={() => (reglerChiffre = false)} disabled={app.busy}>Annuler</button>
          </div>
        </form>
      {:else if b?.chiffrees}
        <button class="btn-ghost small changer" onclick={() => (reglerChiffre = true)} disabled={app.busy}>Changer la phrase…</button>
      {/if}

      <div class="row">
        <div class="row-text">
          <span class="row-name">Dossier</span>
          {#if choisi}
            <span class="path">{b.dossier}</span>
          {:else}
            <span class="row-detail warn-text">à choisir</span>
          {/if}
        </div>
        <span class="btns">
          <button class="small {choisi ? 'btn-ghost' : 'btn-primary'}" onclick={chooseBackupFolder} disabled={app.busy}>
            {choisi ? 'Changer' : 'Choisir…'}
          </button>
          {#if choisi}
            <button class="btn-ghost small" onclick={openBackupFolder} disabled={app.busy}>Ouvrir</button>
          {/if}
        </span>
      </div>
      {#if !choisi}
        <p class="hint warn-text">
          Aucune sauvegarde tant que le dossier n'est pas choisi, la quotidienne comprise. Choisis de préférence un dossier
          OneDrive ou un disque externe ; tu peux en créer un nouveau directement dans la fenêtre.
        </p>
      {:else if b.memeDisque}
        <p class="hint warn-text">
          Ce dossier est sur le même disque que le coffre : si le disque lâche, tu perds les deux. Un dossier OneDrive ou un
          disque externe serait plus sûr.
        </p>
      {/if}
    </section>

    <section class="card">
      <h2>Historique</h2>
      {#if liste.length}
        <div class="list">
          {#each liste as s (s.nom)}
            <div class="row">
              <div class="row-text">
                <span class="row-name">{formatDateHeure(s.date)}</span>
                <span class="row-detail mono">{formatTaille(s.taille)}{#if s.chiffree}<span class="tag">chiffrée</span>{/if}</span>
              </div>
              <button
                class="small {aConfirmer === s.nom ? 'btn-primary' : 'btn-ghost'}"
                onclick={() => restaurer(s.nom)}
                onblur={() => aConfirmer === s.nom && (aConfirmer = '')}
                disabled={app.busy}
              >
                {aConfirmer === s.nom ? 'Confirmer ?' : 'Restaurer'}
              </button>
            </div>
            {#if phraseDe === s.nom}
              <form class="form phrase" onsubmit={(e) => { e.preventDefault(); restaurer(s.nom, phraseRestau) }}>
                <span class="row-detail">Phrase de passe de cette sauvegarde :</span>
                <span class="with-btn">
                  <input type="password" bind:value={phraseRestau} placeholder="Phrase choisie à l'époque" autocomplete="off" />
                  <button type="submit" class="btn-primary small" disabled={app.busy || !phraseRestau}>Restaurer</button>
                  <button type="button" class="btn-ghost small" onclick={() => (phraseDe = phraseRestau = '')} disabled={app.busy}>Annuler</button>
                </span>
              </form>
            {/if}
          {/each}
        </div>
        <p class="hint">Restaurer remplace le coffre actuel, qui est d'abord mis de côté : rien n'est jamais supprimé.</p>
      {:else}
        <p class="card-text">Aucune sauvegarde pour l'instant.</p>
      {/if}
      <div class="actions">
        <button class="btn-ghost grow" onclick={importer} disabled={app.busy}>Importer un ancien coffre…</button>
      </div>
      <p class="hint">Un ancien dossier data, comme celui de C:\Vaultwarden.</p>
    </section>

    <section class="card cq reglages-card">
      <div class="card-head">
        <h2>Réglages d'Aria</h2>
        <button class="btn-ghost small" onclick={() => (montrerPhrases = !montrerPhrases)}>
          {montrerPhrases ? 'Masquer les phrases' : 'Afficher les phrases'}
        </button>
      </div>
      <p class="card-text">
        Fournisseur DNS, domaine, e-mail et identifiants, dans un fichier .ariavault à part du coffre, entièrement chiffré par ta phrase
        de passe : sans elle, rien n'y est lisible. Sur un autre PC, tu retrouves tout sans rien ressaisir.
      </p>
      <div class="grid-2">
        <form class="form" onsubmit={(e) => { e.preventDefault(); exporter() }}>
          <span class="eyebrow">Exporter</span>
          <input type={montrerPhrases ? 'text' : 'password'} bind:value={phraseExport} placeholder="Phrase de passe (8 caractères min.)" autocomplete="new-password" />
          <input type={montrerPhrases ? 'text' : 'password'} bind:value={phraseExport2} placeholder="La même, pour confirmer" autocomplete="new-password" />
          {#if phraseExport2 && phraseExport !== phraseExport2}<span class="field-error">Les deux phrases sont différentes.</span>{/if}
          <button type="submit" class="btn-ghost" disabled={app.busy || !phraseOk || !app.status?.configure}>Exporter les réglages…</button>
        </form>
        <form class="form" onsubmit={(e) => { e.preventDefault(); importerReglages() }}>
          <span class="eyebrow">Importer</span>
          <input type={montrerPhrases ? 'text' : 'password'} bind:value={phraseImport} placeholder="Phrase de passe du fichier" autocomplete="off" />
          <p class="hint">La phrase choisie à l'export. Si elle est fausse, rien n'est modifié.</p>
          <button type="submit" class="btn-ghost" disabled={app.busy || !phraseImport}>Importer des réglages…</button>
        </form>
      </div>
    </section>
  </div>
</div>

<style>
  /* Sur deux colonnes, les réglages d'Aria passent dessous, sur toute la largeur. */
  @container (min-width: 640px) and (max-width: 1179px) {
    .reglages-card {
      grid-column: 1 / -1;
    }
  }
  .big {
    min-height: 46px;
    font-size: 15px;
  }
  .btns {
    display: flex;
    gap: 8px;
    flex: none;
  }
  .list {
    display: flex;
    flex-direction: column;
    max-height: 300px;
    overflow-y: auto;
    padding-right: 4px;
  }
  .warn-text {
    color: var(--warn);
  }
  .tag {
    margin-left: 8px;
    padding: 1px 6px;
    border-radius: 4px 0;
    background: color-mix(in srgb, var(--accent-2) 16%, transparent);
    color: var(--accent-2);
    font-size: 11px;
  }
  .chiffre,
  .phrase {
    padding: 12px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
  }
  .phrase {
    margin: 4px 0 8px;
    gap: 8px;
  }
  .alerte {
    margin: 0;
    padding: 10px 12px;
    border-left: 3px solid var(--danger);
    background: color-mix(in srgb, var(--danger) 8%, transparent);
    font-size: 12px;
    line-height: 1.55;
    color: #ffc2cd;
  }
  .check {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    font-size: 12px;
    line-height: 1.5;
    color: var(--muted);
    cursor: pointer;
  }
  .check input {
    width: 16px;
    min-height: 16px;
    margin-top: 2px;
    flex: none;
    accent-color: var(--action);
  }
  .changer {
    align-self: flex-start;
  }
  .banner.ok {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    border-color: color-mix(in srgb, var(--accent) 40%, transparent);
    color: #bdf9ff;
  }
</style>
