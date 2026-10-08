import type { FastifyReply, FastifyRequest } from "fastify";
import { scenarios, type ProjectFixture, type ScenarioFixture } from "./fixtures.js";
import type { operations } from "@konfidence/api-client/schema";

const SESSION_COOKIE = "kden-session";
const SCENARIO_COOKIE = "konfidence_mock_scenario";
const MOCK_SESSION = "mock-session";
const MOCK_CODE = "mock-code";
const HTTP_NO_CONTENT = 204;
const COOKIE_OPTIONS = { httpOnly: true, path: "/", sameSite: "lax" } as const;
const MOCK_ORIGIN = "https://mock-api.invalid";
const UI_ORIGINS = new Set([
  "http://127.0.0.1:4173",
  "http://127.0.0.1:5173",
  "http://localhost:4173",
  "http://localhost:5173",
]);

type Query<Name extends keyof operations> = NonNullable<operations[Name]["parameters"]["query"]>;
type MockHandler = (
  request: FastifyRequest,
  reply: FastifyReply,
) => FastifyReply | Promise<FastifyReply>;

const httpError = (status: number, message: string): Error =>
  Object.assign(new Error(message), { statusCode: status });

const MOCK_DELAY_ENABLED = process.env.KONFIDENCE_MOCK_DELAY === "1";
const MIN_MOCK_DELAY_MS = 500;
const MAX_MOCK_DELAY_MS = 2000;

const wait = (ms: number): Promise<void> =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const delay = (): Promise<void> => {
  if (!MOCK_DELAY_ENABLED) {
    return Promise.resolve();
  }
  const duration = MIN_MOCK_DELAY_MS + Math.random() * (MAX_MOCK_DELAY_MS - MIN_MOCK_DELAY_MS);
  return wait(duration);
};

const validatedReturnUrl = (value: string | undefined, whenInvalid: string): string => {
  const parsed = value ? (URL.parse(value, MOCK_ORIGIN) ?? undefined) : undefined;
  const safeRelativePath = value?.startsWith("/") === true && parsed?.origin === MOCK_ORIGIN;
  const allowedUiUrl =
    parsed !== undefined &&
    (parsed.protocol === "http:" || parsed.protocol === "https:") &&
    parsed.username === "" &&
    parsed.password === "" &&
    UI_ORIGINS.has(parsed.origin);

  if (!safeRelativePath && !allowedUiUrl) {
    throw httpError(400, whenInvalid);
  }
  return value!;
};

const inLandscape = <Item extends { landscapeId?: string }>(
  items: Item[],
  landscapeId?: string,
): Item[] => items.filter((item) => landscapeId === undefined || item.landscapeId === landscapeId);

const requireLandscape = (project: ProjectFixture, landscapeId?: string): void => {
  if (landscapeId !== undefined && !project.landscapes.some(({ id }) => id === landscapeId)) {
    throw httpError(404, `Landscape ${landscapeId} not found`);
  }
};

const scenarioFor = (request: FastifyRequest): ScenarioFixture =>
  scenarios[request.cookies[SCENARIO_COOKIE] as keyof typeof scenarios] ?? scenarios.admin;

const projectFor = (request: FastifyRequest): ProjectFixture => {
  const scenario = scenarioFor(request);
  if (scenario.resourcesUnavailable) {
    throw httpError(500, "Mock API unavailable");
  }
  const { projectId } = request.params as { projectId: string };
  const found = scenario.projects.find(({ project }) => project.id === projectId);
  if (!found) {
    throw httpError(403, "Access denied");
  }
  return found;
};

// Looks up a single promotion across all of the project's configs.
const promotionFor = (request: FastifyRequest) => {
  const { vectorPromotionId } = request.params as { vectorPromotionId: string };
  const promotion = projectFor(request)
    .vectorPromotionConfigs.flatMap((config) => config.promotions)
    .find(({ id }) => id === vectorPromotionId);
  if (!promotion) {
    throw httpError(404, "VectorPromotion not found");
  }
  return promotion;
};

