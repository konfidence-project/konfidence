<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import type { StatusTone } from "./status-cell.types.js";

    interface Props {
        label: string;
        tone: StatusTone;
    }
    let { label, tone }: Props = $props();
</script>

<StatusBadge status={tone}>{label}</StatusBadge>
