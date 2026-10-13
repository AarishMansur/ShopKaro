<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { inr, pct, timeLabel } from '$lib/format';
  import { showToast } from '$lib/stores.svelte';
  import type { JobStatus, SummaryResponse } from '$lib/types';

  let lang = $state('en');
  let data = $state<SummaryResponse | null>(null);
  let jobs = $state<JobStatus | null>(null);
  let loading = $state(false);
  let regenerating = $state(false);
  let error = $state('');

  async function load() {
    loading = true;
    error = '';
    try {
      data = await api.weeklySummary(lang);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function regenerate() {
    regenerating = true;
    try {
      await api.runJob('weekly_summary', lang);
      showToast('Weekly report saved');
      await load();
      jobs = await api.jobs();
    } catch (e) {
      showToast((e as Error).message);
    } finally {
      regenerating = false;
    }
  }

  function switchLang(next: string) {
    lang = next;
    load();
  }

  onMount(async () => {
    await load();
    jobs = await api.jobs().catch(() => null);
  });
</script>

<div class="page-head">
  <div>
    <div class="page-title">Weekly report</div>
    <div class="muted" style="font-size: 13px">Last 7 days, in plain language</div>
  </div>
  <div class="chips" style="padding: 0">
    <button class="chip" class:active={lang === 'en'} onclick={() => switchLang('en')}>English</button>
    <button class="chip" class:active={lang === 'hi'} onclick={() => switchLang('hi')}>हिन्दी</button>
  </div>
</div>

{#if error}
  <div class="card"><p class="down">{error}</p></div>
{/if}

{#if loading && !data}
  <div class="empty"><span class="spinner" style="display:inline-block"></span> Writing your summary…</div>
{:else if data}
  <div class="card">
    <div class="row" style="margin-bottom: 10px">
      <span class="pill {data.engine === 'gemma' ? '' : 'neutral'}">
        {data.engine === 'gemma' ? 'Written by Gemma' : 'Template summary'}
      </span>
      <div style="flex:1"></div>
      <button class="btn btn-sm btn-primary" onclick={regenerate} disabled={regenerating}>
        {regenerating ? 'Saving…' : 'Regenerate & save'}
      </button>
    </div>
    <div class="report-text">{data.content}</div>
  </div>

  <div class="section">
    <h2 class="section-title">The numbers behind it</h2>
    <div class="stat-grid">
      <div class="card stat">
        <div class="label">Revenue</div>
        <div class="big">{inr(data.stats.revenue)}</div>
        <div class="sub">
          <span class={data.stats.revenue_change >= 0 ? 'up' : 'down'}>{pct(data.stats.revenue_change)}</span> vs last week
        </div>
      </div>
      <div class="card stat">
        <div class="label">Bills</div>
        <div class="big">{data.stats.orders}</div>
        <div class="sub">{data.stats.units} units</div>
      </div>
      <div class="card stat">
        <div class="label">Top mover</div>
        <div class="big" style="font-size: 16px">{data.stats.top[0]?.name ?? '—'}</div>
        <div class="sub">{data.stats.top[0]?.qty ?? 0} units</div>
      </div>
      <div class="card stat">
        <div class="label">Busy time</div>
        <div class="big" style="font-size: 16px">{data.stats.best_day}</div>
        <div class="sub">Peak {data.stats.busiest_hour}:00</div>
      </div>
    </div>
  </div>
{/if}

{#if jobs}
  <div class="section">
    <h2 class="section-title">Scheduled jobs</h2>
    <div class="card">
      <div class="row" style="margin-bottom: 8px">
        <span class="pill {jobs.mode === 'temporal' ? '' : 'neutral'}">{jobs.mode}</span>
        <span class="muted" style="font-size: 13px">{jobs.detail}</span>
      </div>
      {#each Object.entries(jobs.last_runs) as [name, run]}
        <div class="list-row">
          <div style="flex: 1; font-weight: 600">{name.replace('_', ' ')}</div>
          <div class="muted" style="font-size: 13px">
            {run.last_run ? `last ran ${timeLabel(run.last_run)}` : 'never run'}
          </div>
          <span class="pill {run.last_ok ? '' : 'danger'}">{run.last_ok ? 'ok' : run.error ? 'error' : '—'}</span>
        </div>
      {/each}
      {#each Object.entries(jobs.next_runs) as [name, next]}
        <div class="list-row">
          <div style="flex: 1; font-weight: 600">next: {name.replace('_', ' ')}</div>
          <div class="muted" style="font-size: 13px">{next ? timeLabel(next) : '—'}</div>
        </div>
      {/each}
    </div>
  </div>
{/if}
