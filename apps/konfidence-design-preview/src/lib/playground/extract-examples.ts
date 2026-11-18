import { parse } from "svelte/compiler";
import type { AST } from "svelte/compiler";

const COMPONENT_SECTION_PATTERN =
  /\/gallery\/sections\/(?:Buttons|DataBlocks|Feedback|Forms|Hero|Lists|Navigation|Overlays|Pipeline|Primitives|Shell|ShellPatterns|StatusBadges)\.svelte$/;

// Svelte does not expose an AST walker. This visitor only reads compiler-owned
// syntax trees, including the ESTree script and Svelte template nodes.
const visit = (value: unknown, callback: (node: Record<string, unknown>) => void): void => {
  if (Array.isArray(value)) {
    for (const item of value) {
      visit(item, callback);
    }
  } else if (value !== null && typeof value === "object") {
    const node = value as Record<string, unknown>;
    if (typeof node.type === "string") {
      callback(node);
    }
    for (const [key, child] of Object.entries(node)) {
      if (key !== "loc" && key !== "name_loc") {
        visit(child, callback);
      }
    }
  }
};

const namesIn = (value: unknown): Set<string> => {
  const names = new Set<string>();
  visit(value, (node) => {
    if (node.type === "Identifier" && typeof node.name === "string") {
      names.add(node.name);
    }
    if (node.type === "Component" && typeof node.name === "string") {
      names.add(node.name.split(".")[0]);
    }
  });
  return names;
};

const dedent = (text: string): string => {
  const lines = text.replace(/^\n/, "").trimEnd().split("\n");
  const indentation = Math.min(
    ...lines.filter((line) => line.trim()).map((line) => line.match(/^\s*/)?.[0].length ?? 0),
  );
  return lines.map((line) => line.slice(indentation)).join("\n");
};

interface ExtractedExample {
  attributeOffset: number;
  source: string;
}

const sliceNode = (source: string, node: object): string => {
  if (
    !("start" in node) ||
    !("end" in node) ||
    typeof node.start !== "number" ||
    typeof node.end !== "number"
  ) {
    throw new Error("Compiler node is missing source offsets");
  }
  return source.slice(node.start, node.end);
};

type Statement = NonNullable<AST.Root["instance"]>["content"]["body"][number];
type ImportStatement = Extract<Statement, { type: "ImportDeclaration" }>;

const declaredNames = (statement: Statement): Set<string> => {
  if (statement.type === "VariableDeclaration") {
    return namesIn(statement.declarations.map((declaration) => declaration.id));
  }
  if ("id" in statement) {
    return namesIn(statement.id);
  }
  return new Set();
};

// Include transitive state, handlers and types, but not the catalog's props or
// presentation wrappers. Each sample owns its independent runtime state.
const selectStatements = (
  statements: readonly Statement[],
  needed: Set<string>,
): ReadonlySet<Statement> => {
  const selected = new Set<Statement>();
  let previousCount = -1;
  while (selected.size !== previousCount) {
    previousCount = selected.size;
    for (const statement of statements) {
      if (
        !selected.has(statement) &&
        !namesIn(statement).has("$props") &&
        [...declaredNames(statement)].some((name) => needed.has(name))
      ) {
        selected.add(statement);
        for (const name of namesIn(statement)) {
          needed.add(name);
        }
      }
    }
  }
  return selected;
};

const renderImport = (
  source: string,
  statement: ImportStatement,
  needed: ReadonlySet<string>,
): string => {
  if (statement.specifiers.length === 0) {
    return sliceNode(source, statement);
  }
  const specifiers = statement.specifiers.filter((specifier) => needed.has(specifier.local.name));
  if (specifiers.length === 0) {
    return "";
  }
  const named = specifiers.filter((specifier) => specifier.type === "ImportSpecifier");
  const bindings = specifiers
    .filter((specifier) => specifier.type !== "ImportSpecifier")
    .map((specifier) => sliceNode(source, specifier));
  if (named.length > 0) {
    bindings.push(`{ ${named.map((specifier) => sliceNode(source, specifier)).join(", ")} }`);
  }
  return `import ${"importKind" in statement && statement.importKind === "type" ? "type " : ""}${bindings.join(", ")} from ${JSON.stringify(statement.source.value)};`;
};

const sampleScript = (source: string, ast: AST.Root, sample: AST.Component): string => {
  const needed = namesIn(sample.fragment);
  const statements = ast.instance?.content.body ?? [];
  const selected = selectStatements(statements, needed);
  const script = statements
    .map((statement) => {
      if (statement.type === "ImportDeclaration") {
        return renderImport(source, statement, needed);
      }
      return selected.has(statement) ? sliceNode(source, statement) : "";
    })
    .filter(Boolean)
    .join("\n\n");
  return script ? `<script lang="ts">\n${script}\n</script>\n\n` : "";
};

const sampleMarkup = (source: string, sample: AST.Component): string => {
  const [first] = sample.fragment.nodes;
  const last = sample.fragment.nodes.at(-1);
  return first && last ? dedent(source.slice(first.start, last.end)) : "";
};

const extractExamples = (source: string): readonly ExtractedExample[] => {
  const ast = parse(source, { modern: true });
  const samples: AST.Component[] = [];
  visit(ast.fragment, (node) => {
    if (node.type === "Component" && node.name === "Sample") {
      samples.push(node as unknown as AST.Component);
    }
  });
  const style = ast.css ? `\n\n${source.slice(ast.css.start, ast.css.end)}` : "";
  return samples.map((sample) => ({
    attributeOffset: sample.start + "<Sample".length,
    source: `${sampleScript(source, ast, sample)}<div class="flex flex-wrap items-center gap-3">\n${sampleMarkup(source, sample)}\n</div>${style}\n`,
  }));
};

export { COMPONENT_SECTION_PATTERN, extractExamples };
export type { ExtractedExample };
