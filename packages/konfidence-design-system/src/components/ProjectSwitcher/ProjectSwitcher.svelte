<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    interface Props {
        /**
         * Visible project name. Truncated with ellipsis inside
         * `.proj-switch__name` when it overflows the trigger.
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

    const composedClass = $derived(
        typeof className === "string" && className.length > 0
            ? `proj-switch ${className}`
            : "proj-switch",
    );
</script>

<button type="button" class={composedClass} {...rest}>
    <span class="proj-switch__name">{name}</span>
    <ui5-icon name="slim-arrow-down"></ui5-icon>
</button>

<style>
    ui5-icon {
        width: 14px;
        height: 14px;
        color: currentColor;
    }
</style>

