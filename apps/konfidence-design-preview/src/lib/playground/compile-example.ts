import { compile, parse } from "svelte/compiler";
import type { CompileResult as SvelteCompileResult, Warning } from "svelte/compiler";
import type { ExampleCompilation, SourceDiagnostic } from "./messages.ts";

const COMPONENTS = "@konfidence/design-system/components";
const SIDE_EFFECTS = [
  "@ui5/webcomponents/dist/Icon.js",
  "@ui5/webcomponents-icons/dist/AllIcons.js",
];
const MODULES = new Set(["svelte", COMPONENTS, "@konfidence/design-system", ...SIDE_EFFECTS]);

const checkedCompile = (source: string): SvelteCompileResult => {
  const ast = parse(source, { modern: true });
  for (const script of [ast.instance, ast.module]) {
    for (const statement of script?.content.body ?? []) {
      if (statement.type === "ImportDeclaration" && !MODULES.has(String(statement.source.value))) {
        throw new Error(
          `Unsupported import: ${statement.source.value}. Use svelte or @konfidence/design-system/components.`,
        );
      }
    }
  }
  return compile(source, {
    css: "injected",
    filename: "Example.svelte",
    generate: "client",
    runes: true,
  });
};

const compileExample = (source: string): string => checkedCompile(source).js.code;

const errorDiagnostic = (error: unknown, length: number): SourceDiagnostic => {
  const position =
    error !== null &&
    typeof error === "object" &&
    "position" in error &&
    Array.isArray(error.position)
      ? error.position
      : [];
  const from = Math.min(length, Math.max(0, typeof position[0] === "number" ? position[0] : 0));
  const to = Math.min(
    length,
    Math.max(from, typeof position[1] === "number" ? position[1] : from + 1),
  );
  return {
    from,
    message: error instanceof Error ? error.message : String(error),
    severity: "error",
    to,
  };
};

const warningDiagnostic = (warning: Warning): SourceDiagnostic => ({
  from: warning.start?.character ?? 0,
  message: warning.message,
  severity: "warning",
  to: warning.end?.character ?? warning.start?.character ?? 0,
});

// Rendering and editor linting use the same compiler/import boundary. Compiler
// diagnostics retain character ranges for CodeMirror without depending on it.
const inspectExample = (source: string): ExampleCompilation => {
  try {
    const result = checkedCompile(source);
    return { code: result.js.code, diagnostics: result.warnings.map(warningDiagnostic) };
  } catch (error) {
    const diagnostic = errorDiagnostic(error, source.length);
    return { diagnostics: [diagnostic], error: diagnostic.message };
  }
};

export { COMPONENTS, SIDE_EFFECTS, compileExample, inspectExample };
