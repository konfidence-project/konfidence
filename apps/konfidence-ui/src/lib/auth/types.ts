import type { components } from "$lib/konfidence-api/schema";

type ApiIdentity = components["schemas"]["Identity"];

type AuthRole = string;

interface AuthUser {
  email: string;
  familyName: string;
  givenName: string;
  name: string;
  projectRoles: Record<string, string[]>;
  roles: AuthRole[];
}

type AuthStatus = "authenticated" | "idle" | "loading" | "unauthenticated";

interface AuthState {
  error?: string;
  status: AuthStatus;
  user: AuthUser | null;
}

const toAuthUser = (identity: ApiIdentity): AuthUser => ({
  email: identity.email,
  familyName: identity.familyName,
  givenName: identity.givenName,
  name: identity.name,
  projectRoles: identity.projectRoles,
  roles: [...new Set(Object.values(identity.projectRoles).flat())],
});

export { toAuthUser };
export type { ApiIdentity, AuthRole, AuthState, AuthStatus, AuthUser };
