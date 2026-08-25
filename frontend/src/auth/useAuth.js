import { useState, useEffect, useCallback } from 'react';
import { fetchSession, login as loginRequest, onSessionExpired } from '../api/client.js';
import { readToken, writeToken, clearToken } from './session.js';

/**
 * Whether the viewer may see the app. Starts in a "checking" state so
 * the login form does not flash on a reload while the stored token is
 * being validated.
 */
export function useAuth() {
  const [checking, setChecking] = useState(true);
  const [authenticated, setAuthenticated] = useState(false);
  const [loginConfigured, setLoginConfigured] = useState(true);

  useEffect(() => {
    let cancelled = false;
    fetchSession()
      .then((s) => {
        if (cancelled) return;
        setAuthenticated(!!s.authenticated);
        setLoginConfigured(s.login_configured !== false);
        if (!s.authenticated) clearToken();
      })
      .finally(() => {
        if (!cancelled) setChecking(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Any request may be the one that finds the session gone.
  useEffect(() => onSessionExpired(() => setAuthenticated(false)), []);

  const signIn = useCallback(async (username, password) => {
    const token = await loginRequest(username, password);
    writeToken(token);
    setAuthenticated(true);
  }, []);

  const signOut = useCallback(() => {
    clearToken();
    setAuthenticated(false);
  }, []);

  return { checking, authenticated, loginConfigured, signIn, signOut, hasToken: !!readToken() };
}
