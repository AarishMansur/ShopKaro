<script lang="ts">
  import { api } from '$lib/api';
  import { untrack } from 'svelte';
  import { showToast } from '$lib/stores.svelte';
  import type { Product, ProductSuggestion } from '$lib/types';

  let {
    product = null,
    categories = [],
    onclose,
    onsaved
  }: {
    product?: Product | null;
    categories?: string[];
    onclose: () => void;
    onsaved: () => void;
  } = $props();

  let form = $state(
    untrack(() => ({
      name: product?.name ?? '',
      category: product?.category ?? 'General',
      price: product?.price?.toString() ?? '',
      mrp: product?.mrp?.toString() ?? '',
      unit: product?.unit ?? 'pcs',
      aliasesText: product?.aliases?.join(', ') ?? '',
      sort_order: product?.sort_order ?? 0,
      photo_url: product?.photo_url ?? ('' as string | null),
      active: product?.active ?? true
    }))
  );

  let photoFile = $state<File | null>(null);
  let reading = $state(false);
  let saving = $state(false);
  let error = $state('');

  function onPhotoChosen(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    photoFile = input.files?.[0] ?? null;
  }

  async function readPack() {
    if (!photoFile) {
      error = 'Choose a photo of the pack first';
      return;
    }
    reading = true;
    error = '';
    try {
      const s: ProductSuggestion = await api.fromPhoto(photoFile);
      form.name = s.name;
      form.category = s.category;
      form.price = String(s.price);
      form.mrp = String(s.mrp || s.price);
      form.unit = s.unit;
      form.aliasesText = (s.aliases ?? []).join(', ');
      showToast('Gemma read the pack — check the details');
    } catch (e) {
      error = (e as Error).message;
    } finally {
      reading = false;
    }
  }

  async function save() {
    error = '';
    if (!form.name.trim()) {
      error = 'Name is required';
      return;
    }
    const price = parseFloat(form.price);
    if (!Number.isFinite(price) || price < 0) {
      error = 'Enter a valid price';
      return;
    }
    saving = true;
    const payload = {
      name: form.name.trim(),
      category: form.category.trim() || 'General',
      price,
      mrp: parseFloat(form.mrp) || price,
      unit: form.unit.trim() || 'pcs',
      aliases: form.aliasesText
        .split(',')
        .map((a) => a.trim())
        .filter(Boolean),
      sort_order: form.sort_order,
      photo_url: form.photo_url || null,
      active: form.active
    };
    try {
      if (product) {
        await api.updateProduct(product.id, payload);
        showToast('Product updated');
      } else {
        await api.createProduct(payload);
        showToast('Product added');
      }
      onsaved();
      onclose();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      saving = false;
    }
  }
</script>

<div class="field">
  <label for="p-name">Product name</label>
  <input id="p-name" class="input" bind:value={form.name} placeholder="Amul Taaza Milk 500ml" />
</div>

<div class="grid-2">
  <div class="field">
    <label for="p-price">Selling price (₹)</label>
    <input id="p-price" class="input" type="number" inputmode="decimal" min="0" step="0.5" bind:value={form.price} />
  </div>
  <div class="field">
    <label for="p-mrp">MRP (₹)</label>
    <input id="p-mrp" class="input" type="number" inputmode="decimal" min="0" step="0.5" bind:value={form.mrp} />
  </div>
</div>

<div class="grid-2">
  <div class="field">
    <label for="p-cat">Category</label>
    <input id="p-cat" class="input" list="cat-list" bind:value={form.category} />
    <datalist id="cat-list">
      {#each categories as c}
        <option value={c}></option>
      {/each}
    </datalist>
  </div>
  <div class="field">
    <label for="p-unit">Unit</label>
    <input id="p-unit" class="input" bind:value={form.unit} placeholder="pkt / kg / ltr" />
  </div>
</div>

<div class="field">
  <label for="p-aliases">Voice/search aliases (comma separated)</label>
  <input id="p-aliases" class="input" bind:value={form.aliasesText} placeholder="doodh, milk, taaza" />
</div>

<div class="field">
  <label for="p-photo">Pack photo (optional)</label>
  <input id="p-photo" class="input" type="file" accept="image/*" capture="environment" onchange={onPhotoChosen} />
  <button class="btn btn-sm" style="margin-top:8px" onclick={readPack} disabled={reading || !photoFile}>
    {#if reading}<span class="spinner"></span>{/if}
    {reading ? 'Reading pack…' : 'Fill details from photo (Gemma)'}
  </button>
</div>

{#if error}
  <p class="down" style="margin: 8px 0">{error}</p>
{/if}

<div class="row" style="margin-top: 12px">
  <button class="btn btn-primary" style="flex:1" onclick={save} disabled={saving}>
    {saving ? 'Saving…' : product ? 'Save changes' : 'Add product'}
  </button>
  <button class="btn" onclick={onclose}>Cancel</button>
</div>
