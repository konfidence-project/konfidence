<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";

    import type { ArtifactDeploymentRow } from "../deployments.js";
    import { statusLabel, statusTone } from "../deployments.js";

    interface Props {
        status: ArtifactDeploymentRow["status"];
    }
    let { status }: Props = $props();
</script>

<StatusBadge status={statusTone(status)}>{statusLabel(status)}</StatusBadge>
