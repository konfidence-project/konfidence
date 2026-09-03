<script lang="ts">
    interface Props {
        /** Optional label announced to assistive technology. */
        label?: string;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let { label = "Loading", class: className }: Props = $props();

    const composedClass = $derived(
        className
            ? `flex items-center gap-3 text-sm text-[color:var(--text-secondary)] ${className}`
            : "flex items-center gap-3 text-sm text-[color:var(--text-secondary)]",
    );
</script>

<div class={composedClass} role="status" aria-live="polite">
    <span
        class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent"
        aria-hidden="true"
    ></span>
    <span class="font-[family-name:var(--font-mono)] uppercase tracking-[0.08em]">
        {label}
    </span>
</div>
