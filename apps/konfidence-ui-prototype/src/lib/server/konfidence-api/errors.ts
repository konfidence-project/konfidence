import { type RequestEvent, error, redirect } from "@sveltejs/kit";
import {
  HTTP_BAD_GATEWAY,
  HTTP_BAD_REQUEST,
  HTTP_FORBIDDEN,
  HTTP_MAX_ERROR_STATUS,
  HTTP_SEE_OTHER,
  HTTP_UNAUTHORIZED,
} from "$lib/http-status";

const loginUrl = (event: RequestEvent): string => {
  const returnTo = event.url.pathname + event.url.search;
  const returnUrl = new globalThis.URL(returnTo, event.url.origin).href;
  return `/api/v1/login?return_url=${encodeURIComponent(returnUrl)}`;
};

const throwApiError = (event: RequestEvent, response: Response, message: string): never => {
  if (response.status === HTTP_UNAUTHORIZED) {
    redirect(HTTP_SEE_OTHER, loginUrl(event));
  }
  if (response.status === HTTP_FORBIDDEN) {
    error(HTTP_FORBIDDEN, "You do not have permission to access this resource");
  }

  const { status: responseStatus } = response;
  let status = HTTP_BAD_GATEWAY;
  if (responseStatus >= HTTP_BAD_REQUEST && responseStatus <= HTTP_MAX_ERROR_STATUS) {
    status = responseStatus;
  }
  error(status, message);
};

export { throwApiError };
