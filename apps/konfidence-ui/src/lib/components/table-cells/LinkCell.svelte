<script lang="ts">
    import type { Snippet } from "svelte";
    import { goto } from "$app/navigation";
    import type { ResolvedPathname } from "$app/types";

    interface Props {
        ariaLabel: string;
        children: Snippet;
        url: ResolvedPathname;
    }
    let { ariaLabel, children, url }: Props = $props();

    const navigate = (event: MouseEvent): void => {
        event.stopPropagation();
        void goto(url);
    };
</script>

<button
    type="button"
    class="cursor-pointer border-0 bg-transparent p-0 text-[color:var(--text-link,var(--btn-primary-fg))] underline underline-offset-[3px] focus-visible:rounded-[var(--radius-sm)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none"
    aria-label={ariaLabel}
    onclick={navigate}
>
    <span class="font-[family-name:var(--font-mono)] text-[length:var(--text-sm)]">
        {@render children()}
    </span>
</button>
