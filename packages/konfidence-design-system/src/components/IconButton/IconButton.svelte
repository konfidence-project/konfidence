<script lang="ts">
    import type { HTMLButtonAttributes } from "svelte/elements";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    interface Props extends Omit<HTMLButtonAttributes, "class" | "aria-label"> {
        /** SAP-icons name, e.g. `bell`. */
        icon: string;
        /**
         * Accessible label. Required — the button has no visible text
         * so the SR-only name is the only affordance.
         */
        ariaLabel: string;
        /**
         * Optional numeric badge overlay (`.icon-btn__badge`). Values
         * above `99` render as `99+`.
         */
        badge?: number;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let {
        icon,
        ariaLabel,
        badge,
        class: className,
        type = "button",
        ...rest
    }: Props = $props();

    const composedClass = $derived(className ? `icon-btn ${className}` : "icon-btn");

    const BADGE_MAX = 99;
    const badgeLabel = $derived.by((): string | undefined => {
        if (badge === undefined) {
            return undefined;
        }
        return badge > BADGE_MAX ? `${BADGE_MAX}+` : String(badge);
    });
</script>

<button class={composedClass} {type} aria-label={ariaLabel} {...rest}>
    <ui5-icon name={icon}></ui5-icon>
    {#if badgeLabel !== undefined}
        <span class="icon-btn__badge">{badgeLabel}</span>
    {/if}
</button>

<style>
    ui5-icon {
        width: 20px;
        height: 20px;
        color: currentColor;
    }
</style>
