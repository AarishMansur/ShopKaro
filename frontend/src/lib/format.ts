// Indian-format rupee amounts: 123456 -> ₹1,23,456
export function inr(v: number, decimals = 0): string {
  const neg = v < 0;
  const fixed = Math.abs(v).toFixed(decimals);
  const [intPart, decPart] = fixed.split('.');
  let out = intPart;
  if (intPart.length > 3) {
    const last3 = intPart.slice(-3);
    let rest = intPart.slice(0, -3);
    const groups: string[] = [];
    while (rest.length > 2) {
      groups.unshift(rest.slice(-2));
      rest = rest.slice(0, -2);
    }
    if (rest) groups.unshift(rest);
    out = groups.join(',') + ',' + last3;
  }
  return (neg ? '-' : '') + '₹' + out + (decPart ? '.' + decPart : '');
}

export function compact(v: number): string {
  if (v >= 10000000) return (v / 10000000).toFixed(1) + 'Cr';
  if (v >= 100000) return (v / 100000).toFixed(1) + 'L';
  if (v >= 1000) return (v / 1000).toFixed(1) + 'k';
  return String(Math.round(v));
}

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

export function dayLabel(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map(Number);
  const dt = new Date(Date.UTC(y, m - 1, d));
  return `${DAYS[dt.getUTCDay()]} ${d}`;
}

export function timeLabel(iso: string): string {
  const dt = new Date(iso);
  let h = dt.getHours();
  const m = String(dt.getMinutes()).padStart(2, '0');
  const ap = h >= 12 ? 'PM' : 'AM';
  h = h % 12 || 12;
  return `${h}:${m} ${ap}`;
}

export function dateLabel(iso: string): string {
  const dt = new Date(iso);
  return `${dt.getDate()} ${MONTHS[dt.getMonth()]}`;
}

export function pct(v: number): string {
  return `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`;
}

// Stable two-letter monogram with a deterministic colour per product.
export function initials(name: string): string {
  const words = name.replace(/[^a-zA-Z0-9 ]/g, ' ').split(/\s+/).filter(Boolean);
  if (words.length === 0) return '?';
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (words[0][0] + words[1][0]).toUpperCase();
}

export function colorFor(seed: string): string {
  let h = 0;
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) % 360;
  return `hsl(${h} 42% 38%)`;
}

export function uuid(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID();
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
