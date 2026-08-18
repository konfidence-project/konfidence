const AUTH_CREDENTIAL_HEADERS = ["authorization", "cookie", "x-session-id"] as const;

const hasCredentials = (request: Request): boolean =>
  AUTH_CREDENTIAL_HEADERS.some((name) => request.headers.has(name));

export { hasCredentials };
