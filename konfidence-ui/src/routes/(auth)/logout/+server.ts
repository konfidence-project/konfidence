import { type RequestHandler, redirect } from "@sveltejs/kit";
import { SESSION_COOKIE_NAME, destroySession } from "$lib/server/session";

const HTTP_SEE_OTHER = 303;

export const POST: RequestHandler = ({ cookies }) => {
  const sid = cookies.get(SESSION_COOKIE_NAME);
  destroySession(sid);
  cookies.delete(SESSION_COOKIE_NAME, { path: "/" });
  redirect(HTTP_SEE_OTHER, "/login");
};
