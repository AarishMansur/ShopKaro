import type { PendingSale, Product } from './types';

// Tiny promise wrapper around IndexedDB so tap-to-sell keeps working with the
// network down. The queue is drained by sync.ts once the API is reachable.
const DB_NAME = 'shopkaro';
const DB_VERSION = 1;

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains('queue')) db.createObjectStore('queue', { keyPath: 'client_id' });
      if (!db.objectStoreNames.contains('kv')) db.createObjectStore('kv');
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function withStore<T>(mode: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest): Promise<T> {
  const db = await openDB();
  return new Promise<T>((resolve, reject) => {
    const tx = db.transaction('queue', mode);
    const req = fn(tx.objectStore('queue'));
    req.onsuccess = () => resolve(req.result as T);
    req.onerror = () => reject(req.error);
    tx.oncomplete = () => db.close();
  });
}

export async function queueSale(sale: PendingSale): Promise<void> {
  await withStore('readwrite', (s) => s.put(sale));
}

export async function listQueued(): Promise<PendingSale[]> {
  return (await withStore<PendingSale[]>('readonly', (s) => s.getAll())) ?? [];
}

export async function removeQueued(clientIds: string[]): Promise<void> {
  const db = await openDB();
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction('queue', 'readwrite');
    const store = tx.objectStore('queue');
    for (const id of clientIds) store.delete(id);
    tx.oncomplete = () => {
      db.close();
      resolve();
    };
    tx.onerror = () => reject(tx.error);
  });
}

export async function queueCount(): Promise<number> {
  return (await withStore<number>('readonly', (s) => s.count())) ?? 0;
}

export async function cacheProducts(products: Product[]): Promise<void> {
  try {
    localStorage.setItem('shopkaro:products', JSON.stringify(products));
  } catch {
    /* storage full or blocked — the queue itself still works */
  }
}

export async function loadCachedProducts(): Promise<Product[]> {
  try {
    const raw = localStorage.getItem('shopkaro:products');
    return raw ? (JSON.parse(raw) as Product[]) : [];
  } catch {
    return [];
  }
}
