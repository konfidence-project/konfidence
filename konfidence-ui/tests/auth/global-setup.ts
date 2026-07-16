import { type ChildProcess, spawn } from "node:child_process";
import {
  GenericContainer,
  Network,
  type StartedNetwork,
  type StartedTestContainer,
  Wait,
} from "testcontainers";
import { setTimeout as delay } from "node:timers/promises";
import { resolve } from "node:path";
import { rm } from "node:fs/promises";

const API_PORT = 18_090;
const API_CONTAINER_PORT = 8090;
const DEX_PORT = 15_556;
const DEX_CONTAINER_PORT = 5556;
const UI_CONTAINER_PORT = 3000;
const UI_PORT = 4173;
const STARTUP_TIMEOUT_MS = 120_000;
const POLL_INTERVAL_MS = 250;
const API_BINARY = `/tmp/konfidence-api-e2e-${process.pid}`;
const UI_IMAGE = "konfidence-ui-e2e";

const repositoryRoot = resolve(import.meta.dirname, "../../..");

const dexConfig = `issuer: http://localhost:${DEX_PORT}/dex
storage:
  type: memory
web:
  http: 0.0.0.0:5556
oauth2:
  skipApprovalScreen: true
staticClients:
  - id: kden-e2e
    name: Konfidence E2E
    public: true
    redirectURIs:
      - http://127.0.0.1:${UI_PORT}/api/auth/callback
connectors:
  - type: mockCallback
    id: mock
    name: Mock
enablePasswordDB: true
staticPasswords:
  - email: alice@example.com
    hash: "$2a$10$dz1JnjvIexy8vbdE/cQBX.SdhSuK0v67XthCK3cPLNeUx1HGaC86m"
    username: alice
    userID: alice
    groups:
      - admins
  - email: bob@example.com
    hash: "$2a$10$dz1JnjvIexy8vbdE/cQBX.SdhSuK0v67XthCK3cPLNeUx1HGaC86m"
    username: bob
    userID: bob
    groups:
      - no-access
`;

const startProcess = (
  command: string,
  args: string[],
  env = process.env,
  cwd = repositoryRoot,
): ChildProcess =>
  spawn(command, args, {
    cwd,
    env,
    stdio: "inherit",
  });

const runProcess = async (command: string, args: string[], env = process.env): Promise<void> => {
  const process = startProcess(command, args, env);
  await new Promise<void>((resolveExit, rejectExit) => {
    process.once("exit", (code) => {
      if (code === 0) {
        resolveExit();
        return;
      }
      rejectExit(new Error(`${command} exited with code ${code}`));
    });
  });
};

const waitForUrl = async (url: string): Promise<void> => {
  const deadline = Date.now() + STARTUP_TIMEOUT_MS;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
    } catch {
      // The process may still be binding its port.
    }
    await delay(POLL_INTERVAL_MS);
  }
  throw new Error(`Timed out waiting for ${url}`);
};

export default async function globalSetup(): Promise<() => Promise<void>> {
  let network: StartedNetwork | undefined = undefined;
  let dex: StartedTestContainer | undefined = undefined;
  let api: StartedTestContainer | undefined = undefined;
  let ui: StartedTestContainer | undefined = undefined;

  try {
    const startedNetwork = await new Network().start();
    network = startedNetwork;
    dex = await new GenericContainer("ghcr.io/dexidp/dex:v2.45.1")
      .withCopyContentToContainer([{ content: dexConfig, target: "/etc/dex/config.yaml" }])
      .withCommand(["dex", "serve", "/etc/dex/config.yaml"])
      .withExposedPorts({ container: DEX_CONTAINER_PORT, host: DEX_PORT })
      .withNetwork(startedNetwork)
      .withNetworkAliases("dex")
      .withWaitStrategy(Wait.forHttp("/dex/.well-known/openid-configuration", DEX_CONTAINER_PORT))
      .start();

    await runProcess("go", ["build", "-o", API_BINARY, "./cmd/api/main.go"], {
      ...process.env,
      CGO_ENABLED: "0",
      GOARCH: process.arch.replace("x64", "amd64"),
      GOOS: "linux",
    });
    api = await new GenericContainer("debian:bookworm-slim")
      .withCopyFilesToContainer([{ mode: 0o755, source: API_BINARY, target: "/api" }])
      .withCommand(["/api"])
      .withEnvironment({
        API_ADDR: `0.0.0.0:${API_CONTAINER_PORT}`,
        API_AUTH_AUTHORIZE_URL: `http://localhost:${DEX_PORT}/dex/auth`,
        API_AUTH_CLIENT_ID: "kden-e2e",
        API_AUTH_REDIRECT_URI: `http://127.0.0.1:${UI_PORT}/api/auth/callback`,
        API_AUTH_TOKEN_URL: `http://dex:${DEX_CONTAINER_PORT}/dex/token`,
        API_AUTH_USERINFO_URL: `http://dex:${DEX_CONTAINER_PORT}/dex/userinfo`,
      })
      .withExposedPorts({ container: API_CONTAINER_PORT, host: API_PORT })
      .withNetwork(startedNetwork)
      .withNetworkAliases("api")
      .withWaitStrategy(Wait.forHttp("/healthz", API_CONTAINER_PORT))
      .start();
    await waitForUrl(`http://127.0.0.1:${API_PORT}/healthz`);

    await runProcess("docker", ["build", "--file", "Dockerfile.ui", "--tag", UI_IMAGE, "."]);
    ui = await new GenericContainer(UI_IMAGE)
      .withEnvironment({
        KONFIDENCE_API_URL: `http://api:${API_CONTAINER_PORT}`,
        ORIGIN: `http://127.0.0.1:${UI_PORT}`,
      })
      .withExposedPorts({ container: UI_CONTAINER_PORT, host: UI_PORT })
      .withNetwork(startedNetwork)
      .withWaitStrategy(Wait.forHttp("/robots.txt", UI_CONTAINER_PORT))
      .start();
  } catch (error) {
    await ui?.stop();
    await api?.stop();
    await dex?.stop();
    await network?.stop();
    await rm(API_BINARY, { force: true });
    throw error;
  }

  return async () => {
    await ui?.stop();
    await api?.stop();
    await dex?.stop();
    await network?.stop();
    await rm(API_BINARY, { force: true });
  };
}
