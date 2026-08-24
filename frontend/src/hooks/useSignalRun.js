import { useState, useCallback, useRef, useEffect } from 'react';
import { apiRequest, ApiError } from '../api/client.js';

const POLL_INTERVAL_MS = 2000;

/**
 * Drives on-demand signal generation. The endpoint returns immediately
 * and the run continues server-side, so this polls the status endpoint
 * until it settles and then calls onGenerated so the caller can refetch.
 */
export function useSignalRun(onGenerated) {
  const [status, setStatus] = useState(null);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState(null);
  const pollRef = useRef(null);

  // Kept in a ref so changing the callback doesn't restart polling.
  const onGeneratedRef = useRef(onGenerated);
  useEffect(() => {
    onGeneratedRef.current = onGenerated;
  }, [onGenerated]);

  const stopPolling = useCallback(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const fetchStatus = useCallback(async () => {
    try {
      const next = await apiRequest('/api/signals/status');
      setStatus(next);
      return next;
    } catch {
      return null;
    }
  }, []);

  const startPolling = useCallback(() => {
    stopPolling();
    pollRef.current = setInterval(async () => {
      const next = await fetchStatus();
      if (next && !next.running) {
        stopPolling();
        setGenerating(false);
        onGeneratedRef.current?.();
      }
    }, POLL_INTERVAL_MS);
  }, [fetchStatus, stopPolling]);

  const generate = useCallback(async () => {
    setError(null);
    try {
      await apiRequest('/api/signals/generate', { method: 'POST' });
      setGenerating(true);
      startPolling();
    } catch (err) {
      // A run already in flight isn't a failure — attach to it. Any
      // other error (429 cooling down, 503 not configured) is shown.
      if (err instanceof ApiError && err.status === 409) {
        setGenerating(true);
        startPolling();
        return;
      }
      setError(err.message);
    }
  }, [startPolling]);

  useEffect(() => {
    // A run started elsewhere, or before a reload, should still be
    // reflected here.
    fetchStatus().then((initial) => {
      if (initial?.running) {
        setGenerating(true);
        startPolling();
      }
    });
    return stopPolling;
  }, [fetchStatus, startPolling, stopPolling]);

  return { status, generating, error, generate };
}
