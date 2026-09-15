# Konfidence Mock API

The TypeScript mock server implements `api/openapi.yaml` for dashboard development and tests without Kubernetes.

From the repository root, start it with:

```sh
source ./bin/activate-hermit
pnpm install
pnpm mock-api:dev
```

The server listens on `http://127.0.0.1:8091` by default. Set `KONFIDENCE_MOCK_API_PORT` to use another port. Set `KONFIDENCE_MOCK_DELAY=1` to enable a random 0.5-2 second delay on API requests; responses are immediate when it is unset. Swagger UI is available at `http://127.0.0.1:8091/docs/`; follow **Start a mock session** there before running authenticated requests. UI tests can select a response scenario with the `konfidence_mock_scenario` cookie.

Supported scenarios are `admin` (the default), `developer`, and `degraded`. They represent a multi-project administrator, a developer with one sparse project, and an authenticated operator for whom project resource operations fail.

The CLI exchange endpoint sets the `kden-session` cookie for any request body matching the OpenAPI schema. Anything the spec rejects comes back as an `ErrorResponse`, as does an unknown route.

## Manual verification

From the repository root, start the dashboard and mock server together:

```sh
pnpm ui:dev:mock
```

On the sign-in page, use the mock sign-in flow: the button redirects through the mock login and callback endpoints, which set the `kden-session` cookie, so no real identity provider is needed. The active response scenario is controlled by the `konfidence_mock_scenario` cookie as described above.

In the default `admin` scenario, choose **Payments Platform** to explore a fully populated landscape overview and follow its stage links, including stages whose active and target versions differ. Choose **Identity Service** to see the empty project state. The `developer` and `degraded` scenarios exercise the sparse and unavailable-resource behaviors described above.

Stage statuses in the overview are independent API status values, not an ordered deployment sequence. A stage can report `Ready`, `Failed`, `PendingDeployment`, `DeployingVector`, `MigratingVector`, or `ActivatingVector` independently of the others; do not read across stages as a progression.

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
