import type { Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

// The browser always talks to same-origin /api; the Node server forwards to
// the Go API so there is one URL in every environment (dev, docker, Render).
export const handle: Handle = async ({ event, resolve }) => {
  if (event.url.pathname.startsWith('/api')) {
    const origin = env.API_ORIGIN || 'http://localhost:8080';
    const target = origin + event.url.pathname + event.url.search;
    const headers = new Headers(event.request.headers);
    headers.delete('host');

    const init: RequestInit & { duplex?: 'half' } = {
      method: event.request.method,
      headers,
      redirect: 'manual'
    };
    if (event.request.method !== 'GET' && event.request.method !== 'HEAD') {
      init.body = event.request.body;
      init.duplex = 'half';
    }

    try {
      const res = await fetch(target, init);
      return new Response(res.body, { status: res.status, headers: res.headers });
    } catch {
      return new Response(JSON.stringify({ error: 'API server is not reachable' }), {
        status: 502,
        headers: { 'content-type': 'application/json' }
      });
    }
  }
  return resolve(event);
};
