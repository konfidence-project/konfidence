type AuthRole = "ADMIN" | "DEV" | "PM";

interface AuthUser {
  email?: string;
  emailVerified: boolean;
  id: string;
  name: string;
  role?: AuthRole;
  username: string;
}

interface AuthSession {
  user: AuthUser;
}

export type { AuthRole, AuthSession, AuthUser };
