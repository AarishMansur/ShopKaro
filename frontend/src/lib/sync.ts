import { api } from './api';
import { cacheProducts, listQueued, loadCachedProducts, queueCount, removeQueued } from './offline';
import { app, showToast } from './stores.svelte';
import type { PendingSale } from './types';

let flushing = false;

// Drain the offline queue. Safe to call often: idempotent server-side via
// client_id, and a mutex avoids overlapping batches.
export async function flushQueue(): Promise<number> {
  if (flushing) return 0;
  flushing = true;
  try {
    const queued = await listQueued();
    if (queued.length === 0) return 0;
    const res = await api.batchSales(queued as unknown[]);
    await removeQueued(queued.map((q: PendingSale) => q.client_id));
    const left = await queueCount();
    app.pendingCount = left;
    if (res.accepted > 0) {
      await refreshRecent();
    }
    return res.accepted;
  } catch {
    // still offline (or API down) — keep the queue for the next attempt
    return 0;
  } finally {
    flushing = false;
  }
}

export async function refreshRecent(): Promise<void> {
  try {
    app.recent = await api.recentSales(25);
  } catch {
    /* offline: keep last known list */
  }
}

export async function refreshProducts(): Promise<void> {
  try {
    app.products = await api.products();
    await cacheProducts(app.products);
  } catch {
    if (app.products.length === 0) {
      app.products = await loadCachedProducts();
    }
  }
}

export async function refreshFeatures(): Promise<void> {
  try {
    app.features = await api.features();
    if (app.features.shop?.name) app.shopName = app.features.shop.name;
  } catch {
    /* offline */
  }
}

// Records a sale offline-first: it hits IndexedDB immediately, then syncs.
export async function recordSale(input: {
  product_id: number;
  qty: number;
  unit_price: number;
  source?: string;
}): Promise<void> {
  const pending: PendingSale = {
    client_id: crypto.randomUUID ? crypto.randomUUID() : String(Date.now() + Math.random()),
    product_id: input.product_id,
    qty: input.qty,
    unit_price: input.unit_price,
    sold_at: new Date().toISOString(),
    source: input.source ?? 'tap'
  };
  const { queueSale } = await import('./offline');
  await queueSale(pending);
  app.pendingCount = await queueCount();
  if (app.online) await flushQueue();
}

export function startSync(): void {
  window.addEventListener('online', () => {
    app.online = true;
    flushQueue();
    refreshRecent();
  });
  window.addEventListener('offline', () => {
    app.online = false;
    showToast('Offline — sales are saved on this phone');
  });

  refreshProducts().then(() => {
    app.pendingCount = 0;
    queueCount().then((n) => (app.pendingCount = n));
  });
  refreshFeatures();
  refreshRecent();
  flushQueue();

  setInterval(() => {
    if (navigator.onLine) flushQueue();
  }, 20000);
}
