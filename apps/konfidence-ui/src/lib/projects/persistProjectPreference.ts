const keyFor = (userEmail: string): string => `konfidence:last-project:${userEmail}`;

const readLastProject = (userEmail: string): string | undefined => {
  try {
    return globalThis.sessionStorage.getItem(keyFor(userEmail)) ?? undefined;
  } catch {
    return undefined;
  }
};

const persistProjectPreferenceToSessionStorage = (userEmail: string, projectId: string): void => {
  try {
    globalThis.sessionStorage.setItem(keyFor(userEmail), projectId);
  } catch {
    // Project selection remains usable when storage is unavailable, blocked, or full.
  }
};

export { persistProjectPreferenceToSessionStorage, readLastProject };
