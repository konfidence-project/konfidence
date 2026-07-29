const AUTH_CREDENTIAL_HEADERS = ["authorization", "cookie", "x-session-id"] as const;

const credentialHeaders = (request: Request): Headers => {
  const headers = new globalThis.Headers();
  for (const name of AUTH_CREDENTIAL_HEADERS) {
    const value = request.headers.get(name);
    if (value) {
      headers.set(name, value);
    }
  }
  return headers;
};

const hasCredentials = (request: Request): boolean =>
  AUTH_CREDENTIAL_HEADERS.some((name) => request.headers.has(name));

export { credentialHeaders, hasCredentials };