const operationHandlers = {
  approveVectorPromotionV1: (request, reply) => {
    promotionFor(request);
    return reply.code(HTTP_NO_CONTENT).send();
  },
  authCallbackV1: (request, reply) => {
    const {
      error,
      error_description: description,
      state,
    } = request.query as Query<"authCallbackV1">;
    if (error) {
      throw httpError(401, description ?? error);
    }
    return reply
      .setCookie(SESSION_COOKIE, MOCK_SESSION, COOKIE_OPTIONS)
      .redirect(validatedReturnUrl(state, "Invalid authentication state"));
  },
  getArtifactDeploymentV1: (request, reply) => {
    const { artifactDeploymentId } = request.params as { artifactDeploymentId: string };
    const deployment = projectFor(request).artifactDeployments.find(
      ({ id }) => id === artifactDeploymentId,
    );
    if (!deployment) {
      throw httpError(404, "ArtifactDeployment not found");
    }
    return reply.send(deployment);
  },
  getLandscapeV1: (request, reply) => {
    const { landscapeId } = request.params as { landscapeId: string };
    const landscape = projectFor(request).landscapes.find(({ id }) => id === landscapeId);
    if (!landscape) {
      throw httpError(404, "Landscape not found");
    }
    return reply.send(landscape);
  },
  getIdentityV1: (request, reply) => {
    const { projects, user } = scenarioFor(request);
    const projectRoles = Object.fromEntries(
      projects.map(({ project, roles }) => [project.id, roles]),
    );
    return reply.send({ ...user, projectRoles });
  },
  getStageV1: (request, reply) => {
    const { stageId } = request.params as { stageId: string };
    const stage = projectFor(request).stages.find(({ id }) => id === stageId);
    if (!stage) {
      throw httpError(404, "Stage not found");
    }
    return reply.send(stage);
  },
  getVectorDeploymentV1: (request, reply) => {
    const { vectorDeploymentId } = request.params as { vectorDeploymentId: string };
    const deployment = projectFor(request).vectorDeployments.find(
      ({ id }) => id === vectorDeploymentId,
    );
    if (!deployment) {
      throw httpError(404, "VectorDeployment not found");
    }
    return reply.send(deployment);
  },
  getVectorPromotionConfigV1: (request, reply) => {
    const { vectorPromotionConfigId } = request.params as { vectorPromotionConfigId: string };
    const config = projectFor(request).vectorPromotionConfigs.find(
      ({ id }) => id === vectorPromotionConfigId,
    );
    if (!config) {
      throw httpError(404, "VectorPromotionConfig not found");
    }
    return reply.send(config);
  },
  getVectorPromotionV1: (request, reply) => reply.send(promotionFor(request)),
  listArtifactDeploymentsV1: async (request, reply) => {
    const { landscapeId, vectorDeploymentId } = request.query as Query<"listArtifactDeploymentsV1">;
    const data = inLandscape(projectFor(request).artifactDeployments, landscapeId).filter(
      (deployment) =>
        !vectorDeploymentId || deployment.vectorDeploymentIds.includes(vectorDeploymentId),
    );
    return reply.send({ data });
  },
  listLandscapesV1: (request, reply) => reply.send({ data: projectFor(request).landscapes }),
  listProjectsV1: (request, reply) =>
    reply.send({ data: scenarioFor(request).projects.map(({ project }) => project) }),
  listStagesV1: (request, reply) => {
    const { landscapeId } = request.query as Query<"listStagesV1">;
    const project = projectFor(request);
    requireLandscape(project, landscapeId);
    return reply.send({ data: inLandscape(project.stages, landscapeId) });
  },
  listVectorDeploymentsV1: (request, reply) => {
    const { landscapeId } = request.query as Query<"listVectorDeploymentsV1">;
    const project = projectFor(request);
    requireLandscape(project, landscapeId);
    return reply.send({ data: inLandscape(project.vectorDeployments, landscapeId) });
  },
  listVectorPromotionConfigsV1: (request, reply) =>
    reply.send({ data: projectFor(request).vectorPromotionConfigs }),
  loginV1: (request, reply) => {
    const { return_url: returnUrl } = request.query as Query<"loginV1">;
    const state = encodeURIComponent(validatedReturnUrl(returnUrl, "Invalid return URL"));
    return reply.redirect(`/api/v1/auth/callback?code=${MOCK_CODE}&state=${state}`);
  },
  logoutV1: (_request, reply) => reply.clearCookie(SESSION_COOKIE, COOKIE_OPTIONS).send(),
  postExchangeCodeV1: (_request, reply) =>
    reply.setCookie(SESSION_COOKIE, MOCK_SESSION, COOKIE_OPTIONS).send(),
} satisfies Record<keyof operations, MockHandler>;

const securityHandlers = {
  sessionCookie: (request: FastifyRequest): void => {
    if (request.cookies[SESSION_COOKIE] !== MOCK_SESSION) {
      throw httpError(401, "Authentication required");
    }
  },
};

export { delay, operationHandlers, securityHandlers, UI_ORIGINS };
