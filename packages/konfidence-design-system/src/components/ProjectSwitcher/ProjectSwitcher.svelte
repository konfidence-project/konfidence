<script lang="ts">
    import Icon from "../Icon/Icon.svelte";

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
    <Icon name="chevron-down" size={14} />
</button>

