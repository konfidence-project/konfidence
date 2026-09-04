import type { FastifyReply, FastifyRequest } from "fastify";
import { scenarios, type ProjectFixture, type ScenarioFixture } from "./fixtures.js";
import type { operations } from "@konfidence/api-client/schema";

const SESSION_COOKIE = "kden-session";
const SCENARIO_COOKIE = "konfidence_mock_scenario";
const MOCK_SESSION = "mock-session";
const MOCK_CODE = "mock-code";
const HTTP_NO_CONTENT = 204;
const COOKIE_OPTIONS = { httpOnly: true, path: "/", sameSite: "lax" } as const;

type Query<Name extends keyof operations> = NonNullable<operations[Name]["parameters"]["query"]>;
type MockHandler = (request: FastifyRequest, reply: FastifyReply) => FastifyReply;

// Thrown errors carry the status the error handler in server.ts reports.
const httpError = (status: number, message: string): Error =>
  Object.assign(new Error(message), { statusCode: status });

const absoluteUrl = (value: string | undefined, whenInvalid: string): string => {
  try {
    return new URL(String(value)).href;
  } catch {
    throw httpError(400, whenInvalid);
  }
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

// Looks up a single promotion across all of the project's configs, 404ing if absent.
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
      .redirect(absoluteUrl(state, "Invalid authentication state"));
  },
  getIdentityV1: (request, reply) => {
    const { projects, user } = scenarioFor(request);
    const projectRoles = Object.fromEntries(
      projects.map(({ project, roles }) => [project.id, roles]),
    );
    return reply.send({ ...user, projectRoles });
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
  listArtifactDeploymentsV1: (request, reply) => {
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
    const state = encodeURIComponent(absoluteUrl(returnUrl, "Invalid return URL"));
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

export { operationHandlers, securityHandlers };
