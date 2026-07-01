<script lang="ts">
    import type { ConditionStatus } from "$lib/stages.js";

    const { status, label = status } = $props<{
        status: ConditionStatus | "Info";
        label?: string;
    }>();
</script>

<span class={["status-pill", `status-${status.toLowerCase()}`]}>{label}</span>

<style>
    .status-pill {
        display: inline-flex;
        align-items: center;
        width: fit-content;
        padding: 0.125rem 0.5rem;
        border-radius: 999px;
        font-size: var(--sapFontSmallSize);
        font-weight: 700;
        line-height: 1.25rem;
        white-space: nowrap;
    }

    .status-info {
        color: var(--sapInformativeTextColor);
        background: var(--sapInformationBackground);
    }

    .status-true {
        color: var(--sapPositiveTextColor);
        background: var(--sapSuccessBackground);
    }

    .status-false {
        color: var(--sapNegativeTextColor);
        background: var(--sapErrorBackground);
    }

    .status-unknown {
        color: var(--sapCriticalTextColor);
        background: var(--sapWarningBackground);
    }
</style>
