export interface Product {
  id: number;
  name: string;
  category: string;
  price: number;
  mrp: number;
  unit: string;
  aliases: string[];
  photo_url: string | null;
  emoji: string;
  sort_order: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Sale {
  id: number;
  client_id: string;
  product_id: number;
  qty: number;
  unit_price: number;
  total: number;
  source: string;
  sold_at: string;
  product_name?: string;
  emoji?: string;
}

export interface PendingSale {
  client_id: string;
  product_id: number;
  qty: number;
  unit_price: number;
  sold_at: string;
  source: string;
}

export interface DashboardSummary {
  days: number;
  revenue: number;
  prev_revenue: number;
  revenue_change: number;
  orders: number;
  prev_orders: number;
  units: number;
  avg_basket: number;
  top_product: string;
  best_day: string;
  busiest_hour: number;
  active_products: number;
  new_products: number;
  category_revenue: RankedProduct[];
}

export interface RankedProduct {
  product_id: number;
  name: string;
  emoji: string;
  category: string;
  qty: number;
  revenue: number;
  orders: number;
}

export interface DayPoint {
  date: string;
  revenue: number;
  orders: number;
  qty: number;
}

export interface HourPoint {
  hour: number;
  revenue: number;
  orders: number;
}

export interface ReorderRow {
  product_id: number;
  name: string;
  emoji: string;
  category: string;
  unit: string;
  price: number;
  last_week_qty: number;
  avg_4week_qty: number;
  forecast_qty: number;
  suggested_qty: number;
  confidence: string;
  had_history: boolean;
}

export interface Forecast {
  generated_at: string;
  method: string;
  weeks_ahead: number;
  rows: ReorderRow[];
  note: string;
}

export interface ParsedEntry {
  product_id: number;
  name: string;
  qty: number;
  unit_price: number;
  matched_on?: string;
}

export interface ParseResult {
  engine: string;
  entries: ParsedEntry[];
  unmatched: string[];
  transcript?: string;
}

export interface WeeklyStats {
  days: number;
  revenue: number;
  prev_revenue: number;
  revenue_change: number;
  orders: number;
  units: number;
  avg_basket: number;
  top: RankedProduct[];
  slow: RankedProduct[];
  best_day: string;
  busiest_hour: number;
  new_products: number;
  active_products: number;
}

export interface SummaryResponse {
  lang: string;
  engine: string;
  content: string;
  stats: WeeklyStats;
}

export interface Features {
  gemma: boolean;
  gemma_config: boolean;
  gemma_model: string;
  gemma_error: string;
  whisper: boolean;
  tabpfn: boolean;
  temporal: boolean;
  scheduler: string;
  timescaledb: boolean;
  shop: { name: string; tz: string };
}

export interface JobRun {
  last_run: string | null;
  last_ok: boolean;
  error?: string;
}

export interface JobStatus {
  mode: string;
  enabled: boolean;
  detail: string;
  last_runs: Record<string, JobRun>;
  next_runs: Record<string, string>;
}

export interface Report {
  id: number;
  kind: string;
  period_start: string;
  period_end: string;
  lang: string;
  content: string;
  meta: string;
  created_at: string;
}

export interface ProductSuggestion {
  name: string;
  category: string;
  price: number;
  mrp: number;
  unit: string;
  aliases: string[];
}
