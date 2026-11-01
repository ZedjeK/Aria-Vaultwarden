<script>
  // Tutoriel : réserver l'adresse IP de ce PC sur la box (bail DHCP statique).
  // Les libellés suivent les pages d'aide des opérateurs ; ils peuvent varier
  // légèrement selon le modèle et la version du logiciel de la box.
  import Icon from './Icon.svelte'
  import { openLink } from './state.svelte.js'

  let { ip = '', mac = '', carte = '' } = $props()

  const boxes = [
    {
      id: 'freebox',
      nom: 'Freebox',
      lien: 'http://mafreebox.freebox.fr',
      etapes: [
        'Ouvre Freebox OS (mafreebox.freebox.fr) et connecte-toi avec ton mot de passe Freebox OS.',
        'Va dans « Paramètres de la Freebox », passe en « Mode avancé », puis ouvre « DHCP ».',
        'Onglet « Baux statiques » → « Ajouter un bail DHCP statique ».',
        'Saisis l\'adresse MAC et l\'adresse IP de ce PC (encadré « Ce PC »), puis enregistre.',
      ],
      astuce: 'Plus rapide : dans l\'onglet « Baux actifs », fais un clic droit sur ce PC pour rendre son adresse permanente.',
    },
    {
      id: 'livebox',
      nom: 'Livebox',
      lien: 'http://192.168.1.1',
      etapes: [
        'Ouvre l\'interface de la Livebox (192.168.1.1) et connecte-toi avec le mot de passe d\'administration.',
        'Clique sur « Paramètres avancés », puis sur « Réseau ».',
        'Onglet « DHCP », rubrique « Baux DHCP statiques ».',
        'Choisis ce PC dans la liste des équipements, saisis son adresse IP (encadré « Ce PC »), puis clique sur « Ajouter ».',
      ],
      astuce: 'Le PC n\'apparaît pas dans la liste ? Choisis « Équipement » en bas de la liste et saisis son adresse MAC à la main.',
    },
    {
      id: 'bbox',
      nom: 'Bbox',
      lien: 'https://mabbox.bytel.fr',
      etapes: [
        'Ouvre l\'interface de la Bbox (mabbox.bytel.fr, ou 192.168.1.254) et identifie-toi.',
        'Rubrique « Services », puis « Configurer » sur la ligne « DHCP ».',
        'Ajoute une réservation d\'adresse pour ce PC : son adresse MAC et l\'adresse IP voulue (encadré « Ce PC »).',
        'Enregistre.',
      ],
      astuce: 'Si la Bbox refuse l\'adresse, choisis-en une en dehors de la plage distribuée automatiquement, affichée sur la même page.',
    },
    {
      id: 'sfr',
      nom: 'Box SFR',
      lien: 'http://192.168.1.1',
      etapes: [
        'Ouvre l\'interface de la box (192.168.1.1) ; l\'identifiant est « admin », le mot de passe est imprimé sur l\'étiquette de la box.',
        'Onglet « Réseau », puis « DHCP ».',
        'Dans la section « Adresses statiques », ajoute ce PC : son adresse MAC et son adresse IP (encadré « Ce PC »).',
        'Valide.',
      ],
      astuce: 'Les menus varient selon le modèle de box SFR : si tu ne trouves pas « Adresses statiques », cherche « DHCP » ou « IP fixe », ou appelle l\'assistance SFR.',
    },
    {
      id: 'autre',
      nom: 'Autre box',
      etapes: [
        'Ouvre l\'interface de ta box ou de ton routeur : souvent 192.168.1.1 ou 192.168.0.1 (l\'adresse et le mot de passe sont en général sur l\'étiquette).',
        'Cherche la partie « DHCP » ou « Réseau local », puis une option nommée « réservation », « bail statique », « bail permanent » ou « IP fixe ».',
        'Associe l\'adresse MAC de ce PC à son adresse IP actuelle (encadré « Ce PC »), puis enregistre.',
      ],
      astuce: 'La notice ou l\'aide en ligne de ton fournisseur d\'accès décrit cette option sous le nom « attribuer une adresse IP fixe à un appareil ».',
    },
  ]

  let choix = $state('freebox')
  const box = $derived(boxes.find((b) => b.id === choix))
