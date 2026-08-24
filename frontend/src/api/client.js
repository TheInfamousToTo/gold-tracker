/** An HTTP failure, carrying the status so callers can branch on it. */
export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

/**
 * Calls the API and returns the parsed body, throwing ApiError on any
 * non-2xx response. The backend reports failures as {"error": "..."},
 * so that message is preferred over a generic one where present.
 */
export async function apiRequest(url, options) {
  const res = await fetch(url, options);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(body.error || `Request failed (${res.status})`, res.status);
  }
  return body;
}
