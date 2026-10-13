<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { app } from '$lib/stores.svelte';
  import { startSync } from '$lib/sync';

  let { children } = $props();

  const icons = {
    sell: '<rect x="3" y="3" width="7.5" height="7.5" rx="2"/><rect x="13.5" y="3" width="7.5" height="7.5" rx="2"/><rect x="3" y="13.5" width="7.5" height="7.5" rx="2"/><rect x="13.5" y="13.5" width="7.5" height="7.5" rx="2"/>',
    products:
      '<path d="M3 7.5 12 3l9 4.5v9L12 21l-9-4.5v-9z"/><path d="m3 7.5 9 4.5 9-4.5M12 12v9"/>',
    stats: '<path d="M3 21h18M7 21V11M12 21V4M17 21v-7"/>',
    reorder:
      '<circle cx="9.5" cy="20" r="1.4"/><circle cx="17.5" cy="20" r="1.4"/><path d="M2 4h2.6l2.4 11.5h11L20 8H7"/>',
    report:
      '<path d="M6 3h9l5 5v13a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z"/><path d="M14 3v6h6M9 13h6M9 17h5"/>'
  };

  const nav = [
    { href: '/', label: 'Sell', icon: icons.sell },
    { href: '/products', label: 'Products', icon: icons.products },
    { href: '/dashboard', label: 'Stats', icon: icons.stats },
    { href: '/reorder', label: 'Reorder', icon: icons.reorder },
    { href: '/report', label: 'Report', icon: icons.report }
  ];

  const path = $derived(page.url.pathname);
  const isActive = (href: string) => (href === '/' ? path === '/' : path.startsWith(href));

  onMount(() => {
    startSync();
    if ('serviceWorker' in navigator && import.meta.env.PROD) {
      navigator.serviceWorker.register('/service-worker.js').catch(() => undefined);
    }
  });
</script>

<svelte:head>
  <title>ShopKaro — {app.shopName}</title>
</svelte:head>

<div class="app">
  <aside class="sidebar">
    <div class="brand">
      <img src="/icon.svg" alt="" />
      <b>ShopKaro</b>
    </div>
    <nav>
      {#each nav as item}
        <a href={item.href} class:active={isActive(item.href)}>
          {@html `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${item.icon}</svg>`}
          {item.label}
        </a>
      {/each}
    </nav>
    <div class="side-foot">
      <span class="pill {app.online ? '' : 'warn'}">
        {app.online ? 'Online' : 'Offline'}
        {#if app.pendingCount > 0}· {app.pendingCount} waiting{/if}
      </span>
      {#if app.features}
        <span>Gemma: {app.features.gemma ? 'on' : 'off'}</span>
        <span>Forecast: {app.features.tabpfn ? 'TabPFN' : 'fallback'}</span>
        <span>Jobs: {app.features.scheduler}</span>
      {/if}
      <span>{app.shopName}</span>
    </div>
  </aside>

  <main class="main">
    {@render children()}
  </main>

  <nav class="bottomnav">
    {#each nav as item}
      <a href={item.href} class:active={isActive(item.href)}>
        {@html `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${item.icon}</svg>`}
        {item.label}
      </a>
    {/each}
  </nav>
</div>

{#if app.toast}
  <div class="toast" role="status">{app.toast}</div>
{/if}
