<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    interface Props {
        /**
         * Visible project name. Truncated with ellipsis inside
         * `.project-switch__name` when it overflows the trigger.
         */
        name: string;
        /**
         * Extra class names for layout tweaks. Widely typed so
         * framework-owned attribute bags (Zag `getTriggerProps()`,
         * Skeleton's `ClassValue`) spread through without friction.
         */
        class?: unknown;
        /**
         * Any additional attributes are forwarded to the root `<button>`.
         * Kept loosely typed so framework-owned attribute bags (Zag's
         * `getTriggerProps()`) can spread through.
         */
        [key: string]: unknown;
    }

    let { name, class: className, ...rest }: Props = $props();

    const BASE_CLASS =
        "project-switch flex items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--surface-subtle)] py-1.5 px-2.5 text-[length:var(--text-sm)] font-semibold text-[var(--text-primary)] cursor-pointer hover:bg-[var(--surface-sunken)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--border-focus)]";

    const composedClass = $derived(
        typeof className === "string" && className.length > 0
            ? `${BASE_CLASS} ${className}`
            : BASE_CLASS,
    );
</script>

<button type="button" class={composedClass} {...rest}>
    <span class="project-switch__name">{name}</span>
    <ui5-icon class="size-3.5 text-current" name="slim-arrow-down"></ui5-icon>
</button>

