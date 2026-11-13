import { inspectExample } from "./compile-example.ts";
import type { CompileRequest, CompileResult } from "./messages.ts";

const resultFor = ({ id, source }: CompileRequest): CompileResult => ({
  ...inspectExample(source),
  id,
});

globalThis.addEventListener("message", (event: MessageEvent<CompileRequest>): void => {
  globalThis.postMessage(resultFor(event.data));
});
