import type {
  DashboardSummary,
  DayPoint,
  Features,
  Forecast,
  HourPoint,
  JobStatus,
  ParseResult,
  Product,
  ProductSuggestion,
  RankedProduct,
  Report,
  Sale,
  SummaryResponse
} from './types';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      /* keep default message */
    }
    throw new Error(message);
  }
  return res.json() as Promise<T>;
}

const json = (body: unknown): RequestInit => ({
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify(body)
});

export const api = {
  features: () => request<Features>('/api/features'),

  products: () => request<{ products: Product[] }>('/api/products').then((r) => r.products),
  createProduct: (input: Partial<Product>) => request<Product>('/api/products', json(input)),
  updateProduct: (id: number, input: Partial<Product>) =>
    request<Product>(`/api/products/${id}`, { ...json(input), method: 'PUT' }),
  deleteProduct: (id: number) => request<{ result: string }>(`/api/products/${id}`, { method: 'DELETE' }),
  fromPhoto: (file: File) => {
    const fd = new FormData();
    fd.append('photo', file);
    return request<ProductSuggestion>('/api/products/from-photo', { method: 'POST', body: fd });
  },

  recentSales: (limit = 25) =>
    request<{ sales: Sale[] }>(`/api/sales?limit=${limit}`).then((r) => r.sales),
  batchSales: (sales: unknown[]) => request<{ accepted: number }>('/api/sales/batch', json({ sales })),
  updateSaleQty: (id: number, qty: number) =>
    request<Sale>(`/api/sales/${id}`, { ...json({ qty }), method: 'PATCH' }),
  deleteSale: (id: number) => request<{ deleted: boolean }>(`/api/sales/${id}`, { method: 'DELETE' }),

  summary: (days = 7) => request<DashboardSummary>(`/api/dashboard/summary?days=${days}`),
  bestSellers: (days: number, limit = 10) =>
    request<{ products: RankedProduct[] }>(`/api/dashboard/best-sellers?days=${days}&limit=${limit}`).then(
      (r) => r.products
    ),
  slowMovers: (days: number, limit = 10) =>
    request<{ products: RankedProduct[] }>(`/api/dashboard/slow-movers?days=${days}&limit=${limit}`).then(
      (r) => r.products
    ),
  byDay: (days = 28) => request<{ days: DayPoint[] }>(`/api/dashboard/by-day?days=${days}`).then((r) => r.days),
  byHour: (days = 28) => request<{ hours: HourPoint[] }>(`/api/dashboard/by-hour?days=${days}`).then((r) => r.hours),

  forecast: () => request<Forecast>('/api/reorder/forecast'),

  parseText: (text: string) => request<ParseResult>('/api/ai/parse', json({ text })),
  parseVoice: (blob: Blob) => {
    const fd = new FormData();
    fd.append('audio', blob, 'voice.webm');
    return request<ParseResult>('/api/ai/voice', { method: 'POST', body: fd });
  },
  weeklySummary: (lang: string) => request<SummaryResponse>(`/api/ai/summary?lang=${lang}`),

  jobs: () => request<JobStatus>('/api/jobs/status'),
  runJob: (kind: string, lang?: string) => request<Report>('/api/jobs/run', json({ kind, lang })),
  latestReport: (kind: string) =>
    request<{ report: Report | null }>(`/api/reports/latest?kind=${kind}`).then((r) => r.report)
};
