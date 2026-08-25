const STORAGE_KEY = 'gold-tracker.session';

/**
 * The session token lives in localStorage so a reload does not force a
 * fresh sign-in. Storage can throw in a private window or when site
 * data is blocked, so every access is guarded.
 */
export function readToken() {
  try {
    return localStorage.getItem(STORAGE_KEY) || null;
  } catch {
    return null;
  }
}

export function writeToken(token) {
  try {
    localStorage.setItem(STORAGE_KEY, token);
  } catch {
    // A session that lasts only as long as the tab is still usable.
  }
}

export function clearToken() {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing to clear.
  }
}
