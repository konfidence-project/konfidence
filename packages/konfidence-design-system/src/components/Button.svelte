<script lang="ts">
    import type { HTMLAnchorAttributes, HTMLButtonAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";

    type Variant = "primary" | "secondary" | "ghost" | "danger";

    interface CommonProps {
        /** Visual variant. Only one primary button per context. */
        variant?: Variant;
        /** Optional extra class names appended after the variant class. */
        class?: string;
        /** Button label. */
        children?: Snippet;
    }

    type ButtonProps = CommonProps & {
        href?: undefined;
    } & Omit<HTMLButtonAttributes, "class" | "children">;

    type AnchorProps = CommonProps & {
        href: string;
    } & Omit<HTMLAnchorAttributes, "class" | "children" | "href">;

    type Props = ButtonProps | AnchorProps;

    let { variant = "primary", class: className, children, ...rest }: Props = $props();

    const VARIANT_CLASS: Record<Variant, string> = {
        danger: "btn btn--danger",
        ghost: "btn btn--ghost",
        primary: "btn btn--primary",
        secondary: "btn btn--secondary",
    };

    const composedClass = $derived(
        className ? `${VARIANT_CLASS[variant]} ${className}` : VARIANT_CLASS[variant],
    );
</script>

{#if "href" in rest && rest.href !== undefined}
    <a class={composedClass} {...rest as HTMLAnchorAttributes}>
        {@render children?.()}
    </a>
{:else}
    <button class={composedClass} type={(rest as HTMLButtonAttributes).type ?? "button"} {...rest as HTMLButtonAttributes}>
        {@render children?.()}
    </button>
{/if}

<style>
    .btn {
        position: relative;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 7px;
        font-family: inherit;
        font-size: var(--text-sm);
        font-weight: var(--weight-semibold);
        text-decoration: none;
        padding: var(--control-py) var(--btn-px);
        border-radius: var(--btn-radius);
        border: 1px solid transparent;
        cursor: pointer;
        line-height: 1.2;
        overflow: hidden;
        white-space: nowrap;
        transition:
            transform var(--motion-fast) var(--ease),
            box-shadow var(--motion-base) var(--ease),
            background var(--motion-fast) var(--ease),
            border-color var(--motion-fast) var(--ease);
    }

    .btn :global(svg) {
        width: var(--icon-md);
        height: var(--icon-md);
        position: relative;
        z-index: 1;
    }

    .btn > :global(span) {
        position: relative;
        z-index: 1;
    }

    /* Primary — brand amber gradient */
    .btn--primary {
        background: var(--btn-primary-bg);
        color: var(--btn-primary-fg);
        box-shadow: var(--btn-primary-shadow);
    }

    .btn--primary:hover {
        transform: translateY(-1px);
        box-shadow:
            0 6px 22px rgba(255, 149, 12, 0.45),
            0 0 0 1px rgba(255, 149, 12, 0.2),
            0 0 24px rgba(255, 181, 48, 0.35);
    }

    /* Secondary */
    .btn--secondary {
        background: var(--btn-secondary-bg);
        color: var(--btn-secondary-fg);
        border-color: var(--btn-secondary-bd);
        box-shadow: var(--shadow-xs);
    }

    .btn--secondary:hover {
        background: var(--surface-subtle);
        border-color: var(--border-strong);
    }

    /* Ghost */
    .btn--ghost {
        background: transparent;
        color: var(--text-secondary);
    }

    .btn--ghost:hover {
        background: var(--surface-sunken);
        color: var(--text-primary);
    }

    /* Danger — neutral until hover */
    .btn--danger {
        background: var(--surface-default);
        color: var(--btn-danger-fg);
        border-color: var(--border-default);
        box-shadow: var(--shadow-xs);
    }

    .btn--danger:hover {
        background: var(--btn-danger-bg);
    }

    .btn:disabled {
        opacity: var(--disabled-opacity);
        cursor: not-allowed;
        box-shadow: none;
        transform: none;
    }

    /* Neutralise Skeleton's hover `filter: brightness()` so it doesn't
       stack on our gradient. */
    @media (hover: hover) {
        .btn:not(:disabled):hover {
            filter: none;
        }
    }
</style>
