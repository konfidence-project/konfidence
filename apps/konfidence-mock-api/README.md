# Konfidence Mock API

The TypeScript mock server implements `api/openapi.yaml` for dashboard development and tests without Kubernetes.

From the repository root, start it with:

```sh
source ./bin/activate-hermit
pnpm install
pnpm mock-api:dev
```

The server listens on `http://127.0.0.1:8091` by default. Set `KONFIDENCE_MOCK_API_PORT` to use another port and `KONFIDENCE_MOCK_SCENARIO` to select a default response scenario. Individual tests can override the scenario with the `konfidence_mock_scenario` cookie. Login redirects may target localhost by default; add comma-separated origins with `KONFIDENCE_MOCK_ALLOWED_RETURN_ORIGINS` when developing against another host.

Supported scenarios are `default`, `unauthenticated`, `forbidden`, `internal-error`, `no-projects`, and `one-project`.

The CLI exchange endpoint accepts `mock-code` with verifier `mock-verifier` and returns the session ID `mock-session`.

Use `createMockServer` from `src/server.ts` to start the server on an ephemeral port in integration tests. For Playwright, it can also be configured as a web server:

```ts
{
  command: "pnpm --filter konfidence-mock-api start",
  url: "http://127.0.0.1:8091/health",
}
```

Run all type, lint, formatting, generated-code, and contract checks with:

```sh
pnpm mock-api:all
```
