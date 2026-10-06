<script lang="ts">
    // oxlint-disable-next-line import/no-unassigned-import
    import "../app.css";
    import type { Snippet } from "svelte";
    import { Brandbar } from "@konfidence/design-system/components";
    import BrandLogo from "@konfidence/design-system/components/BrandLogo";

    let { children }: { children: Snippet } = $props();
</script>

<div class="flex min-h-dvh flex-col bg-[image:var(--gradient-hero-bg)]">
    <Brandbar />
    <main class="flex flex-1 flex-col items-center justify-center p-8 text-center">
        <BrandLogo class="mb-8 w-[28rem]" />
        {@render children()}
    </main>
</div>
