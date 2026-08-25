import { useState } from 'react';
import { Button } from '../components/ui/Button.jsx';
import { Field, inputClass } from '../components/ui/Field.jsx';

/**
 * The gate in front of everything. Nothing about the portfolio renders
 * until this succeeds.
 */
export function LoginPage({ onSignIn, loginConfigured }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await onSignIn(username, password);
    } catch (err) {
      setError(err.message);
      setPassword('');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-ink px-4">
      <div className="w-full max-w-sm">
        <div className="mb-6 text-center">
          <h1 className="font-display text-lg font-bold uppercase tracking-stamp text-chalk">
            Gold Tracker
          </h1>
          <p className="mt-1 text-xs text-muted">Sign in to view your portfolio.</p>
        </div>

        <div className="rounded-lg border border-line bg-ink-raised p-6">
          {!loginConfigured ? (
            <p className="text-sm leading-relaxed text-muted">
              No sign-in is configured on the server. Set{' '}
              <code className="font-mono text-chalk">GOLD_AUTH_USERNAME</code> and{' '}
              <code className="font-mono text-chalk">GOLD_AUTH_PASSWORD</code> in the API's
              environment, then restart it.
            </p>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <Field label="Username" htmlFor="username">
                <input
                  id="username"
                  name="username"
                  autoComplete="username"
                  autoFocus
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className={inputClass}
                />
              </Field>

              <Field label="Password" htmlFor="password">
                <input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className={inputClass}
                />
              </Field>

              {error && (
                <p role="alert" className="text-sm text-bad-bright">
                  {error}
                </p>
              )}

              <Button type="submit" size="lg" loading={busy} loadingLabel="Signing in">
                Sign in
              </Button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}
