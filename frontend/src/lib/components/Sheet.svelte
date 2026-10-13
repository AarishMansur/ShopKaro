<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    onclose,
    children
  }: { title: string; onclose: () => void; children: Snippet } = $props();

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose();
  }
</script>

<svelte:window {onkeydown} />

<div
  class="sheet-backdrop"
  role="presentation"
  onclick={(e) => {
    if (e.target === e.currentTarget) onclose();
  }}
>
  <div class="sheet" role="dialog" aria-modal="true" aria-label={title}>
    <div class="sheet-head">
      <div class="sheet-title">{title}</div>
      <button class="btn btn-sm btn-ghost" onclick={onclose} aria-label="Close">✕</button>
    </div>
    {@render children()}
  </div>
</div>
