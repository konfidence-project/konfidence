<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    interface Props {
        name: string;
        class?: unknown;
        [key: string]: unknown;
    }

    let { name, class: className, ...rest }: Props = $props();

    const BASE_CLASS =
        "project-switch flex cursor-pointer items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--surface-subtle)] px-2.5 py-1.5 text-[length:var(--text-sm)] font-semibold text-[var(--text-primary)] hover:bg-[var(--surface-sunken)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--border-focus)]";

    const composedClass = $derived(
        typeof className === "string" && className.length > 0
            ? `${BASE_CLASS} ${className}`
            : BASE_CLASS,
    );
</script>

<button type="button" class={composedClass} aria-label="Change project" {...rest}>
    <span class="project-switch__name min-w-0 flex-1 truncate text-left">{name}</span>
    <ui5-icon class="size-3.5 text-current" name="slim-arrow-down"></ui5-icon>
</button>
