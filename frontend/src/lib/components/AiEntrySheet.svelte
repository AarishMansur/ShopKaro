<script lang="ts">
  import { api } from '$lib/api';
  import { inr } from '$lib/format';
  import { app, showToast } from '$lib/stores.svelte';
  import { recordSale } from '$lib/sync';
  import type { ParseResult } from '$lib/types';
  import Sheet from './Sheet.svelte';

  let { onclose }: { onclose: () => void } = $props();

  let text = $state('');
  let result = $state<ParseResult | null>(null);
  let busy = $state(false);
  let listening = $state(false);
  let error = $state('');
  let mediaRec: MediaRecorder | null = null;
  let audioStream: MediaStream | null = null;

  const gemmaOn = $derived(app.features?.gemma ?? false);

  async function parse() {
    if (!text.trim()) return;
    busy = true;
    error = '';
    try {
      result = await api.parseText(text.trim());
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
    }
  }

  async function recordAll() {
    if (!result) return;
    for (const entry of result.entries) {
      await recordSale({
        product_id: entry.product_id,
        qty: entry.qty,
        unit_price: entry.unit_price,
        source: 'ai'
      });
    }
    const n = result.entries.length;
    showToast(`Recorded ${n} item${n === 1 ? '' : 's'}`);
    onclose();
  }

  async function toggleMic() {
    if (listening && mediaRec) {
      mediaRec.stop();
      return;
    }
    error = '';
    if (app.features?.whisper) {
      await recordWithWhisper();
      return;
    }
    startBrowserRecognition();
  }

  async function recordWithWhisper() {
    try {
      audioStream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      error = 'Microphone permission is needed';
      return;
    }
    const mime = MediaRecorder.isTypeSupported('audio/webm') ? 'audio/webm' : '';
    mediaRec = new MediaRecorder(audioStream, mime ? { mimeType: mime } : undefined);
    const chunks: Blob[] = [];
    mediaRec.ondataavailable = (e) => {
      if (e.data.size > 0) chunks.push(e.data);
    };
    mediaRec.onstop = async () => {
      audioStream?.getTracks().forEach((t) => t.stop());
      audioStream = null;
      listening = false;
      busy = true;
      try {
        result = await api.parseVoice(new Blob(chunks, { type: mediaRec?.mimeType || 'audio/webm' }));
      } catch (e) {
        error = (e as Error).message;
      } finally {
        busy = false;
        mediaRec = null;
      }
    };
    mediaRec.start();
    listening = true;
    showToast('Listening — tap the mic again to stop');
  }

  function startBrowserRecognition() {
    const w = window as unknown as {
      SpeechRecognition?: new () => any;
      webkitSpeechRecognition?: new () => any;
    };
    const SR = w.SpeechRecognition || w.webkitSpeechRecognition;
    if (!SR) {
      error =
        'Voice needs the open-source Whisper server. Start it with: docker compose --profile ai up — or type instead.';
      return;
    }
    const recog = new SR();
    recog.lang = 'hi-IN';
    recog.interimResults = false;
    recog.maxAlternatives = 1;
    recog.onresult = (ev: any) => {
      text = ev.results[0][0].transcript as string;
      parse();
    };
    recog.onerror = () => {
      listening = false;
      error = 'Could not hear that — try again';
    };
    recog.onend = () => {
      listening = false;
    };
    recog.start();
    listening = true;
    showToast('Boliye… (browser voice)');
  }
</script>

<Sheet title="Voice or text sale" {onclose}>
  <div class="row" style="margin-bottom: 10px">
    <span class="pill {gemmaOn ? '' : 'neutral'}">{gemmaOn ? 'Gemma' : 'offline parser'}</span>
    <span class="muted" style="font-size: 13px">Try “2 Maggi, 1 doodh”</span>
  </div>

  <div class="row" style="gap: 8px; align-items: stretch">
    <button
      class="btn {listening ? 'btn-danger' : ''}"
      style="flex: 0 0 auto; min-width: 58px"
      onclick={toggleMic}
      aria-label="Voice entry"
    >
      <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="9" y="3" width="6" height="11" rx="3" />
        <path d="M5 11a7 7 0 0 0 14 0M12 18v3" />
      </svg>
    </button>
    <input
      class="input"
      bind:value={text}
      placeholder="2 Maggi, 1 doodh"
      onkeydown={(e) => e.key === 'Enter' && parse()}
    />
    <button class="btn btn-primary" onclick={parse} disabled={busy || !text.trim()}>
      {#if busy}<span class="spinner"></span>{/if}
      Parse
    </button>
  </div>

  {#if error}
    <p class="down" style="margin: 10px 0 0">{error}</p>
  {/if}

  {#if result}
    <div class="card" style="margin-top: 12px">
      {#if result.transcript}
        <div class="muted" style="font-size: 13px; margin-bottom: 6px">“{result.transcript}”</div>
      {/if}
      {#if result.entries.length === 0}
        <div class="empty">Could not match any product. Try shorter names.</div>
      {:else}
        {#each result.entries as entry (entry.product_id)}
          <div class="list-row">
            <div style="flex:1; font-weight: 600">{entry.name}</div>
            <div class="pill">×{entry.qty}</div>
            <div style="font-weight: 700">{inr(entry.qty * entry.unit_price)}</div>
          </div>
        {/each}
      {/if}
      {#if result.unmatched?.length}
        <div class="muted" style="font-size: 13px; margin-top: 8px">
          Not matched: {result.unmatched.join(', ')}
        </div>
      {/if}
    </div>
    {#if result.entries.length > 0}
      <button class="btn btn-primary" style="width: 100%; margin-top: 12px" onclick={recordAll}>
        Record {result.entries.length} sale{result.entries.length === 1 ? '' : 's'}
      </button>
    {/if}
  {/if}
</Sheet>
