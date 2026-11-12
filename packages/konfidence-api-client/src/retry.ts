/* eslint-disable no-await-in-loop -- Each retry depends on the previous response and backoff. */

type RequestFetch = (request: Request, init?: RequestInit) => Promise<Response>;

const conflictMaxAttempts = 3;
const conflictBackoffMs = 100;

// Retry HTTP 409 conflicts caused by optimistic locking.
const withConflictRetries =
  (fetch: RequestFetch): RequestFetch =>
  async (request, init) => {
    for (let attempt = 1; ; attempt++) {
      // Fetch consumes the body, so keep the original available for each retry.
      const response = await fetch(request.clone(), init);
      if (response.status !== 409 || attempt >= conflictMaxAttempts) {
        return response;
      }

      await response.body?.cancel();
      await new Promise<void>((resolve) => globalThis.setTimeout(resolve, conflictBackoffMs));
    }
  };

export { withConflictRetries };
