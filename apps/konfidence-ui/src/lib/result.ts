type Result<Value> = { ok: true; value: Value } | { ok: false; error: unknown };

const attempt = <Value>(operation: () => Value): Result<Value> => {
  try {
    return { ok: true, value: operation() };
  } catch (error) {
    return { ok: false, error };
  }
};

export { attempt };
export type { Result };
