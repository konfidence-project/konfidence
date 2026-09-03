<script lang="ts">
    import { ICONS } from "./icons.js";
    import type { IconName } from "./icons.js";

    /** Default width/height in pixels. Matches the source `viewBox` size. */
    const DEFAULT_SIZE = 16;

    interface Props {
        /** Icon identifier from the curated SAP-Icons set. */
        name: IconName;
        /**
         * Rendered pixel size (width and height). Inherits colour from
         * `currentColor` so consumers set colour via CSS on a parent.
         * Defaults to 16 — the source `viewBox` size.
         */
        size?: number;
        /**
         * Whether the icon should be hidden from assistive technology.
         * Defaults to `true` because icons in shell chrome are decorative
         * (a text label sits alongside). Set to `false` on standalone
         * icons that carry meaning and pair with `aria-label` on the
         * parent.
         */
        ariaHidden?: boolean;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let {
        name,
        size = DEFAULT_SIZE,
        ariaHidden = true,
        class: className,
    }: Props = $props();

    const path = $derived(ICONS[name]);
</script>

<svg
    class={className}
    width={size}
    height={size}
    viewBox="0 0 16 16"
    fill="currentColor"
    aria-hidden={ariaHidden ? "true" : undefined}
    data-icon={name}
    focusable="false"
>
    <path d={path} />
</svg>
