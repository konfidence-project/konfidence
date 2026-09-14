<script lang="ts">
    import type { HTMLSelectAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";

    interface Props extends Omit<HTMLSelectAttributes, "class" | "value" | "children"> {
        value?: string;
        class?: string;
        children: Snippet;
    }

    let {
        value = $bindable(""),
        class: className,
        disabled,
        children,
        ...rest
    }: Props = $props();
</script>

<select
    {...rest}
    {disabled}
    bind:value
    class={[
        "min-w-[12rem] appearance-none rounded-[var(--input-radius)] border border-[color:var(--input-bd)] bg-[color:var(--input-bg)] px-3 py-2",
        "text-[length:var(--text-sm)] text-[color:var(--input-fg)]",
        "focus-visible:border-[color:var(--border-strong)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className,
    ]}
>
    {@render children()}
</select>