</script>

<div class="tuto">
  <div class="mine" aria-label="Ce PC">
    <span class="eyebrow">Ce PC</span>
    <span><span class="k">adresse IP</span><strong class="mono">{ip || '…'}</strong></span>
    <span><span class="k">adresse MAC</span><strong class="mono">{mac || '…'}</strong></span>
    {#if carte}<span><span class="k">carte réseau</span><span class="mono">{carte}</span></span>{/if}
  </div>

  <div class="etapes">
    <div class="tabs" role="tablist" aria-label="Ta box">
      {#each boxes as b (b.id)}
        <button role="tab" aria-selected={choix === b.id} class:on={choix === b.id} onclick={() => (choix = b.id)}>{b.nom}</button>
      {/each}
    </div>

    <ol class="steps">
      {#each box.etapes as e, i (i)}
        <li>{e}</li>
      {/each}
    </ol>
    {#if box.astuce}<p class="hint">💡 {box.astuce}</p>{/if}
    {#if box.lien}
      <button class="btn-ghost small link" onclick={() => openLink(box.lien)} title={box.lien}>
        Ouvrir l'interface de la {box.nom.replace('Box ', 'box ')} <Icon name="external" size={14} />
      </button>
    {/if}
  </div>

  <div class="notes">
    <p class="hint">
      <strong>En Wi-Fi :</strong> si Windows utilise une « adresse matérielle aléatoire » pour ton réseau, l'adresse MAC peut
      changer et la réservation ne marche plus. Dans Paramètres → Réseau et Internet → Wi-Fi → ton réseau, laisse ce réglage
      désactivé. Un câble Ethernet évite le souci.
    </p>
    <p class="hint">
      Ensuite, redémarre le PC : il doit retrouver la même adresse IP. Aria vérifie de toute façon toutes les 10 minutes que ton
      domaine pointe bien dessus.
    </p>
  </div>
</div>

<style>
  /* Étroit : « Ce PC », étapes, notes. Large : étapes à gauche, le reste à droite. */
  .tuto {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: 'mine' 'etapes' 'notes';
    gap: 14px;
    align-items: start;
  }
  .mine {
    grid-area: mine;
  }
  .etapes {
    grid-area: etapes;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .notes {
    grid-area: notes;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  @container (min-width: 820px) {
    .tuto {
      grid-template-columns: minmax(0, 1.5fr) minmax(260px, 1fr);
      grid-template-areas: 'etapes mine' 'etapes notes';
      column-gap: 28px;
    }
    .mine {
      flex-direction: column;
    }
  }
  .mine {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 18px;
    padding: 10px 12px;
    border: 1px dashed var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
    font-size: 12px;
  }
  .mine > span {
    display: flex;
    gap: 8px;
    align-items: baseline;
  }
  .k {
    color: var(--faint);
  }
  .mine strong {
    color: var(--accent);
    font-weight: 600;
    font-size: 13px;
  }
  .mono {
    font-family: var(--font-mono);
  }
  .tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .tabs button {
    min-height: 34px;
    padding: 0 12px;
    border: 1px solid var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    background: transparent;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .tabs button:hover {
    border-color: var(--accent-2);
    color: var(--text);
  }
  .tabs button.on {
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 10%, transparent);
    color: var(--accent);
  }
  .steps {
    margin: 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 13px;
    line-height: 1.55;
    color: var(--text);
  }
  .steps li::marker {
    color: var(--action);
    font-family: var(--font-mono);
    font-weight: 600;
  }
  .link {
    align-self: flex-start;
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .hint strong {
    color: var(--muted);
    font-weight: 600;
  }
</style>
