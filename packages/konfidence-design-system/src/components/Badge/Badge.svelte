<script lang="ts">
    import type { Snippet } from "svelte";
    import type { HTMLAttributes } from "svelte/elements";
    import type { TagColor } from "../Tag/types.js";

    interface Props extends Omit<HTMLAttributes<HTMLSpanElement>, "children" | "class"> {
        /** Semantic state. Unknown values retain their label and inherit the surrounding tone. */
        status: string;
        /** Whether to show the leading state dot. Defaults to `true`. */
        showDot?: boolean;
        /** Pulse the optional dot for any state; respects reduced motion. */
        animated?: boolean;
        /** Filled state surface (default) or a subtle tinted surface and border. */
        variant?: "filled" | "subtle";
        /** Small (default) or medium with larger text and spacing. */
        size?: "sm" | "md";
        /** Optional category palette, independent of the state label. */
        color?: TagColor;
        /** Extra class names for layout tweaks. */
        class?: string;
        /**
         * Human-readable label. Required — a badge without visible text
         * violates the design-system auditability rule.
         */
        children: Snippet;
    }

    let { status, showDot = true, animated = false, variant = "filled", size = "sm", color, class: className, children, ...rest }: Props = $props();

    const STATUS_CLASSES: Record<string, { badge: string; dot: string }> = {
        active: { badge: "text-status-deploying-fg bg-status-deploying-bg", dot: "bg-status-deploying-solid" },
        blocked: { badge: "text-status-warning-fg bg-status-warning-bg", dot: "bg-status-warning-solid" },
        degraded: { badge: "text-status-degraded-fg bg-status-degraded-bg", dot: "bg-status-degraded-solid" },
        deploying: { badge: "text-status-deploying-fg bg-status-deploying-bg", dot: "bg-status-deploying-solid" },
        error: { badge: "text-status-error-fg bg-status-error-bg", dot: "bg-status-error-solid" },
        failed: { badge: "text-status-error-fg bg-status-error-bg", dot: "bg-status-error-solid" },
        healthy: { badge: "text-status-healthy-fg bg-status-healthy-bg", dot: "bg-status-healthy-solid" },
        promoting: { badge: "text-status-promoting-fg bg-status-promoting-bg", dot: "bg-status-promoting-solid" },
        queued: { badge: "text-status-queued-fg bg-status-queued-bg", dot: "bg-status-queued-solid" },
        ready: { badge: "text-status-promoting-fg bg-status-promoting-bg", dot: "bg-status-promoting-solid" },
        rejected: { badge: "text-[var(--status-rejected-fg)] bg-[var(--status-rejected-bg)]", dot: "bg-[var(--status-rejected-solid)]" },
        succeeded: { badge: "text-status-healthy-fg bg-status-healthy-bg", dot: "bg-status-healthy-solid" },
        superseded: { badge: "text-[var(--status-superseded-fg)] bg-[var(--status-superseded-bg)]", dot: "bg-[var(--status-superseded-solid)]" },
        waiting: { badge: "text-status-queued-fg bg-status-queued-bg", dot: "bg-status-queued-solid" },
        warning: { badge: "text-status-warning-fg bg-status-warning-bg", dot: "bg-status-warning-solid" },
    };
    const STATE_COLOR: Record<string, TagColor> = { active: "blue", blocked: "amber", degraded: "orange", deploying: "blue", error: "red", failed: "red", healthy: "green", promoting: "violet", queued: "neutral", ready: "violet", rejected: "red", succeeded: "green", superseded: "slate", waiting: "neutral", warning: "amber" };
    const palette = $derived(color ?? STATE_COLOR[status] ?? "neutral");
</script>

