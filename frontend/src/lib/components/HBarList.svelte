<script lang="ts">
  let {
    rows,
    format = (v: number) => String(Math.round(v))
  }: {
    rows: { label: string; value: number; sub?: string }[];
    format?: (v: number) => string;
  } = $props();

  const max = $derived(Math.max(1, ...rows.map((r) => r.value)));
</script>

{#if rows.length === 0}
  <div class="empty">No sales in this period yet</div>
{:else}
  {#each rows as row}
    <div class="hbar-row">
      <div class="hbar-label" title={row.label}>
        {row.label}
        {#if row.sub}<span class="muted" style="font-weight:500"> · {row.sub}</span>{/if}
      </div>
      <div class="hbar-track">
        <div class="hbar-fill" style="width: {(row.value / max) * 100}%"></div>
      </div>
      <div class="hbar-value">{format(row.value)}</div>
    </div>
  {/each}
{/if}
