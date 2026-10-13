import type { Features, Product, Sale } from './types';

// Single reactive app state (Svelte 5 runes). Components read and mutate this
// directly; persistence lives in offline.ts / sync.ts.
export const app = $state({
  products: [] as Product[],
  recent: [] as Sale[],
  pendingCount: 0,
  online: typeof navigator === 'undefined' ? true : navigator.onLine,
  features: null as Features | null,
  shopName: 'ShopKaro',
  toast: '',
  loading: false
});

let toastTimer: ReturnType<typeof setTimeout> | undefined;

export function showToast(message: string): void {
  app.toast = message;
  if (toastTimer) clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    app.toast = '';
  }, 2600);
}

export function categories(): string[] {
  const set = new Set<string>();
  for (const p of app.products) if (p.active) set.add(p.category);
  return [...set].sort((a, b) => a.localeCompare(b));
}

export function productById(id: number): Product | undefined {
  return app.products.find((p) => p.id === id);
}
