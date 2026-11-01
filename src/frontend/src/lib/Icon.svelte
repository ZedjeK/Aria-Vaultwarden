<script>
  // Icônes au trait, toutes en 24×24.
  let { name, size = 20 } = $props()
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  {#if name === 'home'}
    <path d="M4 11l8-7 8 7" /><path d="M6 10v10h12V10" /><path d="M10 20v-5h4v5" />
  {:else if name === 'lock'}
    <rect x="5" y="11" width="14" height="9" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /><path d="M12 15v2" />
  {:else if name === 'settings'}
    <path d="M4 7h10" /><path d="M18 7h2" /><circle cx="16" cy="7" r="2" /><path d="M4 17h2" /><path d="M10 17h10" /><circle cx="8" cy="17" r="2" />
  {:else if name === 'journal'}
    <path d="M9 6h11" /><path d="M9 12h11" /><path d="M9 18h11" /><path d="M4 6h.01" /><path d="M4 12h.01" /><path d="M4 18h.01" />
  {:else if name === 'backup'}
    <rect x="3" y="4" width="18" height="5" rx="1.5" /><path d="M5 9v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V9" /><path d="M10 13h4" />
  {:else if name === 'vault'}
    <rect x="3" y="4" width="18" height="15" rx="2" /><circle cx="12" cy="11.5" r="4" /><path d="M12 9.5v2l1.5 1" /><path d="M6 19v2" /><path d="M18 19v2" />
  {:else if name === 'services'}
    <path d="M4 6h16v12H4z" /><path d="M4 7l8 6 8-6" /><path d="M18 2v3" /><path d="M21 4h-3" />
  {:else if name === 'replier'}
    <path d="M11 7l-5 5 5 5" /><path d="M18 7l-5 5 5 5" />
  {:else if name === 'deplier'}
    <path d="M6 7l5 5-5 5" /><path d="M13 7l5 5-5 5" />
  {:else if name === 'external'}
    <path d="M14 4h6v6" /><path d="M20 4l-9 9" /><path d="M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5" />
  {/if}
</svg>
