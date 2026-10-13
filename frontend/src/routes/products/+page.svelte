<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { inr } from '$lib/format';
  import Monogram from '$lib/components/Monogram.svelte';
  import ProductForm from '$lib/components/ProductForm.svelte';
  import Sheet from '$lib/components/Sheet.svelte';
  import { app, categories, showToast } from '$lib/stores.svelte';
  import { refreshProducts } from '$lib/sync';
  import type { Product } from '$lib/types';

  let editing = $state<Product | null>(null);
  let creating = $state(false);
  let deleting = $state(0);

  onMount(refreshProducts);

  async function remove(p: Product) {
    if (!confirm(`Remove ${p.name}?`)) return;
    deleting = p.id;
    try {
      const res = await api.deleteProduct(p.id);
      showToast(res.result === 'deleted' ? 'Product deleted' : 'Product archived (it has sales history)');
      await refreshProducts();
    } catch (e) {
      showToast((e as Error).message);
    } finally {
      deleting = 0;
    }
  }
</script>

<div class="page-head">
  <div>
    <div class="page-title">Products</div>
    <div class="muted" style="font-size: 13px">{app.products.filter((p) => p.active).length} active</div>
  </div>
  <button class="btn btn-primary" onclick={() => (creating = true)}>+ Add product</button>
</div>

{#if app.products.length === 0}
  <div class="empty card">No products yet — add the first one.</div>
{:else}
  <div class="card">
    {#each app.products as p (p.id)}
      <div class="list-row">
        <Monogram name={p.name} small />
        <div style="flex: 1; min-width: 0">
          <div style="font-weight: 600">{p.name}</div>
          <div class="muted" style="font-size: 12px">
            {p.category} · {inr(p.price)} / {p.unit}
            {#if !p.active}<span class="pill warn">archived</span>{/if}
          </div>
        </div>
        <button class="btn btn-sm" onclick={() => (editing = p)}>Edit</button>
        <button class="btn btn-sm btn-danger" onclick={() => remove(p)} disabled={deleting === p.id}>
          {deleting === p.id ? '…' : 'Remove'}
        </button>
      </div>
    {/each}
  </div>
{/if}

{#if creating}
  <Sheet title="Add product" onclose={() => (creating = false)}>
    <ProductForm categories={categories()} onclose={() => (creating = false)} onsaved={refreshProducts} />
  </Sheet>
{/if}

{#if editing}
  <Sheet title="Edit product" onclose={() => (editing = null)}>
    <ProductForm product={editing} categories={categories()} onclose={() => (editing = null)} onsaved={refreshProducts} />
  </Sheet>
{/if}
