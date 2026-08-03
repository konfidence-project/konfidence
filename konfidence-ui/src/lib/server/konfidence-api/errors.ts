import { type RequestEvent, error, redirect } from "@sveltejs/kit";
import {
  HTTP_BAD_GATEWAY,
  HTTP_BAD_REQUEST,
  HTTP_FORBIDDEN,
  HTTP_MAX_ERROR_STATUS,
  HTTP_SEE_OTHER,
  HTTP_UNAUTHORIZED,
} from "$lib/http-status";

interface ApiErrorOptions {
  code?: string;
  message: string;
}

const toErrorBody = (options: ApiErrorOptions | string): ApiErrorOptions => {
  if (typeof options === "string") {
    return { message: options };
  }
  return options;
};

const resolveErrorStatus = (responseStatus: number): number => {
  if (responseStatus >= HTTP_BAD_REQUEST && responseStatus <= HTTP_MAX_ERROR_STATUS) {
    return responseStatus;
  }
  return HTTP_BAD_GATEWAY;
};

const throwApiError = (
  event: RequestEvent,
  response: Response,
  options: ApiErrorOptions | string,
): never => {
  const errorBody = toErrorBody(options);

  if (response.status === HTTP_UNAUTHORIZED) {
    const returnTo = event.url.pathname + event.url.search;
    redirect(HTTP_SEE_OTHER, `/api/login?returnTo=${encodeURIComponent(returnTo)}`);
  }
  if (response.status === HTTP_FORBIDDEN) {
    error(HTTP_FORBIDDEN, {
      code: "FORBIDDEN",
      message: "You do not have permission to access this resource",
    });
  }

  error(resolveErrorStatus(response.status), errorBody);
};

export { throwApiError };
export type { ApiErrorOptions };
