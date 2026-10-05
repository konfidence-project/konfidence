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

    const BADGE_MAX = 99;
    const badgeLabel = $derived.by((): string | undefined => {
        if (badge === undefined) {
            return undefined;
        }
        return badge > BADGE_MAX ? `${BADGE_MAX}+` : String(badge);
    });
</script>

<button class={["icon-btn relative flex size-9 cursor-pointer items-center justify-center rounded-base border border-transparent bg-transparent text-content-secondary transition-colors duration-[var(--motion-fast)] ease-[cubic-bezier(var(--ease))] hover:bg-surface-sunken focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-outline-focus", className]} {type} aria-label={ariaLabel} {...rest}>
    <ui5-icon class="size-5 text-current" name={icon}></ui5-icon>
    {#if badgeLabel !== undefined}
        <span
            class="icon-btn__badge absolute top-[3px] right-[3px] flex h-4 min-w-4 items-center justify-center rounded-full border-2 border-surface-default bg-status-error-solid px-1 text-xs font-bold text-white"
        >{badgeLabel}</span>
    {/if}
</button>
