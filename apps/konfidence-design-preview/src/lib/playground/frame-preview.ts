import frameDocument from "./frame.html?raw";
import { compileInWorker, loadRuntime, loadStyles } from "./compiler-client.ts";
import type { FrameResult } from "./messages.ts";

type PreviewState =
  | { status: "loading" }
  | { status: "ready" }
  | { status: "error"; message: string };

interface FramePreview {
  render: (source: string, signal: AbortSignal) => Promise<void>;
  dispose: () => void;
}

const dark = (): boolean => document.documentElement.classList.contains("dark");

const observeTheme = (frame: HTMLIFrameElement): (() => void) => {
  const observer = new MutationObserver(() =>
    frame.contentWindow?.postMessage({ dark: dark(), type: "theme" }, "*"),
  );
  observer.observe(document.documentElement, { attributeFilter: ["class"], attributes: true });
  return () => observer.disconnect();
};

interface FrameResources {
  code: string;
  runtime: string;
  styles: string;
}

interface FrameOptions {
  title: string;
  onState: (state: PreviewState) => void;
}

const listenToFrame = (
  frame: HTMLIFrameElement,
  resources: FrameResources,
  options: FrameOptions,
): (() => void) => {
  let failed = false;
  const fail = (message: string): void => {
    failed = true;
    options.onState({ message, status: "error" });
  };
  const timeout = setTimeout(
    () => fail("Preview did not start. Reset the example to try again."),
    15_000,
  );
  const receive = ({ source, data: message }: MessageEvent<FrameResult>): void => {
    if (source !== frame.contentWindow || !message || typeof message !== "object") {
      return;
    }
    if (message.type === "focus-search") {
      // Keyboard events in the sandbox cannot bubble into the catalog window.
      globalThis.dispatchEvent(new Event("preview:focus-search"));
    } else if (message.type === "ready") {
      frame.contentWindow?.postMessage({ ...resources, dark: dark(), type: "render" }, "*");
    } else if (message.type === "error" && typeof message.message === "string") {
      clearTimeout(timeout);
      fail(message.message);
    } else if (message.type === "rendered") {
      clearTimeout(timeout);
      if (!failed) {
        options.onState({ status: "ready" });
      }
    }
  };
  window.addEventListener("message", receive);
  const stopTheme = observeTheme(frame);
  return () => {
    clearTimeout(timeout);
    window.removeEventListener("message", receive);
    stopTheme();
  };
};

const attachFrame = (
  container: HTMLElement,
  resources: FrameResources,
  options: FrameOptions,
): (() => void) => {
  const frame = document.createElement("iframe");
  frame.title = `Live preview: ${options.title}`;
  frame.sandbox.add("allow-scripts");
  frame.className = "block h-full w-full border-0";
  frame.srcdoc = frameDocument;
  const cleanup = listenToFrame(frame, resources, options);
  container.append(frame);
  return () => {
    cleanup();
    frame.remove();
  };
};

// A sample owns one displayed sandbox and its listeners. Compilation failure
// keeps the last render; replacement and unmount release the old frame together.
const createFramePreview = (
  container: HTMLElement,
  options: Omit<FrameOptions, "onState">,
  onState: (state: PreviewState) => void,
): FramePreview => {
  let disposeFrame: (() => void) | undefined = undefined;
  let disposed = false;
  return {
    dispose() {
      disposed = true;
      disposeFrame?.();
    },
    // eslint-disable-next-line max-statements -- Keep cancellation, replacement and recovery in one lifecycle.
    async render(source, signal) {
      if (signal.aborted || disposed) {
        return;
      }
      onState({ status: "loading" });
      try {
        const [code, runtime, styles] = await Promise.all([
          compileInWorker(source, signal),
          loadRuntime(),
          loadStyles(),
        ]);
        if (signal.aborted || disposed) {
          return;
        }
        disposeFrame?.();
        disposeFrame = attachFrame(
          container,
          { code, runtime, styles },
          {
            onState(state) {
              if (!signal.aborted && !disposed) {
                onState(state);
              }
            },
            ...options,
          },
        );
      } catch (error) {
        if (!signal.aborted && !disposed) {
          onState({
            message: error instanceof Error ? error.message : String(error),
            status: "error",
          });
        }
      }
    },
  };
};

export { createFramePreview };
export type { FramePreview, PreviewState };
