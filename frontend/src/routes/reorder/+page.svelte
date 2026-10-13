<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { dateLabel, inr } from '$lib/format';
  import Monogram from '$lib/components/Monogram.svelte';
  import { showToast } from '$lib/stores.svelte';
  import type { Forecast } from '$lib/types';

  let forecast = $state<Forecast | null>(null);
  let loading = $state(false);
  let saving = $state(false);
  let error = $state('');

  async function load() {
    loading = true;
    error = '';
    try {
      forecast = await api.forecast();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function saveReport() {
    saving = true;
    try {
      await api.runJob('reorder_forecast');
      showToast('Reorder report saved for Monday');
    } catch (e) {
      showToast((e as Error).message);
    } finally {
      saving = false;
    }
  }

  onMount(load);
</script>

{#snippet confidencePill(c: string)}
  <span class="pill {c === 'high' ? '' : c === 'medium' ? 'warn' : 'neutral'}">{c}</span>
{/snippet}

<div class="page-head">
  <div>
    <div class="page-title">Reorder list</div>
    <div class="muted" style="font-size: 13px">
      {#if forecast}
        Next week · {forecast.method === 'tabpfn' ? 'TabPFN forecast' : 'trend fallback'} ·
        {dateLabel(forecast.generated_at)}
      {/if}
    </div>
  </div>
  <div class="row" style="gap: 8px">
    <button class="btn btn-sm" onclick={load} disabled={loading}>
      {loading ? 'Forecasting…' : 'Refresh'}
    </button>
    <button class="btn btn-sm btn-primary" onclick={saveReport} disabled={saving || !forecast}>
      {saving ? 'Saving…' : 'Save as report'}
    </button>
  </div>
</div>

{#if error}
  <div class="card"><p class="down">{error}</p></div>
{/if}

{#if forecast}
  <div class="card" style="margin-bottom: 12px">
    <div class="muted" style="font-size: 14px">{forecast.note}</div>
  </div>

  {#if forecast.rows.filter((r) => r.suggested_qty > 0).length === 0}
    <div class="empty card">Not enough sales history yet — keep tapping sales for a week or two.</div>
  {:else}
    <div class="cards-grid">
      {#each forecast.rows.filter((r) => r.suggested_qty > 0) as row (row.product_id)}
        <div class="card">
          <div class="row" style="gap: 10px">
            <Monogram name={row.name} small />
            <div style="flex: 1; min-width: 0">
              <div style="font-weight: 700; font-size: 14px">{row.name}</div>
              <div class="muted" style="font-size: 12px">{row.category} · {inr(row.price)}/{row.unit}</div>
            </div>
            {@render confidencePill(row.confidence)}
          </div>
          <div class="row" style="margin-top: 10px; justify-content: space-between; gap: 6px">
            <div style="text-align: center">
              <div class="muted" style="font-size: 11px">LAST WEEK</div>
              <div style="font-weight: 700">{row.last_week_qty}</div>
            </div>
            <div class="muted">→</div>
            <div style="text-align: center">
              <div class="muted" style="font-size: 11px">FORECAST</div>
              <div style="font-weight: 700">{row.forecast_qty}</div>
            </div>
            <div class="muted">→</div>
            <div style="text-align: center">
              <div class="muted" style="font-size: 11px">ORDER</div>
              <div style="font-weight: 800; color: var(--accent-dark); font-size: 18px">{row.suggested_qty}</div>
            </div>
          </div>
          <div class="muted" style="font-size: 12px; margin-top: 8px">
            ≈ {inr(row.suggested_qty * row.price)} at current price · 4-week avg {row.avg_4week_qty}
          </div>
        </div>
      {/each}
    </div>
  {/if}
{:else if loading}
  <div class="empty"><span class="spinner" style="display:inline-block"></span> Building forecast from your tap history…</div>
{/if}
