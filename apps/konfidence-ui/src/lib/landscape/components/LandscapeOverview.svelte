<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import {
        BottomSheet,
        Button,
        OrbitLoader,
        PageHeader,
        SidePanel,
    } from "@konfidence/design-system/components";
    import type { Landscape, Stage, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
    import LandscapeFlow from "#lib/landscape/components/LandscapeFlow.svelte";
    import PromotionDetails from "#lib/landscape/components/PromotionDetails.svelte";
    import StageDetails from "#lib/landscape/components/StageDetails.svelte";
    import { buildLandscapeGraph } from "#lib/landscape/landscapeGraph.ts";
    import { PROMOTION_PARAM } from "#lib/landscape/params.ts";
    import { useStageOverlay } from "#lib/landscape/stageOverlay.svelte.ts";
    import { panelDimensions } from "#lib/shell/panelDimensions.svelte.ts";
    import { createDesktopPanelViewport } from "#lib/shell/panelPresentation.svelte.ts";
    import { projectLandscapeUrl } from "#lib/projects/url.ts";

    interface Props {
        error?: string;
        status: "loading" | "ready" | "error";
        landscapes: readonly Landscape[];
        stages: readonly Stage[];
        promotionConfigs: readonly VectorPromotionConfig[];
        projectId: string;
        embedded: boolean;
        onRetry: () => void;
    }
    let {
        error,
        status,
        landscapes,
        stages,
        promotionConfigs,
        projectId,
        embedded,
        onRetry,
    }: Props = $props();

    const desktopPanelViewport = createDesktopPanelViewport();

    const overlay = useStageOverlay({
        embedded: () => embedded,
        projectId: () => projectId,
        stages: () => stages,
    });
    const selectedStage = $derived(overlay.selected);

    const graph = $derived(buildLandscapeGraph(landscapes, stages, promotionConfigs));
    const selectedPromotionId = $derived(
        selectedStage === undefined
            ? (page.url.searchParams.get(PROMOTION_PARAM) ?? undefined)
            : undefined,
    );
    const selectedPromotion = $derived(
        selectedPromotionId === undefined
            ? undefined
            : graph.edges.find((edge) => edge.id === selectedPromotionId),
    );
    // The source and target stage cards of the selected promotion are highlighted
    // in the canvas so the user sees which two stages the promotion connects.
    const highlightedStageKeys = $derived(
        selectedPromotion === undefined
            ? []
            : [selectedPromotion.sourceStageKey, selectedPromotion.targetStageKey],
    );
    const selectedPromotionConfig = $derived(
        selectedPromotion === undefined
            ? undefined
            : promotionConfigs.find((config) => config.id === selectedPromotion.id),
    );

    type PanelPlacement = "right" | "bottom";
    type CloseFocus = "trigger" | "canvas";
    interface PendingSelection {
        placement: PanelPlacement;
        run: () => void;
    }

    const activePlacement = $derived.by((): PanelPlacement | undefined => {
        if (selectedPromotion !== undefined) {
            return "bottom";
        }
        return selectedStage === undefined ? undefined : "right";
    });
    let closingPlacement: PanelPlacement | undefined = $state();
    let pendingSelection: PendingSelection | undefined = undefined;
    let lastTrigger: Element | undefined = undefined;
    let focusAfterClose: CloseFocus | undefined = undefined;
    let overlayWasOpen = false;
    let suppressStageFocusReveal = $state(false);

    const overviewUrl = (): URL =>
        new URL(projectLandscapeUrl(projectId, embedded), page.url.origin);

    const isVisibleInCanvas = (trigger: Element, canvas: HTMLElement): boolean => {
        const triggerRect = trigger.getBoundingClientRect();
        const canvasRect = canvas.getBoundingClientRect();
        return (
            triggerRect.bottom >= canvasRect.top &&
            triggerRect.top <= canvasRect.bottom &&
            triggerRect.right >= canvasRect.left &&
            triggerRect.left <= canvasRect.right
        );
    };

    const restoreTriggerFocus = (): void => {
        queueMicrotask(() => {
            const trigger = lastTrigger;
            const canvas = document.querySelector<HTMLElement>("[data-testid='landscape-flow']");
            if (
                (trigger instanceof HTMLElement || trigger instanceof SVGElement) &&
                canvas !== null &&
                trigger.isConnected &&
                isVisibleInCanvas(trigger, canvas)
            ) {
                trigger.focus({ preventScroll: true });
                return;
            }
            canvas?.focus({ preventScroll: true });
        });
    };

    const focusCanvas = (): void => {
        queueMicrotask(() =>
            document
                .querySelector<HTMLElement>("[data-testid='landscape-flow']")
                ?.focus({ preventScroll: true }),
        );
    };

    const closePromotion = (): void => {
        if (!page.url.searchParams.has(PROMOTION_PARAM)) {
            return;
        }
        if (page.state.overlayOrigin === "landscape") {
            globalThis.history.back();
            return;
        }
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- projectLandscapeUrl is already resolved.
        void goto(overviewUrl(), { replace: true, reset: false });
    };

    const closeActiveOverlay = (): void => {
        if (selectedPromotion !== undefined) {
            closePromotion();
        } else if (selectedStage !== undefined) {
            overlay.close();
        }
    };

    const dismissOverlay = (restoreFocus: boolean): void => {
        const close = (): void => {
            suppressStageFocusReveal = true;
            focusAfterClose = restoreFocus ? "trigger" : "canvas";
            closeActiveOverlay();
        };
        if (closingPlacement !== undefined) {
            pendingSelection = { placement: closingPlacement, run: close };
            return;
        }
        close();
    };

    $effect(() => {
        const overlayOpen = activePlacement !== undefined;
        if (overlayWasOpen && !overlayOpen) {
            if (focusAfterClose === "canvas") {
                focusCanvas();
            } else {
                restoreTriggerFocus();
            }
            focusAfterClose = undefined;
            queueMicrotask(() => {
                suppressStageFocusReveal = false;
            });
        }
        overlayWasOpen = overlayOpen;
    });

    const switchOverlay = (selection: PendingSelection): void => {
        if (closingPlacement !== undefined) {
            pendingSelection = selection;
            return;
        }
        if (activePlacement !== undefined && activePlacement !== selection.placement) {
            pendingSelection = selection;
            closingPlacement = activePlacement;
            return;
        }
        selection.run();
    };

    const finishPanelClose = (placement: PanelPlacement): void => {
        if (closingPlacement !== placement) {
            return;
        }
        closingPlacement = undefined;
        const selection = pendingSelection;
        pendingSelection = undefined;
        selection?.run();
    };

    // The stage (right) and promotion (bottom) panels are mutually exclusive:
    // selecting a promotion replaces any selected stage so at most one is open.
    const selectPromotion = (promotionId: string, trigger: Element): void => {
        lastTrigger = trigger;
        const replace = activePlacement !== undefined;
        switchOverlay({
            placement: "bottom",
            run: () => {
                const next = overviewUrl();
                next.searchParams.set(PROMOTION_PARAM, promotionId);
                // eslint-disable-next-line svelte/no-navigation-without-resolve -- projectLandscapeUrl is already resolved.
                void goto(next, {
                    replace,
                    reset: false,
                    state: {
                        overlayOrigin: page.state.overlayOrigin ?? "landscape",
                    },
                });
            },
        });
    };

    const selectStage = (stage: Stage, trigger: HTMLElement): void => {
        lastTrigger = trigger;
        const replace = activePlacement !== undefined;
        switchOverlay({
            placement: "right",
            run: () => overlay.open(stage, replace),
        });
    };

    $effect(() => {
        if (closingPlacement === undefined) {
            return;
        }
        // oxlint-disable-next-line unicorn/consistent-function-scoping -- closes over the current component coordinator.
        const cancelPending = (event: KeyboardEvent): void => {
            if (event.key !== "Escape") {
                return;
            }
            event.preventDefault();
            event.stopPropagation();
            dismissOverlay(true);
        };
        document.addEventListener("keydown", cancelPending, true);
        return () => document.removeEventListener("keydown", cancelPending, true);
    });
</script>

<section
    class="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]"
    aria-labelledby="landscape-title"
>
    <div class="px-6 pt-6">
        <PageHeader
            id="landscape-title"
            title="Landscape"
        />
    </div>
    {#if status === "loading"}
        <div class="place-self-center p-6">
            <OrbitLoader label="Loading landscapes and stages" />
        </div>
    {:else if status === "error"}
        <div
            class="place-self-center p-6 text-center text-content-secondary"
            role="alert"
        >
            <h2>Landscape could not be loaded</h2>
            <p>{error}</p>
            <Button onclick={onRetry}>Retry</Button>
        </div>
    {:else if landscapes.length === 0}
        <div
            class="place-self-center p-6 text-center text-content-secondary"
        >
            <h2>No landscapes yet</h2>
            <p>This project has no landscapes or stages to display.</p>
        </div>
    {:else}
        <LandscapeFlow
            {landscapes}
            {stages}
            {promotionConfigs}
            selectedStage={selectedPromotion === undefined ? selectedStage : undefined}
            stageHref={overlay.hrefFor}
            stageReference={overlay.referenceFor}
            {selectedPromotionId}
            {highlightedStageKeys}
            onPromotionSelect={selectPromotion}
            onStageSelect={selectStage}
            {suppressStageFocusReveal}
            onPaneSelect={() => {
                dismissOverlay(false);
            }}
        />
    {/if}
</section>

<SidePanel
    open={selectedStage !== undefined && selectedPromotion === undefined && closingPlacement !== "right"}
    title={selectedStage?.name ?? "Stage"}
    onClose={() => dismissOverlay(true)}
    onCloseComplete={() => finishPanelClose("right")}
    fullPageHref={overlay.selectedHref}
    width={panelDimensions.width}
    onWidthChange={panelDimensions.setWidth}
    modal={!desktopPanelViewport.current}
    focusKey={selectedStage?.id}
>
    {#if selectedStage}
        <StageDetails {projectId} stage={selectedStage} status="ready" onRetry={() => undefined} />
    {/if}
</SidePanel>

<BottomSheet
    open={selectedPromotion !== undefined && closingPlacement !== "bottom"}
    title={selectedPromotion
        ? `${selectedPromotion.sourceName} → ${selectedPromotion.targetName}`
        : "Promotion"}
    onClose={() => dismissOverlay(true)}
    onCloseComplete={() => finishPanelClose("bottom")}
    height={panelDimensions.height}
    onHeightChange={panelDimensions.setHeight}
    modal={!desktopPanelViewport.current}
    focusKey={selectedPromotion?.id}
>
    {#if selectedPromotion}
        <PromotionDetails promotion={selectedPromotion} config={selectedPromotionConfig} />
    {/if}
</BottomSheet>
