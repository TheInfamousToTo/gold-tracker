import { readToken, clearToken } from '../auth/session.js';

/** An HTTP failure, carrying the status so callers can branch on it. */
export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

const listeners = new Set();

/**
 * Notifies the app that the session is gone, so it can show the login
 * page instead of a screen full of failed requests. Any request can be
 * the one that discovers an expired session, so this is a subscription
 * rather than a return value.
 */
export function onSessionExpired(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/**
 * Calls the API and returns the parsed body, throwing ApiError on any
 * non-2xx response. The backend reports failures as {"error": "..."},
 * so that message is preferred over a generic one where present.
 *
 * The session token is attached here rather than at each call site, so
 * no request can forget it.
 */
export async function apiRequest(url, options = {}) {
  const token = readToken();
  const headers = { ...(options.headers || {}) };
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(url, { ...options, headers });
  const body = await res.json().catch(() => ({}));

  if (res.status === 401) {
    clearToken();
    listeners.forEach((fn) => fn());
    throw new ApiError(body.error || 'Your session has expired. Sign in again.', 401);
  }
  if (!res.ok) {
    throw new ApiError(body.error || `Request failed (${res.status})`, res.status);
  }
  return body;
}

/** Exchanges credentials for a session token. */
export async function login(username, password) {
  const res = await fetch('/api/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(body.error || `Sign-in failed (${res.status})`, res.status);
  }
  return body.token;
}

/** Reports whether the stored token is still good, without a 401. */
export async function fetchSession() {
  const token = readToken();
  const res = await fetch('/api/session', {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  return res.json().catch(() => ({ authenticated: false, login_configured: true }));
}
