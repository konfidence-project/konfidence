type AuthRole = "ADMIN" | "DEV" | "PM";

interface AuthUser {
  email: string;
  familyName: string;
  givenName: string;
  middleName?: string;
  name: string;
  roles: AuthRole[];
}

interface AuthSession {
  user: AuthUser;
}

export type { AuthRole, AuthSession, AuthUser };
