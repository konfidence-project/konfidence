<script lang="ts">
    import ColorModeSelect from "../ColorModeSelect.svelte";
    import type { ColorMode } from "../types.js";

    let value = $state<ColorMode>("system");
    let resolvedMode = $state<"light" | "dark">("light");
</script>

<ColorModeSelect {value} {resolvedMode} onValueChange={(next) => value = next} />
<output data-testid="selected-mode">{value}</output>
<button onclick={() => resolvedMode = "dark"}>Resolve dark</button>
