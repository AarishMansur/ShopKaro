<script lang="ts">
  let {
    data,
    height = 170,
    accent = '#0e7a5f',
    format = (v: number) => String(Math.round(v))
  }: {
    data: { label: string; value: number }[];
    height?: number;
    accent?: string;
    format?: (v: number) => string;
  } = $props();

  const W = 640;
  const padTop = 18;
  const padBottom = 22;
  const max = $derived(Math.max(1, ...data.map((d) => d.value)));
  const barW = $derived(data.length > 0 ? (W / data.length) * 0.68 : 0);
  const labelEvery = $derived(Math.max(1, Math.ceil(data.length / 9)));
</script>

<div class="chart">
  <svg viewBox="0 0 {W} {height}" role="img" aria-label="bar chart">
    {#each [0.25, 0.5, 0.75, 1] as grid}
      <line
        x1="0"
        x2={W}
        y1={padTop + (height - padTop - padBottom) * (1 - grid)}
        y2={padTop + (height - padTop - padBottom) * (1 - grid)}
        stroke="#e7ebe7"
        stroke-width="1"
      />
    {/each}
    {#each data as d, i}
      {@const h = ((height - padTop - padBottom) * d.value) / max}
      {@const x = (W / data.length) * i + (W / data.length - barW) / 2}
      <rect
        {x}
        y={height - padBottom - h}
        width={barW}
        height={Math.max(d.value > 0 ? 2 : 0, h)}
        rx="3"
        fill={accent}
        opacity={d.value === 0 ? 0.25 : 1}
      >
        <title>{d.label}: {format(d.value)}</title>
      </rect>
      {#if i % labelEvery === 0}
        <text x={x + barW / 2} y={height - 6} text-anchor="middle" font-size="11" fill="#5d6b63">
          {d.label}
        </text>
      {/if}
    {/each}
    <text x="2" y="12" font-size="11" fill="#5d6b63">{format(max)}</text>
  </svg>
</div>
