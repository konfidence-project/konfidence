<script lang="ts">
    import type { HTMLInputAttributes } from "svelte/elements";
    import "@ui5/webcomponents-icons/dist/search.js";
    import "@ui5/webcomponents/dist/Icon.js";

    interface Props extends Omit<HTMLInputAttributes, "class" | "value" | "type"> {
        value?: string;
        class?: string;
        placeholder?: string;
    }

    let {
        value = $bindable(""),
        class: className,
        placeholder = "Search\u2026",
        disabled,
        ...rest
    }: Props = $props();

    const wrapperClass = $derived(
        [
            "relative inline-flex w-full max-w-[24rem] items-center gap-1.5 px-2.5",
            "rounded-[var(--input-radius)] border border-[color:var(--input-bd)] bg-[color:var(--input-bg)] text-[color:var(--input-fg)]",
            "transition-[border-color,box-shadow] duration-[var(--motion-fast)]",
            "focus-within:border-[color:var(--border-strong)] focus-within:shadow-[var(--focus-ring)]",
            disabled && "cursor-not-allowed opacity-50 focus-within:border-[color:var(--input-bd)] focus-within:shadow-none",
            className,
        ]
            .filter(Boolean)
            .join(" "),
    );
</script>

<span class={wrapperClass}>
    <ui5-icon
        class="text-[color:var(--text-tertiary)] h-[var(--icon-md)] w-[var(--icon-md)] shrink-0"
        name="search"
        aria-hidden="true"
    ></ui5-icon>
    <input
        {...rest}
        {disabled}
        type="search"
        {placeholder}
        bind:value
        class="w-full min-w-0 border-0 bg-transparent py-2 text-[length:var(--text-sm)] text-inherit outline-none placeholder:text-[color:var(--input-placeholder)] disabled:cursor-not-allowed [&::-webkit-search-cancel-button]:appearance-none"
        autocomplete="off"
        spellcheck="false"
    />
    {#if value && !disabled}
        <button
            type="button"
            class="cursor-pointer rounded-[var(--radius-sm)] border-0 bg-transparent px-1 py-0.5 text-base leading-none text-[color:var(--text-tertiary)] hover:bg-[color:var(--surface-sunken)] hover:text-[color:var(--text-primary)] focus-visible:outline-none focus-visible:shadow-[var(--focus-ring)]"
            onclick={() => (value = "")}
            aria-label="Clear search"
            data-testid="search-clear"
        >&times;</button>
    {/if}
</span>
