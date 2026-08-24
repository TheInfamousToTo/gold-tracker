import { describe, it, expect, vi, afterEach } from 'vitest';
import { apiRequest, ApiError } from './client.js';

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

  it('passes options through to fetch', async () => {
    const spy = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) });
    globalThis.fetch = spy;
    const options = { method: 'POST', body: '{}' };
    await apiRequest('/api/items', options);
    expect(spy).toHaveBeenCalledWith('/api/items', options);
  });

  it('tolerates a success body that is not JSON', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => { throw new Error('empty body'); },
    });
    await expect(apiRequest('/api/thing')).resolves.toEqual({});
  });
});
