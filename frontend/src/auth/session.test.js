import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { readToken, writeToken, clearToken } from './session.js';

describe('session storage', () => {
  beforeEach(() => {
    const store = new Map();
    globalThis.localStorage = {
      getItem: (k) => (store.has(k) ? store.get(k) : null),
      setItem: (k, v) => store.set(k, v),
      removeItem: (k) => store.delete(k),
    };
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('round-trips a token', () => {
    expect(readToken()).toBeNull();
    writeToken('abc');
    expect(readToken()).toBe('abc');
    clearToken();
    expect(readToken()).toBeNull();
  });

  it('survives storage being unavailable', () => {
    globalThis.localStorage = {
      getItem() { throw new Error('blocked'); },
      setItem() { throw new Error('blocked'); },
      removeItem() { throw new Error('blocked'); },
    };
    expect(readToken()).toBeNull();
    expect(() => writeToken('abc')).not.toThrow();
    expect(() => clearToken()).not.toThrow();
  });
});
