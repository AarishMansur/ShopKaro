<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { inr, timeLabel } from '$lib/format';
  import AiEntrySheet from '$lib/components/AiEntrySheet.svelte';
  import Monogram from '$lib/components/Monogram.svelte';
  import { app, categories, showToast } from '$lib/stores.svelte';
  import { flushQueue, recordSale, refreshRecent } from '$lib/sync';
  import type { Product } from '$lib/types';

  let search = $state('');
  let category = $state('All');
  let multiplier = $state(1);
  let showAi = $state(false);
  let todayRevenue = $state(0);
  let todayOrders = $state(0);
  const flash = $state<Record<number, number>>({});

  const multipliers = [1, 2, 3, 5];

  const filtered = $derived.by(() => {
    const q = search.trim().toLowerCase();
    return app.products.filter((p) => {
      if (!p.active) return false;
      if (category !== 'All' && p.category !== category) return false;
      if (!q) return true;
      if (p.name.toLowerCase().includes(q)) return true;
      return p.aliases.some((a) => a.toLowerCase().includes(q));
    });
  });

  async function refreshToday() {
    try {
      const s = await api.summary(1);
      todayRevenue = s.revenue;
      todayOrders = s.orders;
    } catch {
      /* offline */
    }
  }

  async function tap(p: Product) {
    if (navigator.vibrate) navigator.vibrate(18);
    flash[p.id] = (flash[p.id] ?? 0) + multiplier;
    setTimeout(() => {
      delete flash[p.id];
    }, 900);
    await recordSale({ product_id: p.id, qty: multiplier, unit_price: p.price });
    refreshToday();
  }

  async function adjustQty(id: number, delta: number) {
    const sale = app.recent.find((s) => s.id === id);
    if (!sale) return;
    const next = sale.qty + delta;
    try {
      if (next <= 0) {
        await api.deleteSale(id);
        showToast('Sale removed');
      } else {
        await api.updateSaleQty(id, next);
      }
      await refreshRecent();
      refreshToday();
    } catch (e) {
      showToast((e as Error).message);
    }
  }

  async function undo(id: number) {
    try {
      await api.deleteSale(id);
      showToast('Sale removed');
      await refreshRecent();
      refreshToday();
    } catch (e) {
      showToast((e as Error).message);
    }
  }

  onMount(() => {
    refreshToday();
  });
</script>

<div class="page-head">
  <div>
    <div class="page-title">Sell</div>
    <div class="muted" style="font-size: 13px">
      Today {inr(todayRevenue)} · {todayOrders} bills
      {#if app.pendingCount > 0}
        · <span class="pill warn">{app.pendingCount} waiting to sync</span>
      {/if}
    </div>
  </div>
  <span class="pill {app.online ? '' : 'warn'}">{app.online ? 'Online' : 'Offline mode'}</span>
</div>

<div class="row" style="margin-bottom: 8px; gap: 8px; align-items: stretch">
  <input class="input" placeholder="Search products…" bind:value={search} />
</div>

<div class="chips" role="group" aria-label="Quantity per tap">
  {#each multipliers as m}
    <button class="chip" class:active={multiplier === m} onclick={() => (multiplier = m)}>×{m}</button>
  {/each}
</div>

<div class="chips" role="group" aria-label="Categories">
  <button class="chip" class:active={category === 'All'} onclick={() => (category = 'All')}>All</button>
  {#each categories() as c}
    <button class="chip" class:active={category === c} onclick={() => (category = c)}>{c}</button>
  {/each}
</div>

{#if app.products.length === 0}
  <div class="empty card">
    <p>No products yet.</p>
    <a class="btn btn-primary" href="/products">Add your first product</a>
  </div>
{:else if filtered.length === 0}
  <div class="empty">No products match “{search}”</div>
{:else}
  <div class="tile-grid" style="margin-top: 4px">
    {#each filtered as p (p.id)}
      <button class="tile" onclick={() => tap(p)} aria-label="Sell {p.name}">
        <div class="row" style="justify-content: space-between">
          <Monogram name={p.name} />
          {#if flash[p.id]}
            <span class="pill">+{flash[p.id]}</span>
          {/if}
        </div>
        <div class="tile-name">{p.name}</div>
        <div class="tile-price">
          {inr(p.price)} <span class="tile-unit">/ {p.unit}</span>
        </div>
      </button>
    {/each}
  </div>
{/if}

<div class="section">
  <h2 class="section-title">Recent sales</h2>
  <div class="card">
    {#if app.recent.length === 0}
      <div class="empty">Tap a product to record the first sale</div>
    {:else}
      {#each app.recent.slice(0, 15) as sale (sale.id)}
        <div class="list-row">
          <Monogram name={sale.product_name || '?'} small />
          <div style="flex: 1; min-width: 0">
            <div style="font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              {sale.product_name}
            </div>
            <div class="muted" style="font-size: 12px">
              {timeLabel(sale.sold_at)} · {inr(sale.unit_price)} each
              {#if sale.source !== 'tap'}
                · {sale.source}
              {/if}
            </div>
          </div>
          <div style="text-align: right">
            <div style="font-weight: 800">{inr(sale.total)}</div>
            <div class="muted" style="font-size: 12px">×{sale.qty}</div>
          </div>
          <div class="row" style="gap: 6px">
            <button class="qty-btn" onclick={() => adjustQty(sale.id, -1)} aria-label="Decrease quantity">−</button>
            <button class="qty-btn" onclick={() => adjustQty(sale.id, 1)} aria-label="Increase quantity">+</button>
            <button class="qty-btn" onclick={() => undo(sale.id)} aria-label="Undo sale" title="Undo">↺</button>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<button class="fab" onclick={() => (showAi = true)} aria-label="Voice or text entry">
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
    <rect x="9" y="3" width="6" height="11" rx="3" />
    <path d="M5 11a7 7 0 0 0 14 0M12 18v3" />
  </svg>
</button>

{#if showAi}
  <AiEntrySheet onclose={() => (showAi = false)} />
{/if}
