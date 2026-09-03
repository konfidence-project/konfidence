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
