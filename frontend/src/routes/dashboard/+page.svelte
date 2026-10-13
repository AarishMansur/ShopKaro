<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { compact, dayLabel, inr, pct } from '$lib/format';
  import BarChart from '$lib/components/BarChart.svelte';
  import HBarList from '$lib/components/HBarList.svelte';
  import type { DashboardSummary, DayPoint, HourPoint, RankedProduct } from '$lib/types';

  let days = $state(7);
  let summary = $state<DashboardSummary | null>(null);
  let best = $state<RankedProduct[]>([]);
  let slow = $state<RankedProduct[]>([]);
  let byDay = $state<DayPoint[]>([]);
  let byHour = $state<HourPoint[]>([]);
  let loading = $state(false);
  let error = $state('');

  const chartDays = $derived(days === 7 ? 7 : 28);

  async function load() {
    loading = true;
    error = '';
    try {
      const [s, b, sl, d, h] = await Promise.all([
        api.summary(days),
        api.bestSellers(days, 8),
        api.slowMovers(days, 6),
        api.byDay(chartDays),
        api.byHour(days)
      ]);
      summary = s;
      best = b;
      slow = sl;
      byDay = d;
      byHour = h;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<div class="page-head">
  <div class="page-title">Stats</div>
  <div class="chips" style="padding: 0">
    <button class="chip" class:active={days === 7} onclick={() => { days = 7; load(); }}>7 days</button>
    <button class="chip" class:active={days === 28} onclick={() => { days = 28; load(); }}>28 days</button>
  </div>
</div>

{#if error}
  <div class="card"><p class="down">{error}</p></div>
{/if}

{#if loading && !summary}
  <div class="empty"><span class="spinner" style="display:inline-block"></span> Loading stats…</div>
{:else if summary}
  <div class="stat-grid">
    <div class="card stat">
      <div class="label">Revenue ({days}d)</div>
      <div class="big">{inr(summary.revenue)}</div>
      <div class="sub">
        <span class={summary.revenue_change >= 0 ? 'up' : 'down'}>{pct(summary.revenue_change)}</span> vs previous
      </div>
    </div>
    <div class="card stat">
      <div class="label">Bills</div>
      <div class="big">{summary.orders}</div>
      <div class="sub">{summary.units} units sold</div>
    </div>
    <div class="card stat">
      <div class="label">Average bill</div>
      <div class="big">{inr(summary.avg_basket)}</div>
      <div class="sub">{summary.active_products} products sold</div>
    </div>
    <div class="card stat">
      <div class="label">Busiest hour</div>
      <div class="big">{summary.busiest_hour}:00</div>
      <div class="sub">Best day: {summary.best_day}</div>
    </div>
  </div>

  <div class="section">
    <h2 class="section-title">Revenue by day</h2>
    <div class="card">
      <BarChart data={byDay.map((d) => ({ label: dayLabel(d.date), value: d.revenue }))} format={(v) => compact(v)} />
    </div>
  </div>

  <div class="two-col section">
    <div>
      <h2 class="section-title">Best sellers</h2>
      <div class="card">
        <HBarList rows={best.map((b) => ({ label: b.name, value: b.qty, sub: inr(b.revenue) }))} />
      </div>
    </div>
    <div>
      <h2 class="section-title">Slow movers</h2>
      <div class="card">
        <HBarList rows={slow.map((b) => ({ label: b.name, value: b.qty, sub: inr(b.revenue) }))} />
      </div>
    </div>
  </div>

  <div class="section">
    <h2 class="section-title">Sales by hour of day</h2>
    <div class="card">
      <BarChart
        data={byHour.map((h) => ({ label: h.hour % 3 === 0 ? String(h.hour) : '', value: h.revenue }))}
        format={(v) => compact(v)}
        accent="#1663c7"
      />
    </div>
  </div>

  <div class="section">
    <h2 class="section-title">By category</h2>
    <div class="card">
      <HBarList
        rows={summary.category_revenue.map((c) => ({ label: c.category, value: c.revenue, sub: `${c.qty} units` }))}
        format={(v) => compact(v)}
      />
    </div>
  </div>
{/if}
