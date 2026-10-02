<script lang="ts">
    import type { Snippet } from "svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { Link } from "@konfidence/design-system/components";

    interface Props {
        ariaLabel: string;
        children: Snippet;
        url: string;
    }
    let { ariaLabel, children, url }: Props = $props();

    const external = $derived(new globalThis.URL(url, page.url).origin !== page.url.origin);

    const stopRowSelection = (event: MouseEvent): void => {
        event.stopPropagation();
    };

    const navigate = (event: MouseEvent): void => {
        stopRowSelection(event);
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- internal URLs are resolved by the call site before they reach this generic cell.
        void goto(url);
    };
</script>

{#if external}
    <Link
        href={url}
        class="font-[family-name:var(--font-mono)] text-[length:var(--text-sm)]"
        aria-label={ariaLabel}
        onclick={stopRowSelection}
    >
        {@render children()}
    </Link>
{:else}
    <Link
        class="font-[family-name:var(--font-mono)] text-[length:var(--text-sm)]"
        aria-label={ariaLabel}
        onclick={navigate}
    >
        {@render children()}
    </Link>
{/if}
