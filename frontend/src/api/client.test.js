import { describe, it, expect, vi, afterEach } from 'vitest';
import { apiRequest, ApiError, onSessionExpired } from './client.js';

afterEach(() => {
  vi.restoreAllMocks();
});

describe('apiRequest', () => {
  it('returns parsed JSON on success', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ hello: 'world' }),
    });
    await expect(apiRequest('/api/thing')).resolves.toEqual({ hello: 'world' });
  });

  it('throws the server-supplied message on failure', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: 'bad payload' }),
    });
    await expect(apiRequest('/api/thing')).rejects.toThrow('bad payload');
  });

  it('falls back to a generic message when the error body is not JSON', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => { throw new Error('not json'); },
    });
    await expect(apiRequest('/api/thing')).rejects.toThrow('Request failed (500)');
  });

  it('carries the HTTP status on the thrown error', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      json: async () => ({ error: 'conflict' }),
    });
    // 409 and 429 both need distinguishing by the signal-run hook.
    await expect(apiRequest('/api/thing')).rejects.toMatchObject({
      status: 409,
      name: 'ApiError',
    });
  });

  it('throws ApiError specifically', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      json: async () => ({ error: 'AI is not configured' }),
    });
    await expect(apiRequest('/api/thing')).rejects.toBeInstanceOf(ApiError);
  });

  it('passes options through to fetch, with headers merged', async () => {
    const spy = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) });
    globalThis.fetch = spy;
    await apiRequest('/api/items', { method: 'POST', body: '{}' });
    expect(spy).toHaveBeenCalledWith('/api/items', {
      method: 'POST',
      body: '{}',
      headers: {},
    });
  });

  it('attaches the stored session token', async () => {
    const store = new Map([['gold-tracker.session', 'session-abc']]);
    globalThis.localStorage = {
      getItem: (k) => (store.has(k) ? store.get(k) : null),
      setItem: (k, v) => store.set(k, v),
      removeItem: (k) => store.delete(k),
    };
    const spy = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) });
    globalThis.fetch = spy;

    await apiRequest('/api/portfolio');

    expect(spy.mock.calls[0][1].headers.Authorization).toBe('Bearer session-abc');
  });

  it('clears the session and notifies on 401', async () => {
    const store = new Map([['gold-tracker.session', 'stale']]);
    globalThis.localStorage = {
      getItem: (k) => (store.has(k) ? store.get(k) : null),
      setItem: (k, v) => store.set(k, v),
      removeItem: (k) => store.delete(k),
    };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ error: 'unauthorized' }),
    });

    const seen = vi.fn();
    const stop = onSessionExpired(seen);
    await expect(apiRequest('/api/portfolio')).rejects.toBeInstanceOf(ApiError);
    stop();

    expect(seen).toHaveBeenCalled();
    expect(store.has('gold-tracker.session')).toBe(false);
  });

  it('tolerates a success body that is not JSON', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => { throw new Error('empty body'); },
    });
    await expect(apiRequest('/api/thing')).resolves.toEqual({});
  });
});
