// État partagé de l'interface et appels au backend Go (VaultService).
import { Call } from '@wailsio/runtime'

const call = (method, ...args) => Call.ByName(`main.VaultService.${method}`, ...args)
const message = (err) => String(err?.message ?? err)

export const app = $state({
  page: 'home',
  status: null,
  loadError: '',
  busy: false,
  error: '', // erreur de la dernière action
  journal: [],
  backups: null, // page Sauvegardes
  fournisseurs: [], // fournisseurs DNS et leurs tutoriels
  installation: null, // null, ou pourcentage téléchargé de la mise à jour
})

export async function loadFournisseurs() {
  if (app.fournisseurs.length) return
  try {
    app.fournisseurs = (await call('Fournisseurs')) ?? []
  } catch (err) {
    app.error = message(err)
  }
}

export async function refresh() {
  try {
    app.status = await call('Status')
    app.loadError = ''
  } catch (err) {
    app.loadError = message(err)
  }
}

export async function refreshJournal() {
  try {
    app.journal = (await call('Journal')) ?? []
  } catch {}
}

// L'état change tout seul (démarrage, certificat, plantage) : on le suit.
let timer = null
export function startPolling() {
  if (timer) return
  timer = setInterval(() => {
    if (app.busy) return
    refresh()
    if (app.page === 'journal' || app.page === 'home' || app.status?.moteur?.etat === 'demarrage') refreshJournal()
  }, 2000)
}

async function run(fn) {
  app.busy = true
  app.error = ''
  try {
    return await fn()
  } catch (err) {
    app.error = message(err)
  } finally {
    app.busy = false
    await refresh()
    refreshJournal()
  }
}

export const start = () => run(() => call('Start'))
export const stop = () => run(() => call('Stop'))
export const restart = () => run(() => call('Restart'))
export const setInscriptions = (on) => run(() => call('SetInscriptions', on))
export const setAutostart = (on) => run(() => call('SetAutostart', on))
export const openVault = () => run(() => call('OpenVault'))
export const openData = () => run(() => call('OpenDataFolder'))
export const setIPAuto = (on) => run(() => call('SetIPAuto', on))
export const checkIP = () => run(() => call('CheckIP'))
export const setVerifMaj = (on) => run(() => call('SetVerifMaj', on))
export const checkUpdate = () => run(() => call('CheckUpdate'))
export const openRelease = () => run(() => call('OpenRelease'))

// Installe la nouvelle version : téléchargement vérifié, remplacement de l'exe,
// puis la nouvelle version prend le relais et cette fenêtre se ferme.
export async function installUpdate() {
  app.installation = 0
  const ok = await run(async () => {
    await call('InstallUpdate')
    return true
  })
  if (!ok) app.installation = null
  else app.installation = 100
  return ok
}
export const setEmpecherVeille = (on) => run(() => call('SetEmpecherVeille', on))

// Les liens des tutoriels s'ouvrent dans le navigateur, sans bloquer l'interface.
export async function openLink(url) {
  try {
    await call('OpenLink', url)
  } catch (err) {
    app.error = message(err)
  }
}

export async function saveConfig(input) {
  const res = await run(() => call('SaveConfig', input))
  if (res?.error) app.error = res.error
  return res ?? { error: app.error }
}

// ---------- Mise en forme ----------

// Go envoie une date nulle pour « jamais » (année 1).
const valid = (iso) => iso && new Date(iso).getFullYear() > 2000

export function formatDate(iso) {
  if (!valid(iso)) return ''
  return new Date(iso).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
}

export function formatSince(iso) {
  if (!valid(iso)) return ''
  const s = Math.max(0, Math.round((Date.now() - new Date(iso)) / 1000))
  if (s < 60) return 'à l\'instant'
  if (s < 3600) return `depuis ${Math.round(s / 60)} min`
  if (s < 86400) return `depuis ${Math.floor(s / 3600)} h ${Math.round((s % 3600) / 60)} min`
  return `depuis ${Math.floor(s / 86400)} j`
}

export function daysLeft(iso) {
  if (!valid(iso)) return null
  return Math.floor((new Date(iso) - Date.now()) / 86400000)
}

// ---------- Sauvegardes ----------

export async function loadBackups() {
  try {
    app.backups = await call('Backups')
  } catch (err) {
    app.error = message(err)
  }
}

async function runBackup(fn) {
  const res = await run(fn)
  await loadBackups()
  return res
}

export const backupNow = () => runBackup(() => call('Backup'))

// Accueil et Ctrl+S : sans dossier choisi, la page Sauvegardes le fait choisir.
export async function sauvegarderMaintenant() {
  if (!app.status?.configure) return
  if (!app.backups) await loadBackups()
  if (!app.backups?.dossier) {
    app.page = 'sauvegardes'
    return
  }
  return backupNow()
}
export async function restoreBackup(nom, phrase = '') {
  const res = await runBackup(() => call('Restore', nom, phrase))
  if (res?.error) app.error = res.error
  return res ?? { error: app.error }
}
export const setBackupEncryption = (on, phrase = '') => runBackup(() => call('SetBackupEncryption', on, phrase))
export const importVault = () => runBackup(() => call('ImportVault'))
export const chooseBackupFolder = () => runBackup(() => call('ChooseBackupFolder'))
export const openBackupFolder = () => run(() => call('OpenBackupFolder'))
export const setDailyBackup = (on) => runBackup(() => call('SetDailyBackup', on))
export const exportSettings = (phrase) => run(() => call('ExportSettings', phrase))

export async function importSettings(phrase) {
  const res = await run(() => call('ImportSettings', phrase))
  if (res?.error) app.error = res.error
  return res
}

export function formatHeure(iso) {
  if (!valid(iso)) return ''
  return new Date(iso).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
}

export function formatDateHeure(iso) {
  if (!valid(iso)) return ''
  return new Date(iso).toLocaleString('fr-FR', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export function formatTaille(n) {
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1).replace('.', ',')} Mo`
  if (n >= 1 << 10) return `${Math.round(n / (1 << 10))} Ko`
  return `${n} o`
}
