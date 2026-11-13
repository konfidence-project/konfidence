import type { CompileResult, ExampleCompilation, SourceDiagnostic } from "./messages.ts";

// One lazy compiler worker belongs to the preview tab. Individual samples own
// their pending requests and abort them when edited again or unmounted.
let worker: Worker | undefined = undefined;
let nextId = 0;
const pending = new Map<
  number,
  { resolve: (result: ExampleCompilation) => void; reject: (error: Error) => void }
>();

const getWorker = (): Worker => {
  if (worker) {
    return worker;
  }
  worker = new Worker(new URL("compiler.worker.ts", import.meta.url), { type: "module" });
  worker.addEventListener("message", (event: MessageEvent<CompileResult>) => {
    const result = event.data;
    const request = pending.get(result.id);
    if (!request) {
      return;
    }
    pending.delete(result.id);
    request.resolve(result);
  });
  worker.addEventListener("error", (event) => {
    event.preventDefault();
    for (const request of pending.values()) {
      request.reject(new Error(event.message || "Compiler worker failed"));
    }
    pending.clear();
    worker?.terminate();
    worker = undefined;
  });
  return worker;
};

const inspectInWorker = (source: string, signal: AbortSignal): Promise<ExampleCompilation> =>
  new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException("Compilation cancelled", "AbortError"));
      return;
    }
    const compiler = getWorker();
    const id = ++nextId;
    const abort = (): void => {
      pending.delete(id);
      reject(new DOMException("Compilation cancelled", "AbortError"));
    };
    signal.addEventListener("abort", abort, { once: true });
    pending.set(id, {
      reject(error) {
        signal.removeEventListener("abort", abort);
        reject(error);
      },
      resolve(result) {
        signal.removeEventListener("abort", abort);
        resolve(result);
      },
    });
    compiler.postMessage({ id, source });
  });

const compileInWorker = async (source: string, signal: AbortSignal): Promise<string> => {
  const result = await inspectInWorker(source, signal);
  if (result.error !== undefined) {
    throw new Error(result.error);
  }
  return result.code;
};

const lintInWorker = async (
  source: string,
  signal: AbortSignal,
): Promise<readonly SourceDiagnostic[]> => {
  const result = await inspectInWorker(source, signal);
  return result.diagnostics;
};

let runtime: Promise<string> | undefined = undefined;
const loadRuntime = (): Promise<string> => {
  runtime ??= import("virtual:playground-runtime")
    .then((module) => module.default)
    .catch((error: unknown) => {
      runtime = undefined;
      throw error;
    });
  return runtime;
};

// Only existing application styles are shared, not DOM nodes or mutable state.
let styles: Promise<string> | undefined = undefined;
const loadStyles = (): Promise<string> => {
  styles ??= Promise.all(
    [...document.querySelectorAll<HTMLLinkElement>('link[rel="stylesheet"]')].map(async (link) => {
      const response = await fetch(link.href);
      if (!response.ok) {
        throw new Error(`Could not load preview styles (${response.status})`);
      }
      return response.text();
    }),
  )
    .then(
      (sheets) =>
        sheets.join("\n") +
        [...document.querySelectorAll("style")].map((style) => style.textContent).join("\n"),
    )
    .catch((error: unknown) => {
      styles = undefined;
      throw error;
    });
  return styles;
};

export { compileInWorker, lintInWorker, loadRuntime, loadStyles };
