<script lang="ts">
    import BoxesIcon from "@lucide/svelte/icons/boxes";
    import NetworkIcon from "@lucide/svelte/icons/network";

    interface NavItem {
        href: string;
        icon: "landscape" | "vector";
        text: string;
    }

    const { collapsed = false, currentPath, items, onNavigate } = $props<{
        collapsed?: boolean;
        currentPath: string;
        items: readonly NavItem[];
        onNavigate?: () => void;
    }>();
</script>

<nav class="px-2 py-3" aria-label="Delivery">
    <h2
        class={[
            "px-3 py-2 text-xs font-semibold tracking-[0.05em] text-muted-foreground uppercase",
            collapsed && "min-[48rem]:hidden",
        ]}
    >
        Delivery
    </h2>
    <ul class="grid list-none gap-1 p-0">
        {#each items as item (item.href)}
            <li class={collapsed ? "min-[48rem]:relative min-[48rem]:min-h-11" : undefined}>
                <a
                    class={[
                        "group flex min-h-11 items-center gap-[0.7rem] rounded-[0.55rem] border-l-[3px] border-transparent px-3 py-2.5 text-sm text-sidebar-foreground no-underline hover:bg-sidebar-accent/70 [&_svg]:w-[1.05rem] [&_svg]:shrink-0",
                        collapsed &&
                            "min-[48rem]:absolute min-[48rem]:inset-y-0 min-[48rem]:right-0 min-[48rem]:left-0 min-[48rem]:justify-center min-[48rem]:px-0 min-[48rem]:hover:right-auto min-[48rem]:hover:z-10 min-[48rem]:hover:w-max min-[48rem]:hover:bg-sidebar min-[48rem]:hover:pr-3 min-[48rem]:hover:pl-4 min-[48rem]:hover:shadow-md",
                        currentPath.startsWith(item.href) &&
                            "border-l-sidebar-primary bg-sidebar-accent text-sidebar-accent-foreground",
                    ]}
                    href={item.href}
                    aria-current={currentPath.startsWith(item.href) ? "page" : undefined}
                    aria-label={item.text}
                    onclick={onNavigate}
                >
                    {#if item.icon === "landscape"}
                        <NetworkIcon aria-hidden="true" />
                    {:else}
                        <BoxesIcon aria-hidden="true" />
                    {/if}
                    <span
                        class={collapsed ? "min-[48rem]:hidden min-[48rem]:group-hover:inline" : undefined}
                        >{item.text}</span
                    >
                </a>
            </li>
        {/each}
    </ul>
</nav>
