# Code signing policy

> **Statut : pas encore signé.** Aria Vault prépare une candidature au programme gratuit de
> [SignPath Foundation](https://signpath.org) pour les projets open source. Tant qu'elle n'est pas
> acceptée, `Aria-Vault.exe` n'est pas signé : vérifie l'empreinte SHA-256 et l'attestation de
> provenance publiées avec chaque release (voir le [README](../README.md#provenance-des-composants)).

<!-- À décommenter une fois le projet accepté par SignPath Foundation :
Free code signing provided by [SignPath.io](https://about.signpath.io), certificate by [SignPath Foundation](https://signpath.org).
-->

## Ce qui est signé

Uniquement `Aria-Vault.exe`, compilé par le workflow [`build.yml`](../.github/workflows/build.yml) de ce
dépôt sur un runner GitHub Actions, à partir du code source de ce dépôt. Les composants amont
embarqués dans l'exe (Vaultwarden, compilé sans modification depuis son source officiel, et
l'interface web officielle vérifiée par SHA-256) ne sont pas signés par ce projet.

Chaque demande de signature est créée par le workflow lors d'une release, puis **approuvée à la
main** sur SignPath avant d'être signée. Une compilation de test n'est jamais signée.

## Équipe

| Rôle | Membres |
| --- | --- |
| Auteurs (committers) | [ZedjeK](https://github.com/ZedjeK) |
| Relecteurs (reviewers) | [ZedjeK](https://github.com/ZedjeK) |
| Approbateurs (approvers) | [ZedjeK](https://github.com/ZedjeK) |

Toute contribution extérieure (pull request) est relue avant fusion. Tous les membres utilisent
l'authentification multifacteur sur GitHub et sur SignPath.

## Confidentialité

Aria Vault n'envoie aucune télémétrie et ne transmet rien à ses auteurs. Les seules connexions
réseau sont celles nécessaires à ce que tu lui demandes :

| Destination | Pourquoi | Désactivable |
| --- | --- | --- |
| Ton fournisseur DNS (DuckDNS, Cloudflare, OVHcloud ou deSEC) | Preuve DNS-01 pour le certificat, et publication de l'IP de ce PC | La publication de l'IP : oui (page HTTPS). La preuve DNS : non, c'est elle qui fournit le certificat |
| Let's Encrypt | Obtention et renouvellement du certificat HTTPS | Non (indispensable au HTTPS) |
| Résolveurs DNS publics (1.1.1.1, 9.9.9.9, 8.8.8.8) | Lire l'adresse publiée pour ton domaine | Oui, avec la publication de l'IP |
| `api.github.com` | Une fois par jour : y a-t-il une nouvelle version d'Aria Vault ? | Oui (page Réglages) |
| `github.com` (releases de ce dépôt) | Téléchargement de la nouvelle version, uniquement quand tu cliques sur « Installer » | Oui : il suffit de ne pas cliquer |

Vaultwarden, embarqué, peut télécharger les icônes des sites enregistrés dans ton coffre quand
une appli Bitwarden les demande (comportement par défaut de Vaultwarden, voir sa
[documentation](https://github.com/dani-garcia/vaultwarden/wiki)). Les politiques de
confidentialité de ces services s'appliquent à leurs connexions :
[DuckDNS](https://www.duckdns.org) et [deSEC](https://desec.io) (sur leurs sites),
[Cloudflare](https://www.cloudflare.com/privacypolicy/),
[OVHcloud](https://www.ovhcloud.com/fr/terms-and-conditions/privacy-policy/),
[Let's Encrypt](https://letsencrypt.org/privacy/),
[GitHub](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement).

## Désinstallation

Aria Vault ne s'installe pas : quitte-le (clic droit sur l'icône → Quitter), désactive « Lancer
à l'ouverture de session » au préalable si tu l'avais activé, puis supprime l'exe. Tes données
restent dans `%LOCALAPPDATA%\AriaVault` tant que tu ne supprimes pas ce dossier toi-même.

---

## Pour le mainteneur : activer la signature

1. Déposer la candidature sur [signpath.org/apply](https://signpath.org/apply) en indiquant ce
   dépôt et cette page.
2. Une fois accepté, dans SignPath : créer le projet (slug `aria-vault`), une politique de
   signature (slug `release-signing`) avec approbation manuelle, et des restrictions de métadonnées
   « Product name = Aria Vault » et « Product version » égale à la version de la release.
3. Dans GitHub, *Settings → Secrets and variables → Actions* : le secret `SIGNPATH_API_TOKEN` et la
   variable `SIGNPATH_ORGANIZATION_ID` (et, si les slugs diffèrent, `SIGNPATH_PROJECT_SLUG` et
   `SIGNPATH_POLICY_SLUG`). Vérifier aussi la version de l'action
   `signpath/github-action-submit-signing-request` utilisée dans `build.yml`.
4. Décommenter la ligne d'attribution en haut de cette page et retirer le statut « pas encore signé »
   du README et des notes de release.
