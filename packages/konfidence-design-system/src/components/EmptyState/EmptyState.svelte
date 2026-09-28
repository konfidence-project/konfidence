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
</script>

<div
    class={[
        "flex flex-col items-center justify-center gap-2 px-6 py-10 text-center text-content-secondary",
        className,
    ]}
    {role}
    aria-live={ariaLive}
>
    {#if icon}
        <div
            class="text-content-tertiary [&_ui5-icon]:size-icon-2xl"
            aria-hidden="true"
        >
            {@render icon()}
        </div>
    {/if}
    <h2
        class={[
            "m-0 text-ui-h3 font-semibold",
            tone === "error" ? "text-[color:var(--btn-danger-fg)]" : "text-content-primary",
        ]}
    >{title}</h2>
    {#if description}
        <p class="m-0 max-w-[40ch] text-ui-sm">{description}</p>
    {/if}
    {#if action}
        <div class="mt-3">{@render action()}</div>
    {/if}
</div>
