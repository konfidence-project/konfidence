import { build } from "vite";
import type { Plugin } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import SourceString from "magic-string";
import { COMPONENT_SECTION_PATTERN, extractExamples } from "./extract-examples.ts";

const VIRTUAL_RUNTIME = "virtual:playground-runtime";

const playgroundPlugin = (root: string): Plugin => {
  let runtime: Promise<string> | undefined = undefined;
  const bundleRuntime = async (): Promise<string> => {
    const result = await build({
      build: {
        lib: {
          entry: `${root}/src/lib/playground/runtime.ts`,
          formats: ["es"],
        },
        rolldownOptions: { output: { codeSplitting: false } },
        write: false,
      },
      configFile: false,
      logLevel: "warn",
      plugins: [svelte({ compilerOptions: { css: "injected" }, configFile: false })],
      publicDir: false,
      root,
    });
    const outputs = Array.isArray(result) ? result : [result];
    const chunks = outputs.flatMap((output) => ("output" in output ? output.output : []));
    const chunk = chunks.find((output) => output.type === "chunk" && output.isEntry);
    if (!chunk || chunk.type !== "chunk") {
      throw new Error("Playground runtime bundle is missing");
    }
    return chunk.code;
  };

  return {
    enforce: "pre",
    handleHotUpdate(context) {
      if (
        context.file.includes("/packages/konfidence-design-system/src/") ||
        context.file === `${root}/src/app.css`
      ) {
        // The secondary bundle has no client HMR graph. Rebuild it and reload
        // its sandbox consumers rather than continuing with stale DS code/CSS.
        runtime = undefined;
        const module = context.server.moduleGraph.getModuleById(`\0${VIRTUAL_RUNTIME}`);
        if (module) {
          context.server.moduleGraph.invalidateModule(module);
        }
        context.server.ws.send({ type: "full-reload" });
        return [];
      }
    },
    async load(id) {
      if (id !== `\0${VIRTUAL_RUNTIME}`) {
        return;
      }
      runtime ??= bundleRuntime();
      return `export default ${JSON.stringify(await runtime)};`;
    },
    name: "konfidence-playground",
    resolveId(id) {
      if (id === VIRTUAL_RUNTIME) {
        return `\0${VIRTUAL_RUNTIME}`;
      }
    },
    transform: {
      filter: { id: COMPONENT_SECTION_PATTERN },
      handler(code, id) {
        const transformed = new SourceString(code);
        for (const example of extractExamples(code)) {
          transformed.appendLeft(
            example.attributeOffset,
            ` source={${JSON.stringify(example.source)}}`,
          );
        }
        return {
          code: transformed.toString(),
          map: transformed.generateMap({ hires: true, source: id }),
        };
      },
    },
  };
};

export { playgroundPlugin };
