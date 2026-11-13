import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { test } from "node:test";
import { compileExample, inspectExample } from "./compile-example.ts";
import { COMPONENT_SECTION_PATTERN, extractExamples } from "./extract-examples.ts";

test("editor diagnostics keep Svelte syntax and accessibility ranges", () => {
  const broken = "<p>unclosed";
  const syntax = inspectExample(broken);
  assert.ok(syntax.error);
  assert.equal(syntax.diagnostics[0].severity, "error");
  assert.ok(syntax.diagnostics[0].from >= 0 && syntax.diagnostics[0].to <= broken.length);
  const warning = inspectExample("<img />");
  assert.equal(warning.error, undefined);
  assert.ok(
    warning.diagnostics.some(
      (diagnostic) => diagnostic.severity === "warning" && diagnostic.message.includes("alt"),
    ),
  );
  assert.deepEqual(inspectExample("<p>Valid</p>").diagnostics, []);
});

test("examples include their transitive state and imports, not catalog props or unrelated samples", () => {
  const [example] = extractExamples(`<script lang="ts">
    import { Button, Select } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    let { name = "Button" } = $props();
    let counter = $state(0);
    let unrelated = $state("hidden");
    const increment = () => counter += 1;
  </script>
  <Sample title="Counter"><Button onclick={increment}>{counter}</Button></Sample>`);
  assert.match(example.source, /import \{ Button \}/);
  assert.match(example.source, /let counter = \$state\(0\)/);
  assert.match(example.source, /const increment/);
  assert.doesNotMatch(example.source, /unrelated|\$props|import Sample|Select/);
  assert.doesNotThrow(() => compileExample(example.source));
});

test("every component sample is independently compilable from its existing Svelte source", () => {
  const sections = new URL("../gallery/sections/", import.meta.url);
  let count = 0;
  for (const file of readdirSync(sections).filter((name) =>
    COMPONENT_SECTION_PATTERN.test(new URL(name, sections).pathname),
  )) {
    const examples = extractExamples(readFileSync(new URL(file, sections), "utf8"));
    assert.ok(examples.length > 0, `${file} needs at least one example`);
    for (const example of examples) {
      assert.doesNotThrow(() => compileExample(example.source), `${file}: ${example.source}`);
      count += 1;
    }
  }
  assert.ok(count >= 30, "Existing component examples must not silently disappear");
});

test("compiler reports syntax errors and rejects unsupported static imports", () => {
  assert.throws(() => compileExample("<Button"));
  assert.throws(
    () => compileExample('<script>import x from "https://example.com/x.js";</script>'),
    /Unsupported import/,
  );
  assert.doesNotThrow(() =>
    compileExample(
      '<script>import { Button } from "@konfidence/design-system/components";</script><Button>Hi</Button>',
    ),
  );
});
