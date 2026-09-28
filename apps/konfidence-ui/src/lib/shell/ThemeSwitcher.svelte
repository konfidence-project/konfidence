<script lang="ts">
    import { ColorModeSelect } from "@konfidence/design-system/components";
    import { mode, setMode, userPrefersMode } from "mode-watcher";
</script>

<ColorModeSelect value={userPrefersMode.current} resolvedMode={mode.current} onValueChange={setMode} />
