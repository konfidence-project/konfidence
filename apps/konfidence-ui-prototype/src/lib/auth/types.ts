type AuthRole = string;

interface AuthUser {
  email: string;
  familyName: string;
  givenName: string;
  middleName?: string;
  name: string;
  projectRoles: Record<string, string[]>;
  roles: AuthRole[];
}

interface AuthSession {
  user: AuthUser;
}

export type { AuthRole, AuthSession, AuthUser };
