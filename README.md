<p align="center">
  <img src="docs/logo.png" width="160" alt="Logo d'Aria Vault : un A néon glitché avec une mallette blindée orange">
</p>

<h1 align="center">Aria Vault</h1>

<p align="center">
  Ton coffre de mots de passe Vaultwarden, en HTTPS, dans un seul exe Windows.<br>
  Il démarre en silence dans la zone de notification et sert les applis Bitwarden<br>
  de ton téléphone, de ton navigateur et de ton PC. Rien à installer, aucun port à ouvrir.
</p>

<p align="center">
  <a href="../../releases/latest"><b>Télécharger la dernière version</b></a> · v0.8 · Windows 10/11 x64 · licence MIT · <a href="docs/SIGNATURE.md">Code signing policy</a>
</p>

## Ce qu'il fait

| Fonction | Ce que ça fait |
| --- | --- |
| **Coffre Vaultwarden** | Serveur compatible Bitwarden, compilé depuis le source officiel et embarqué dans l'exe. Il n'écoute que sur `127.0.0.1` : invisible du réseau |
| **HTTPS automatique** | Vrai certificat Let's Encrypt obtenu et renouvelé seul, par validation DNS chez **DuckDNS, Cloudflare, OVHcloud ou deSEC**. Aucun port à ouvrir sur la box, et l'appli Bitwarden Android l'accepte |
| **IP publiée toute seule** | Toutes les 10 minutes, Aria vérifie que ton domaine pointe vers ce PC et corrige l'enregistrement chez ton fournisseur si besoin |
| **Zone de notification** | Démarre en silence à l'ouverture de session. L'icône s'allume quand le coffre est en ligne, le clic droit ouvre le menu |
| **Sauvegardes** | En un clic ou chaque jour, chiffrées par une phrase de passe si tu veux, les 10 dernières sont gardées. Une restauration ne supprime jamais rien, et une restauration coupée par une panne se termine au lancement suivant |
| **Réglages exportables** | Fournisseur, domaine et identifiants dans un fichier `.ariavault` chiffré par ta phrase de passe, pour réinstaller ailleurs sans rien ressaisir |
| **Mises à jour** | Aria te dit si elle est à jour (accueil et menu de l'icône) et installe la nouvelle version en un clic : téléchargement depuis ce dépôt, empreintes SHA-256 vérifiées, relève automatique |
| **Synchronisation instantanée** | En option, ton téléphone voit chaque modification en quelques secondes, grâce au relais de notifications gratuit de Bitwarden |
| **E-mails** | En option : alertes de nouvelle connexion, double authentification par e-mail, invitations. Gmail préconfiguré, bouton d'e-mail de test |
| **Pas de veille** | En option, le PC ne s'endort pas tout seul tant que le coffre est en ligne |

## Avant de commencer

Il te faut deux choses : **un nom de domaine** chez un des fournisseurs DNS gérés, et **une adresse IP fixe pour ce PC**, réservée sur ta box. Les deux tutos sont ci-dessous, et aussi dans l'appli, page **HTTPS**.

Hors de chez toi, ton téléphone passe par ton **VPN** : ton domaine pointe vers une adresse privée, rien n'est exposé sur Internet.

### 1. Ton domaine et ton fournisseur DNS

| Fournisseur | Pour qui | Ce que tu saisis dans Aria |
| --- | --- | --- |
| [DuckDNS](https://www.duckdns.org) | Pas de nom de domaine : un sous-domaine `exemple.duckdns.org` gratuit | Le jeton (token) |
| [Cloudflare](https://dash.cloudflare.com) | Ton domaine, avec son DNS chez Cloudflare (offre gratuite suffisante) | Un jeton d'API |
| [OVHcloud](https://www.ovhcloud.com) | Ton domaine, avec sa zone DNS chez OVHcloud | Trois clés d'API |
| [deSEC](https://desec.io) | Ton domaine, ou un sous-domaine `exemple.dedyn.io` gratuit | Un jeton |

<details>
<summary><b>Tuto DuckDNS</b> (le plus simple, gratuit)</summary>

1. Ouvre [duckdns.org](https://www.duckdns.org) et connecte-toi avec un des comptes proposés en haut de la page (GitHub, Google, Reddit…). Aucun compte DuckDNS à créer.
2. Dans le champ « sub domain », saisis le nom de ton choix (par exemple « exemple ») puis clique sur « add domain ». Ton domaine devient `exemple.duckdns.org`.
3. Recopie le « token » affiché en haut de la page : c'est le jeton à saisir dans Aria. Garde-le pour toi, il permet de modifier ton domaine.
4. Pas besoin de saisir l'IP sur le site : Aria fait pointer le domaine vers l'IP de ce PC tout seul.

</details>

<details>
<summary><b>Tuto Cloudflare</b></summary>

1. Ton domaine doit déjà utiliser les serveurs DNS de Cloudflare (il apparaît comme « Active » dans ton tableau de bord).
2. Ouvre [My Profile → API Tokens](https://dash.cloudflare.com/profile/api-tokens), puis « Create Token ».
3. Choisis le modèle « Edit zone DNS ». Dans « Permissions », garde *Zone · DNS · Edit* et ajoute une ligne *Zone · Zone · Read* (Aria en a besoin pour trouver ta zone).
4. Dans « Zone Resources », choisis *Include · Specific zone · ton domaine* : le jeton ne pourra toucher à rien d'autre.
5. « Continue to summary » puis « Create Token ». Copie le jeton tout de suite : Cloudflare ne l'affiche qu'une fois.
6. Aria crée l'enregistrement A en « DNS only » (nuage gris) : c'est obligatoire, Cloudflare ne peut pas relayer vers une adresse privée.

</details>

<details>
<summary><b>Tuto OVHcloud</b></summary>

1. Ta zone DNS doit être chez OVHcloud : ton domaine apparaît dans « Web Cloud → Noms de domaine » avec un onglet « Zone DNS ».
2. Ouvre la [page de création de clés d'API](https://api.eu.ovhcloud.com/createToken/?GET=/domain/zone/*&POST=/domain/zone/*&PUT=/domain/zone/*&DELETE=/domain/zone/*) (Europe ; ce lien préremplit les droits) et connecte-toi.
3. Donne un nom (« Aria Vault ») et une description, puis choisis la validité « Unlimited » : sinon les clés expirent et le certificat ne se renouvelle plus.
4. Vérifie les droits : `GET`, `POST`, `PUT` et `DELETE` sur `/domain/zone/*`. Si le formulaire est vide, ajoute ces quatre lignes à la main.
5. Clique sur « Create keys » et recopie les trois valeurs : *Application key*, *Application secret* et *Consumer key*.

Compte OVHcloud Canada ou États-Unis : choisis la région `ovh-ca` ou `ovh-us` dans Aria, et crée les clés sur `api.ca.ovhcloud.com` ou `api.us.ovhcloud.com`.

</details>

<details>
<summary><b>Tuto deSEC</b></summary>

1. Crée un compte sur [desec.io](https://desec.io). À l'inscription, tu peux demander un domaine gratuit en `.dedyn.io`, ou ajouter ensuite ton propre domaine (en le faisant pointer vers les serveurs de deSEC).
2. Une fois connecté, ouvre « Token Management » et crée un nouveau jeton (bouton +), nommé par exemple « Aria Vault ». Il n'a pas besoin du droit de gérer les autres jetons.
3. Copie le jeton tout de suite : deSEC ne le montre qu'une fois.

deSEC impose une durée de cache d'une heure : après un changement d'IP, ton téléphone peut mettre jusqu'à une heure à suivre. D'où l'importance de l'étape 2.

</details>

### 2. Une adresse IP fixe pour ce PC

Ton domaine pointe vers l'adresse locale du PC (`192.168.x.x`). Par défaut, la box peut lui en attribuer une autre après un redémarrage, et ton téléphone chercherait alors le coffre au mauvais endroit. Aria corrige le domaine en 10 minutes au plus, mais les caches DNS peuvent mettre plus longtemps à suivre : **réserve l'adresse du PC sur ta box**, c'est l'affaire de deux minutes.

**Relève l'adresse IP et l'adresse MAC du PC.** Elles sont affichées dans Aria, page HTTPS. Sinon, dans un terminal : `ipconfig /all`, lignes « Adresse IPv4 » et « Adresse physique » de la carte connectée.

Puis, selon ta box (les libellés peuvent varier légèrement selon le modèle et la version) :

<details>
<summary><b>Freebox</b></summary>

1. Ouvre Freebox OS ([mafreebox.freebox.fr](http://mafreebox.freebox.fr)) et connecte-toi avec ton mot de passe Freebox OS.
2. Va dans « Paramètres de la Freebox », passe en « Mode avancé », puis ouvre « DHCP ».
3. Onglet « Baux statiques » → « Ajouter un bail DHCP statique ».
4. Saisis l'adresse MAC et l'adresse IP du PC, puis enregistre.

Plus rapide : dans l'onglet « Baux actifs », un clic droit sur le PC rend son adresse permanente.

</details>

<details>
<summary><b>Livebox</b></summary>

1. Ouvre l'interface de la Livebox ([192.168.1.1](http://192.168.1.1)) et connecte-toi avec le mot de passe d'administration.
2. Clique sur « Paramètres avancés », puis sur « Réseau ».
3. Onglet « DHCP », rubrique « Baux DHCP statiques ».
4. Choisis le PC dans la liste des équipements, saisis son adresse IP, puis clique sur « Ajouter ».

Le PC n'apparaît pas dans la liste ? Choisis « Équipement » en bas de la liste et saisis son adresse MAC à la main.

</details>

<details>
<summary><b>Bbox</b></summary>

1. Ouvre l'interface de la Bbox ([mabbox.bytel.fr](https://mabbox.bytel.fr), ou `192.168.1.254`) et identifie-toi.
2. Rubrique « Services », puis « Configurer » sur la ligne « DHCP ».
3. Ajoute une réservation d'adresse pour le PC : son adresse MAC et l'adresse IP voulue.
4. Enregistre.

Si la Bbox refuse l'adresse, choisis-en une en dehors de la plage distribuée automatiquement, affichée sur la même page.

</details>

<details>
<summary><b>Box SFR</b></summary>

1. Ouvre l'interface de la box ([192.168.1.1](http://192.168.1.1)) ; l'identifiant est « admin », le mot de passe est imprimé sur l'étiquette de la box.
2. Onglet « Réseau », puis « DHCP ».
3. Dans la section « Adresses statiques », ajoute le PC : son adresse MAC et son adresse IP.
4. Valide.

Les menus varient selon le modèle de box SFR : si tu ne trouves pas « Adresses statiques », cherche « DHCP » ou « IP fixe », ou appelle l'assistance SFR.

</details>

<details>
<summary><b>Autre box ou routeur</b></summary>

1. Ouvre l'interface de ta box ou de ton routeur : souvent `192.168.1.1` ou `192.168.0.1` (l'adresse et le mot de passe sont en général sur l'étiquette).
2. Cherche la partie « DHCP » ou « Réseau local », puis une option nommée « réservation », « bail statique », « bail permanent » ou « IP fixe ».
3. Associe l'adresse MAC du PC à son adresse IP actuelle, puis enregistre.

</details>

> **En Wi-Fi :** si Windows utilise une « adresse matérielle aléatoire » pour ton réseau, l'adresse MAC peut changer et la réservation ne marche plus. Dans *Paramètres → Réseau et Internet → Wi-Fi → ton réseau*, laisse ce réglage désactivé. Un câble Ethernet évite le souci.

Ensuite, redémarre le PC : il doit retrouver la même adresse IP.

## Installation

1. Télécharge `Aria-Vault-<version>.zip` dans les [Releases](../../releases) et décompresse-le dans un dossier qui ne bougera pas (pas Téléchargements).
2. Lance `Aria-Vault.exe`. L'exe n'est pas encore signé (voir la [Code signing policy](docs/SIGNATURE.md)) : si Windows SmartScreen affiche « Windows a protégé votre ordinateur », clique sur « Informations complémentaires » puis « Exécuter quand même ».
3. Quand le pare-feu Windows le demande, autorise Aria sur les **réseaux privés** uniquement.

Aria tourne sans droits administrateur : seule l'autorisation du pare-feu peut demander une confirmation.

## Premiers pas

1. **HTTPS** : choisis ton fournisseur DNS, saisis ton domaine et tes identifiants (le tuto s'affiche à côté), puis enregistre. Le premier certificat arrive en quelques minutes au plus, le temps que le fournisseur publie la preuve.
2. **HTTPS** : réserve l'adresse du PC sur ta box, avec le tuto de ta box.
3. **Réglages** : active « Autoriser les inscriptions », crée ton compte depuis le navigateur (bouton « Ouvrir le coffre »), puis désactive-les aussitôt.
4. **Téléphone** : dans l'appli Bitwarden, à l'écran de connexion, choisis un serveur auto-hébergé et saisis `https://ton-domaine`.
5. **Réglages** : active « Lancer à l'ouverture de session ».
6. **Sauvegardes** : choisis un dossier, idéalement dans OneDrive ou sur un autre disque, active le chiffrement si tu veux, puis exporte tes réglages avec une phrase de passe.
7. **Services** (facultatif) : active la synchronisation instantanée et les e-mails, chacun avec son tuto.

> **Le mot de passe maître est irrécupérable.** Aucune procédure de réinitialisation n'existe : s'il est perdu, le coffre est définitivement illisible.

### Tu viens d'une installation Vaultwarden + Caddy ?

Page **Sauvegardes** → « Importer un ancien coffre… », puis choisis ton ancien dossier `data` (par exemple `C:\Vaultwarden\data`). Pense à désactiver tes anciennes tâches planifiées : Caddy occuperait le port 443.

## Au quotidien

- **Fermer la fenêtre** la renvoie dans la zone de notification : le coffre continue de servir ton téléphone.
- **La fenêtre** s'adapte à ton écran (environ 1280×820 au premier lancement), retient sa taille, sa position et son état « agrandie », et passe de 1 à 3 colonnes selon sa largeur. Le bouton `«` en bas de la barre latérale la replie en icônes.
- **Raccourcis clavier** : `Ctrl+1` à `Ctrl+6` pour les pages, `Ctrl+S` pour sauvegarder, `Ctrl+R` ou `F5` pour actualiser l'état.
- **Clic droit sur l'icône** : ouvrir le coffre, sauvegarder, démarrer ou arrêter le serveur, quitter.
- **Adresse IP** : Aria vérifie au démarrage puis toutes les 10 minutes que ton domaine pointe vers ce PC, et la corrige chez ton fournisseur si besoin (désactivable, page HTTPS). Seules les adresses de réseau local sont publiées.
- **Mettre à jour** : une fois par jour, Aria regarde sur GitHub si une nouvelle version est sortie (désactivable dans Réglages). L'accueil et le menu de l'icône indiquent « à jour » ou la version disponible, et un bandeau propose **Installer**. Ce bouton télécharge le zip de la release depuis ce dépôt, vérifie son empreinte SHA-256 et celle de l'exe (celles publiées dans les notes et calculées par GitHub), remplace l'exe puis lance la nouvelle version. Rien n'est téléchargé sans ton clic, et en cas d'erreur rien n'est modifié. L'ancien exe est gardé en `.old` jusqu'au lancement suivant.
- **À la main**, c'est pareil : lance la nouvelle version, l'ancienne termine ce qu'elle fait (sauvegarde en cours, arrêt du serveur), puis lui laisse la place. Une version plus ancienne ne prend jamais la place d'une plus récente.
- **Veille** : en option (Réglages), le PC ne s'endort pas tout seul tant que le serveur tourne. L'écran peut toujours s'éteindre, et une mise en veille demandée (menu Démarrer, capot fermé) reste possible.

## Sauvegardes

Tout le coffre tient dans un petit zip : base SQLite, clés du serveur, pièces jointes et Sends. Tes mots de passe y restent chiffrés par ton mot de passe maître.

- Aucune sauvegarde tant que tu n'as pas choisi le dossier : Aria ne décide pas à ta place où vont tes données.
- Le serveur se met en pause quelques secondes le temps de la copie, puis repart. Le zip est relu et vérifié avant d'être gardé.
- **Chiffrement (option)** : le zip contient aussi les clés du serveur, les adresses e-mail des comptes et les métadonnées. Avec une phrase de passe, tout le fichier est chiffré (scrypt + AES-256-GCM, par blocs authentifiés) : `aria-vault-<date>.zip.chiffre`. Le zip en clair n'existe que le temps de la copie, à côté du coffre, jamais dans le dossier des sauvegardes. Aria garde la phrase pour ton compte Windows (DPAPI) afin de faire la quotidienne ; **sur un autre PC, il te la faudra : sans elle, ces sauvegardes sont irrécupérables.**
- **Restaurer** met d'abord le coffre actuel de côté, dans `data.avant-restauration-<date>` : rien n'est jamais supprimé. Si une panne coupe Aria en plein remplacement, la restauration est terminée (ou annulée) au lancement suivant.
- Les **réglages d'Aria** (fournisseur, domaine, e-mail, identifiants DNS) s'exportent à part, dans un fichier `.ariavault` entièrement chiffré (scrypt + AES-256-GCM). Sans la phrase de passe, rien n'y est lisible.

## Où sont tes données

Ce qui compte est dans **`%USERPROFILE%\.Aria-Vault`** (chez toi, `C:\Users\<toi>\.Aria-Vault`), propre à ton compte Windows :

| Dossier | Contenu |
| --- | --- |
| `data\` | **Le coffre** : base, clés du serveur, pièces jointes |
| `certificats\` | Certificat Let's Encrypt et compte ACME |
| `config.json` | Réglages d'Aria. Identifiants DNS, clé push, mot de passe e-mail et phrase des sauvegardes y sont chiffrés avec DPAPI, liés à ton compte Windows |
| `aria.log` | Journal (aussi visible dans la page Journal) |
| `fenetre.json` | Taille et position de la fenêtre |

Ce qui se recrée tout seul reste dans `%LOCALAPPDATA%\AriaVault` : Vaultwarden et l'interface web (réextraits et vérifiés à chaque démarrage), le cache de la fenêtre, le fichier de relève.

- Ne copie pas `.Aria-Vault` à la main pendant que le serveur tourne : passe par la page Sauvegardes, qui met le serveur en pause.
- Ne le place pas dans OneDrive : une base synchronisée pendant qu'elle tourne se corrompt. Si ton profil était synchronisé, Aria garderait ses données dans `%LOCALAPPDATA%`.
- Au premier lancement de la v0.8, les données des versions précédentes sont déplacées automatiquement depuis `%LOCALAPPDATA%\AriaVault` (et `AriaVaultwarden` pour les v0.3 et avant). Une coupure en plein déplacement est reprise au lancement suivant.

## Sécurité

- **Vaultwarden n'est jamais joignable directement** : il n'écoute que sur `127.0.0.1:8000`, tout passe par le HTTPS d'Aria.
- **Aucun port ouvert sur la box** : le certificat est obtenu par validation DNS, et le domaine pointe vers une adresse privée.
- **Le panneau `/admin` de Vaultwarden est désactivé** : Aria ne définit aucun jeton d'administration, les réglages passent par Aria.
- **La vraie IP des clients est transmise à Vaultwarden**, et celle envoyée par un client est ignorée : son limiteur de tentatives de connexion fonctionne normalement.
- **Rattaché à Aria** : si Aria se ferme ou plante, Windows arrête Vaultwarden avec elle.
- **Identifiants DNS au plus juste** : les tutos créent des jetons limités à ta zone DNS quand le fournisseur le permet.
- **Connexions réseau** : uniquement ton fournisseur DNS, Let's Encrypt, des résolveurs DNS publics, GitHub (vérification des mises à jour, et téléchargement quand tu cliques sur « Installer »), et, si tu les actives, le relais push de Bitwarden et ta messagerie. Aucune télémétrie. Le détail est dans la [Code signing policy](docs/SIGNATURE.md#confidentialité).

### Limites à connaître

- **Le coffre ne tourne que pendant ta session Windows.** Session fermée ou PC en veille : ton téléphone garde sa copie hors ligne et se resynchronise au retour.
- **IPv4 uniquement** pour la publication automatique de l'adresse.
- **L'exe n'est pas encore signé** : vérifie l'empreinte SHA-256 et l'attestation publiées avec chaque release.
- Si tu importes un ancien coffre qui contient un `config.json` écrit par le panneau `/admin`, ses réglages priment sur ceux d'Aria. Le journal te le signale.
- Projet personnel, sans audit de sécurité indépendant.

## Provenance des composants

| Composant | Source | Vérification |
| --- | --- | --- |
| Vaultwarden | Compilé par GitHub Actions depuis [dani-garcia/vaultwarden](https://github.com/dani-garcia/vaultwarden), au tag de la release officielle, **sans aucune modification** | Tag et commit exacts dans les notes de chaque release et dans `BUILDINFO.txt` |
| Interface web | Release officielle de [dani-garcia/bw_web_builds](https://github.com/dani-garcia/bw_web_builds), appariée à la version de Vaultwarden | SHA-256 comparé au `sha256sums.txt` officiel, la compilation échoue en cas d'écart |
| Aria-Vault.exe et le zip | Compilés par le workflow de ce dépôt | Attestation de provenance Sigstore ([`actions/attest-build-provenance`](https://github.com/actions/attest-build-provenance)) |

Pour vérifier que le fichier téléchargé sort bien de ce dépôt (avec le [CLI GitHub](https://cli.github.com), `winget install GitHub.cli`) :

```powershell
gh attestation verify .\Aria-Vault.exe --owner ZedjeK
```

**Suivi de Vaultwarden** : chaque jour, le workflow *Veille Vaultwarden* regarde si une nouvelle version officielle est sortie. Si oui, il compile une nouvelle version d'Aria et prépare sa release **en brouillon** : elle n'est publiée qu'après relecture.

## Compiler soi-même

Le plus simple, sans rien installer : **fork** de ce dépôt, onglet **Actions** → *Compiler Aria Vault* → **Run workflow** (release : `aucune`). Compte 20 à 40 minutes la première fois. L'exe se télécharge en bas de la page du run, section **Artifacts**.

En local, avec [Go](https://go.dev/dl/) 1.26 ou plus et [Node.js](https://nodejs.org/) LTS, à partir d'un Vaultwarden déjà compilé (un dossier contenant `vaultwarden.exe` et `web-vault\`) :

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1 -Vaultwarden D:\vaultwarden
```

## Structure

```text
src/
├── main.go                 # appli Wails : fenêtre, icône, instance unique, relève
├── vaultservice.go         # méthodes appelées par l'interface, IP, mises à jour, veille
├── sauvegardes.go          # sauvegardes, restauration, réglages exportables
├── internal/
│   ├── vaultwarden/        # Vaultwarden embarqué : extraction vérifiée, processus enfant
│   ├── tlsproxy/           # HTTPS : certificat Let's Encrypt (CertMagic, DNS-01) et relais
│   ├── dnsprov/            # fournisseurs DNS (libdns) et leurs tutos
│   ├── ipdyn/              # le domaine suit l'adresse IP de ce PC
│   ├── moteur/             # démarrage, arrêt et état du coffre
│   ├── sauvegarde/         # zip du coffre, chiffrement, rotation, restauration, reprise
│   ├── chiffre/            # scrypt + AES-256-GCM, en un bloc ou en flux
│   ├── config/             # réglages, identifiants chiffrés DPAPI, export .ariavault
│   ├── courriel/           # e-mail de test (SMTP)
│   ├── maj/                # nouvelle version publiée ? installation vérifiée en un clic
│   ├── eveil/              # empêcher la mise en veille
│   ├── fenetre/            # taille et position de la fenêtre retenues
│   ├── releve/             # une nouvelle version prend la place de l'ancienne
│   ├── demarrage/          # lancement à l'ouverture de session
│   ├── paths/              # emplacements, migration depuis Aria Vaultwarden
│   ├── journal/            # journal de l'interface et aria.log
│   └── ports/              # port occupé : par quel programme
├── frontend/src/           # interface Svelte (polices embarquées)
└── build/                  # logo, icônes de l'exe et de la zone de notification
```

## Licence

Code d'Aria Vault sous licence [MIT](LICENSE). Les composants embarqués gardent leur propre licence : Vaultwarden sous AGPL-3.0, l'interface web sous GPL-3.0. Leurs textes complets sont dans le zip de chaque release, la liste complète dans [LICENCES-TIERS.txt](LICENCES-TIERS.txt).

Aria Vault n'est ni affilié ni approuvé par le projet Vaultwarden, ni par Bitwarden, Inc.