<span class={[
    "badge inline-flex items-center whitespace-nowrap border border-transparent font-semibold",
    size === "md"
        ? "badge--md gap-2 rounded-full text-compact leading-normal"
        : "gap-1.5 rounded-[var(--badge-radius)] pt-[var(--badge-py)] pb-[var(--badge-py)] pr-[var(--badge-px)] pl-2 text-meta leading-[1.4]",
    `badge--${status}`,
    STATUS_CLASSES[status]?.badge,
    animated && "badge--animated",
    variant === "subtle" && "badge--subtle",
    color && "badge--colored",
    className,
]} data-status={status} data-color={palette} {...rest}>
    {#if showDot}
        <span class={["dot size-2 shrink-0 rounded-full", STATUS_CLASSES[status]?.dot]} aria-hidden="true"></span>
    {/if}
    {@render children()}
</span>

<style>
    /* Include the border in the medium size spacing. */
    .badge--md { padding: calc(var(--spacing) * 1.5 - 1px) calc(var(--spacing) * 3 - 1px); }
    .badge--animated > .dot { animation: badge-pulse 1.4s ease-in-out infinite; }
    @keyframes badge-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } }
    @media (prefers-reduced-motion: reduce) { .badge--animated > .dot { animation: none; } }
    .badge--colored { color: var(--badge-filled-fg); background: var(--badge-filled-bg); }
    .badge--colored > .dot { background: var(--badge-filled-solid); }
    .badge--subtle { color: var(--badge-palette-fg, var(--text-secondary)); background: var(--badge-palette-bg, var(--surface-sunken)); }
    .badge--subtle { border-color: var(--badge-palette-bd, var(--border-subtle)); }
    .badge--subtle > .dot { background: currentColor; }
    [data-color="red"] { --badge-filled-fg: var(--status-error-fg); --badge-filled-bg: var(--status-error-bg); --badge-filled-solid: var(--status-error-solid); --badge-palette-fg: var(--tag-red-fg); --badge-palette-bg: var(--tag-red-bg); --badge-palette-bd: var(--tag-red-bd); }
    [data-color="orange"] { --badge-filled-fg: var(--status-degraded-fg); --badge-filled-bg: var(--status-degraded-bg); --badge-filled-solid: var(--status-degraded-solid); --badge-palette-fg: var(--tag-orange-fg); --badge-palette-bg: var(--tag-orange-bg); --badge-palette-bd: var(--tag-orange-bd); }
    [data-color="amber"] { --badge-filled-fg: var(--status-warning-fg); --badge-filled-bg: var(--status-warning-bg); --badge-filled-solid: var(--status-warning-solid); --badge-palette-fg: var(--tag-amber-fg); --badge-palette-bg: var(--tag-amber-bg); --badge-palette-bd: var(--tag-amber-bd); }
    [data-color="lime"] { --badge-filled-fg: var(--badge-lime-fg); --badge-filled-bg: var(--badge-lime-bg); --badge-filled-solid: var(--badge-lime-solid); --badge-palette-fg: var(--tag-lime-fg); --badge-palette-bg: var(--tag-lime-bg); --badge-palette-bd: var(--tag-lime-bd); }
    [data-color="green"] { --badge-filled-fg: var(--status-healthy-fg); --badge-filled-bg: var(--status-healthy-bg); --badge-filled-solid: var(--status-healthy-solid); --badge-palette-fg: var(--tag-green-fg); --badge-palette-bg: var(--tag-green-bg); --badge-palette-bd: var(--tag-green-bd); }
    [data-color="teal"] { --badge-filled-fg: var(--badge-teal-fg); --badge-filled-bg: var(--badge-teal-bg); --badge-filled-solid: var(--badge-teal-solid); --badge-palette-fg: var(--tag-teal-fg); --badge-palette-bg: var(--tag-teal-bg); --badge-palette-bd: var(--tag-teal-bd); }
    [data-color="cyan"] { --badge-filled-fg: var(--badge-cyan-fg); --badge-filled-bg: var(--badge-cyan-bg); --badge-filled-solid: var(--badge-cyan-solid); --badge-palette-fg: var(--tag-cyan-fg); --badge-palette-bg: var(--tag-cyan-bg); --badge-palette-bd: var(--tag-cyan-bd); }
    [data-color="blue"] { --badge-filled-fg: var(--status-deploying-fg); --badge-filled-bg: var(--status-deploying-bg); --badge-filled-solid: var(--status-deploying-solid); --badge-palette-fg: var(--tag-blue-fg); --badge-palette-bg: var(--tag-blue-bg); --badge-palette-bd: var(--tag-blue-bd); }
    [data-color="indigo"] { --badge-filled-fg: var(--badge-indigo-fg); --badge-filled-bg: var(--badge-indigo-bg); --badge-filled-solid: var(--badge-indigo-solid); --badge-palette-fg: var(--tag-indigo-fg); --badge-palette-bg: var(--tag-indigo-bg); --badge-palette-bd: var(--tag-indigo-bd); }
    [data-color="violet"] { --badge-filled-fg: var(--status-promoting-fg); --badge-filled-bg: var(--status-promoting-bg); --badge-filled-solid: var(--status-promoting-solid); --badge-palette-fg: var(--tag-violet-fg); --badge-palette-bg: var(--tag-violet-bg); --badge-palette-bd: var(--tag-violet-bd); }
    [data-color="pink"] { --badge-filled-fg: var(--badge-pink-fg); --badge-filled-bg: var(--badge-pink-bg); --badge-filled-solid: var(--badge-pink-solid); --badge-palette-fg: var(--tag-pink-fg); --badge-palette-bg: var(--tag-pink-bg); --badge-palette-bd: var(--tag-pink-bd); }
    [data-color="slate"] { --badge-filled-fg: var(--status-superseded-fg); --badge-filled-bg: var(--status-superseded-bg); --badge-filled-solid: var(--status-superseded-solid); --badge-palette-fg: var(--tag-slate-fg); --badge-palette-bg: var(--tag-slate-bg); --badge-palette-bd: var(--tag-slate-bd); }
    [data-color="neutral"] { --badge-filled-fg: var(--status-queued-fg); --badge-filled-bg: var(--status-queued-bg); --badge-filled-solid: var(--status-queued-solid); }
</style>
