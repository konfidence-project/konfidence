<script lang="ts">
    import "@ui5/webcomponents/dist/Card.js";
    import "@ui5/webcomponents/dist/CardHeader.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents/dist/List.js";
    import "@ui5/webcomponents/dist/ListItemStandard.js";
    import "@ui5/webcomponents-icons/dist/accept.js";
    import "@ui5/webcomponents-icons/dist/error.js";
    import "@ui5/webcomponents-icons/dist/pending.js";
    import "@ui5/webcomponents-icons/dist/upstacked-chart.js";

    import type { ConditionStatus, Stage } from "$lib/stages.js";
    import StageStatusPill from "$lib/components/StageStatusPill.svelte";

    const { stage } = $props<{ stage: Stage }>();

    const ready = $derived(
        stage.status.conditions?.find(
            (condition: NonNullable<Stage["status"]["conditions"]>[number]) =>
                condition.type === "Ready",
        ),
    );

    const conditionIcon = (status: ConditionStatus) => {
        if (status === "True") return "accept";
        if (status === "False") return "error";
        return "pending";
    };

    const formatDateTime = (value: string) =>
        new Intl.DateTimeFormat(undefined, {
            dateStyle: "medium",
            timeStyle: "short",
        }).format(new Date(value));
</script>

<ui5-card class="stage-card">
    <ui5-card-header
        slot="header"
        title-text={stage.metadata.name}
        subtitle-text={stage.metadata.namespace}
        interactive
    >
        <ui5-icon slot="avatar" name="upstacked-chart"></ui5-icon>
        {#if ready}
            <span slot="action">
                <StageStatusPill status={ready.status} />
            </span>
        {/if}
    </ui5-card-header>

    <div class="card-body">
        <dl class="stage-facts">
            <div>
                <dt>Vector</dt>
                <dd>{stage.spec.vector}</dd>
            </div>
            <div>
                <dt>Generation</dt>
                <dd>{stage.metadata.generation}</dd>
            </div>
            <div>
                <dt>Created</dt>
                <dd>{formatDateTime(stage.metadata.creationTimestamp)}</dd>
            </div>
            {#if stage.status.latestVectorDeploymentRef}
                <div>
                    <dt>Latest deployment</dt>
                    <dd>{stage.status.latestVectorDeploymentRef.name}</dd>
                </div>
            {/if}
        </dl>

        <div class="section-title">Conditions</div>
        <ui5-list separators="Inner" accessible-name={`${stage.metadata.name} conditions`}>
            {#each stage.status.conditions ?? [] as condition (condition.type)}
                <ui5-li icon={conditionIcon(condition.status)}>
                    <span class="condition-line">
                        <span>{condition.type}</span>
                        <StageStatusPill status={condition.status} />
                    </span>
                    <span slot="description">{condition.reason}: {condition.message}</span>
                </ui5-li>
            {/each}
        </ui5-list>

        {#if stage.status.vectorHistory?.length}
            <div class="section-title">Vector history</div>
            <div class="history-list">
                {#each stage.status.vectorHistory as vector (vector)}
                    <code>{vector}</code>
                {/each}
            </div>
        {/if}
    </div>
</ui5-card>

<style>
    .stage-card {
        min-width: 0;
    }

    .card-body {
        display: grid;
        gap: 1rem;
        padding: 1rem;
    }

    .stage-facts {
        display: grid;
        gap: 0.75rem;
        margin: 0;
    }

    .stage-facts div {
        min-width: 0;
    }

    dt,
    .section-title {
        color: var(--sapContent_LabelColor);
        font-size: var(--sapFontSmallSize);
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.04em;
    }

    dd {
        margin: 0.125rem 0 0;
        overflow-wrap: anywhere;
    }

    .condition-line {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        width: 100%;
    }

    .history-list {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
    }

    code {
        padding: 0.25rem 0.5rem;
        border: 1px solid var(--sapList_BorderColor);
        border-radius: 0.375rem;
        background: var(--sapList_Background);
        overflow-wrap: anywhere;
    }
</style>
