<script lang="ts">
    import type { Snippet } from "svelte";
    import type { HTMLAnchorAttributes, HTMLButtonAttributes } from "svelte/elements";

    interface CommonProps {
        /** Optional extra class names appended after the base class. */
        class?: string;
        /** Link label. */
        children?: Snippet;
    }

    type ButtonProps = CommonProps & {
        href?: undefined;
    } & Omit<HTMLButtonAttributes, "class" | "children">;

    type AnchorProps = CommonProps & {
        href: string;
    } & Omit<HTMLAnchorAttributes, "class" | "children" | "href">;

    type Props = ButtonProps | AnchorProps;

    let { class: className, children, ...rest }: Props = $props();

    const baseClass =
        "link inline m-0 p-0 border-0 bg-transparent [font:inherit] [line-height:inherit] cursor-pointer underline underline-offset-[3px] text-[color:var(--text-link)] focus-visible:rounded-[var(--radius-sm)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none disabled:text-[color:var(--text-disabled)] disabled:cursor-not-allowed";

    const composedClass = $derived(className ? `${baseClass} ${className}` : baseClass);
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