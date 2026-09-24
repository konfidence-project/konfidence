<script lang="ts">
    import { IconButton, Menu } from "@konfidence/design-system/components";
    import { isDarkMode, themeStore } from "$lib/theme";
    import type { Mode } from "$lib/theme";

    const MODE_META: Record<Mode, { icon: string; label: string }> = {
        dark: { icon: "dark-mode", label: "Dark" },
        light: { icon: "light-mode", label: "Light" },
        system: { icon: "sys-monitor", label: "System" },
    };

    const MODE_ORDER: readonly Mode[] = ["system", "light", "dark"];

    const DARK_QUERY = "(prefers-color-scheme: dark)";

    const currentMode = $derived(themeStore.mode);

    let prefersDark = $state(
        typeof globalThis.matchMedia === "function"
            ? globalThis.matchMedia(DARK_QUERY).matches
            : false,
    );

    $effect(() => {
        if (typeof globalThis.matchMedia !== "function") {
            return;
        }
        const media = globalThis.matchMedia(DARK_QUERY);
        const onChange = (event: MediaQueryListEvent): void => {
            prefersDark = event.matches;
        };
        prefersDark = media.matches;
        media.addEventListener("change", onChange);
        return () => media.removeEventListener("change", onChange);
    });

    const triggerIcon = $derived(
        // The navbar button always shows the resolved appearance (light or dark).
        isDarkMode(currentMode, prefersDark) ? MODE_META.dark.icon : MODE_META.light.icon,
    );

    const handleSelect = (details: { value: string }): void => {
        const next = details.value as Mode;
        if (next in MODE_META) {
            themeStore.setMode(next);
        }
    };
</script>

<Menu positioning={{ placement: "bottom-end" }} onSelect={handleSelect}>
    <Menu.Trigger>
        {#snippet element(attributes)}
            <IconButton
                {...attributes}
                icon={triggerIcon}
                ariaLabel="Change theme"
                class={(attributes.class as string | undefined) ?? undefined}
                data-testid="theme-switcher-trigger"
            />
        {/snippet}
    </Menu.Trigger>
    <Menu.Positioner>
        <Menu.Content>
            <Menu.ItemGroup>
                <Menu.Label>Theme</Menu.Label>
                {#each MODE_ORDER as mode (mode)}
                    <Menu.Item
                        value={mode}
                        active={mode === currentMode}
                        data-testid={`theme-option-${mode}`}
                    >
                        <ui5-icon class="size-4 text-current" name={MODE_META[mode].icon}
                        ></ui5-icon>
                        {MODE_META[mode].label}
                    </Menu.Item>
                {/each}
            </Menu.ItemGroup>
        </Menu.Content>
    </Menu.Positioner>
</Menu>
