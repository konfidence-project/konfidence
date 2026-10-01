<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ColorMode } from "./types.js";
    import IconButton from "../IconButton/IconButton.svelte";
    import { Menu } from "../Menu/index.js";

    interface Props {
        value: ColorMode;
        onValueChange: (value: ColorMode) => void;
        /** The current appearance, for the trigger icon when value is system. */
        resolvedMode?: "light" | "dark";
    }

    let { value, onValueChange, resolvedMode }: Props = $props();

    const OPTIONS: { value: ColorMode; label: string; icon: string }[] = [
        { icon: "sys-monitor", label: "System", value: "system" },
        { icon: "light-mode", label: "Light", value: "light" },
        { icon: "dark-mode", label: "Dark", value: "dark" },
    ];

    const triggerIcon = $derived(resolvedMode === "dark" || (resolvedMode === undefined && value === "dark") ? "dark-mode" : "light-mode");
</script>

<Menu positioning={{ placement: "bottom-end" }}>
    <Menu.Trigger>
        {#snippet element(attributes)}
            <IconButton
                {...attributes}
                icon={triggerIcon}
                ariaLabel="Change color mode"
                class={(attributes.class as string | undefined) ?? undefined}
                data-testid="color-mode-trigger"
            />
        {/snippet}
    </Menu.Trigger>
    <Menu.Positioner>
        <Menu.Content>
            <SkMenu.ItemGroup>
                <SkMenu.ItemGroupLabel class="menu__label">Color mode</SkMenu.ItemGroupLabel>
                {#each OPTIONS as option (option.value)}
                    <SkMenu.OptionItem
                        type="radio"
                        value={option.value}
                        checked={value === option.value}
                        onCheckedChange={(checked) => { if (checked) onValueChange(option.value); }}
                        class="menu__item flex w-full cursor-pointer items-center gap-2.5 rounded-[var(--radius-sm)] border-none bg-transparent px-2.5 py-2 text-left text-[length:var(--text-sm)] text-[var(--text-primary)] hover:bg-[var(--surface-sunken)] data-highlighted:bg-[var(--surface-sunken)] aria-checked:bg-[var(--selection-bg)] aria-checked:text-[var(--selection-fg)]"
                    >
                        <ui5-icon class="size-4 text-current" name={option.icon}></ui5-icon>
                        {option.label}
                        <SkMenu.ItemIndicator class="ml-auto" aria-hidden="true">✓</SkMenu.ItemIndicator>
                    </SkMenu.OptionItem>
                {/each}
            </SkMenu.ItemGroup>
        </Menu.Content>
    </Menu.Positioner>
</Menu>
