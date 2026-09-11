<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        title: string;
        description?: string;
        icon?: Snippet;
        action?: Snippet;
        tone?: "empty" | "error" | "info";
        class?: string;
    }

    let { title, description, icon, action, tone = "empty", class: className }: Props = $props();

    const role = $derived(tone === "error" ? "alert" : "status");
    const ariaLive = $derived(tone === "error" ? "assertive" : "polite");
    const containerClass = $derived(
        [
            "flex flex-col items-center justify-center gap-2 px-6 py-10 text-center text-[color:var(--text-secondary)]",
            className,
        ]
            .filter(Boolean)
            .join(" "),
    );
    const titleClass = $derived(
        [
            "m-0 text-[length:var(--text-h3)] font-semibold",
            tone === "error"
                ? "text-[color:var(--btn-danger-fg)]"
                : "text-[color:var(--text-primary)]",
        ].join(" "),
    );
</script>

<div class={containerClass} {role} aria-live={ariaLive}>
    {#if icon}
        <div
            class="text-[color:var(--text-tertiary)] [&_ui5-icon]:h-[var(--icon-2xl)] [&_ui5-icon]:w-[var(--icon-2xl)]"
            aria-hidden="true"
        >
            {@render icon()}
        </div>
    {/if}
    <h2 class={titleClass}>{title}</h2>
    {#if description}
        <p class="m-0 max-w-[40ch] text-[length:var(--text-sm)]">{description}</p>
    {/if}
    {#if action}
        <div class="mt-3">{@render action()}</div>
    {/if}
</div>
