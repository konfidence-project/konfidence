<script lang="ts">
    import { untrack } from "svelte";
    import { basicSetup } from "codemirror";
    import { Compartment, EditorState } from "@codemirror/state";
    import { EditorView } from "@codemirror/view";
    import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
    import { lintGutter, linter } from "@codemirror/lint";
    import { tags } from "@lezer/highlight";
    import { svelte } from "@replit/codemirror-lang-svelte";
    import { lintInWorker } from "./compiler-client.ts";
    import { mode } from "mode-watcher";

    let { value = $bindable(""), label, onChange }: { value?: string; label: string; onChange?: () => void } = $props();
    let host = $state<HTMLDivElement>();
    let editor = $state<EditorView>();
    const appearance = new Compartment();
    const modeTheme = (dark: boolean) => EditorView.theme({
        "&": { "--editor-selection-bg": `color-mix(in srgb, var(--border-focus) ${dark ? 35 : 60}%, var(--surface-default))` },
    }, { dark });

    const highlighting = syntaxHighlighting(HighlightStyle.define([
        { color: "var(--text-link)", tag: [tags.keyword, tags.tagName] },
        { color: "var(--status-healthy-fg)", tag: [tags.string, tags.attributeValue] },
        { color: "var(--status-warning-fg)", tag: [tags.number, tags.bool] },
        { color: "var(--text-tertiary)", fontStyle: "italic", tag: tags.comment },
    ]));
    // Match CodeMirror's focused selection specificity, not just its range class.
    const theme = EditorView.theme({
        "&": { backgroundColor: "var(--surface-default)", color: "var(--text-primary)" },
        "&.cm-editor .cm-selectionLayer .cm-selectionBackground, &.cm-editor.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground": { backgroundColor: "var(--editor-selection-bg)" },
        "&.cm-editor.cm-has-selection .cm-activeLine": { backgroundColor: "transparent" },
        "&.cm-focused": { outline: "none" },
        ".cm-activeLine, .cm-activeLineGutter": { backgroundColor: "var(--surface-subtle)" },
        ".cm-content": { minHeight: "320px" },
        ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--text-primary)" },
        ".cm-diagnostic-error": { borderLeftColor: "var(--status-error-fg)" },
        ".cm-diagnostic-warning": { borderLeftColor: "var(--status-warning-fg)" },
        ".cm-diagnosticText": { overflowWrap: "anywhere", whiteSpace: "pre-wrap" },
        ".cm-gutters": { backgroundColor: "var(--surface-subtle)", border: "none", color: "var(--text-tertiary)" },
        ".cm-line::selection, .cm-line ::selection": { color: "var(--text-primary)" },
        ".cm-panel-lint button": { backgroundColor: "var(--surface-subtle)", color: "var(--text-secondary)" },
        ".cm-panel.cm-panel-lint": { fontSize: "var(--font-size-meta)", lineHeight: "1.5" },
        ".cm-panel.cm-panel-lint ul [aria-selected]": { backgroundColor: "var(--surface-subtle)", color: "var(--text-primary)" },
        ".cm-scroller": { fontFamily: "var(--font-mono)", fontSize: "var(--font-size-meta)", lineHeight: "1.6", overflow: "auto" },
        ".cm-tooltip, .cm-panel.cm-panel-lint": { backgroundColor: "var(--surface-card)", borderColor: "var(--border-subtle)", color: "var(--text-primary)" },
    });

    $effect(() => {
        if (!host) {return;}
        let pending: AbortController | undefined = undefined;
        const diagnostics = linter(async (view) => {
            pending?.abort();
            const controller = new AbortController();
            pending = controller;
            try { return [...await lintInWorker(view.state.doc.toString(), controller.signal)]; }
            catch (error) {
                if (controller.signal.aborted) {return [];}
                return [{ from: 0, message: error instanceof Error ? error.message : String(error), severity: "error" as const, to: 0 }];
            }
        }, { autoPanel: true, delay: 300 });
        const initialAppearance = modeTheme(untrack(() => mode.current === "dark"));
        const view = new EditorView({
            parent: host,
            state: EditorState.create({
                doc: untrack(() => value),
                extensions: [basicSetup, svelte(), highlighting, theme, appearance.of(initialAppearance), lintGutter(), diagnostics, EditorView.lineWrapping,
                    EditorView.editorAttributes.of((currentView) => ({ class: currentView.state.selection.ranges.some((range) => !range.empty) ? "cm-has-selection" : "" })),
                    EditorView.contentAttributes.of({ "aria-label": label }),
                    EditorView.updateListener.of((update) => { if (update.docChanged) { value = update.state.doc.toString(); onChange?.(); } }),
                ],
            }),
        });
        editor = view;
        return () => { pending?.abort(); view.destroy(); };
    });

    $effect(() => {
        if (editor) {editor.dispatch({ effects: appearance.reconfigure(modeTheme(mode.current === "dark")) });}
    });

    $effect(() => {
        if (editor && value !== editor.state.doc.toString()) {
            editor.dispatch({ changes: { from: 0, insert: value, to: editor.state.doc.length } });
        }
    });
</script>

<div bind:this={host} class="max-h-[560px] min-w-0 overflow-auto rounded-base border border-outline-default focus-within:outline-2 focus-within:outline-outline-focus"></div>
