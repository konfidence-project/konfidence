/**
 * OIDC-flavoured session user. Kept intentionally close to standard ID-token
 * claims so that swapping the stubbed login for a real IdP (e.g. SAP IAS,
 * Azure AD, Google) does not require any changes outside the login endpoint.
 */
interface SessionUser {
  /** Optional email address, for future IdP integration. */
  email?: string;
  /** Display name — the silly generated one for the stub. */
  name: string;
  /** Optional avatar URL, for future IdP integration. */
  picture?: string;
  /** Stable subject identifier — will map to the OIDC `sub` claim later. */
  sub: string;
}

interface StoredSession {
  createdAt: number;
  user: SessionUser;
}

const MS_PER_SECOND = 1000;
const SECONDS_PER_MINUTE = 60;
const MINUTES_PER_HOUR = 60;
const SESSION_TTL_HOURS = 8;

/** 8 hours */
const SESSION_TTL_MS = SESSION_TTL_HOURS * MINUTES_PER_HOUR * SECONDS_PER_MINUTE * MS_PER_SECOND;

/** Cookie name used to correlate a browser with a server session. */
const SESSION_COOKIE_NAME = "konfidence.sid";

/**
 * In-memory session store. Dies on server restart, which is acceptable for the
 * MVP. Replace with Redis / DB behind the same interface without touching the
 * routes.
 */
const sessions = new Map<string, StoredSession>();

const isExpired = (session: StoredSession): boolean =>
  Date.now() - session.createdAt > SESSION_TTL_MS;

const createSession = (user: SessionUser): string => {
  // Web Crypto API is available in Node 19+ and matches what a real IdP-backed
  // Implementation would use client- or edge-side.
  const id = crypto.randomUUID();
  sessions.set(id, { createdAt: Date.now(), user });
  return id;
};

const getSession = (id: string | undefined): SessionUser | undefined => {
  if (!id) {
    return undefined;
  }

  const session = sessions.get(id);
  if (!session) {
    return undefined;
  }

  if (isExpired(session)) {
    sessions.delete(id);
    return undefined;
  }

  return session.user;
};

const destroySession = (id: string | undefined): void => {
  if (!id) {
    return;
  }
  sessions.delete(id);
};

export { createSession, destroySession, getSession, SESSION_COOKIE_NAME, SESSION_TTL_MS };
export type { SessionUser };
