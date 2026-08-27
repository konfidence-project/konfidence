# Konfidence Mock API

The TypeScript mock server implements `api/openapi.yaml` for dashboard development and tests without Kubernetes.

From the repository root, start it with:

```sh
source ./bin/activate-hermit
pnpm install
pnpm mock-api:dev
```

The server listens on `http://127.0.0.1:8091` by default. Set `KONFIDENCE_MOCK_API_PORT` to use another port. Swagger UI is available at `http://127.0.0.1:8091/docs/`; select **Start mock session** there before running authenticated requests. UI tests can select a response scenario with the `konfidence_mock_scenario` cookie.

Supported scenarios are `admin` (the default), `developer`, and `degraded`. They represent a multi-project administrator, a developer with one sparse project, and an authenticated operator for whom project resource operations fail.

The CLI exchange endpoint returns the session ID `mock-session` for any request matching the OpenAPI schema.

Use `createMockServer` from `src/server.ts` to start the server on an ephemeral port in integration tests. For Playwright, it can also be configured as a web server:

```ts
{
  command: "pnpm --filter konfidence-mock-api start",
  url: "http://127.0.0.1:8091/api/v1/projects",
}
```

Run all type, lint, formatting, generated-code, and contract checks with:

```sh
pnpm mock-api:all
```
