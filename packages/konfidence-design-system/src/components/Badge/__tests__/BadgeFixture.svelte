<script lang="ts">
    import Badge from "../Badge.svelte";
    import type { TagColor } from "../../Tag/types.js";

    interface Props { size?: "sm" | "md"; status?: string; animated?: boolean; showDot?: boolean; variant?: "filled" | "subtle"; color?: TagColor; }
    let { status = "waiting", ...rest }: Props = $props();
</script>
<Badge {status} {...rest}>State</Badge>
