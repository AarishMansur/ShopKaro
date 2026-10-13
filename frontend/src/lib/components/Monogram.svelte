<script lang="ts">
  import { colorFor, initials } from '$lib/format';

  let { name, small = false }: { name: string; small?: boolean } = $props();
</script>

<span class={small ? 'mono-sm' : 'mono'} style="background: {colorFor(name)}" aria-hidden="true">
  {initials(name)}
</span>
