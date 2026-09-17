<script lang="ts">
    import { Button, SidePanel } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let panelOpen = $state(false);
    let footerOpen = $state(false);
</script>

<Section
    id="overlays"
    title="Dialogs & side panels"
    subtitle="Real <SidePanel> from @konfidence/design-system (right-anchored `<dialog>` with backdrop). Modal dialogs and master-detail with resizer are still on the roadmap."
    status="partial"
>
    <div class="stack">
        <Sample
            title="SidePanel — details drawer"
            description="Right-anchored, modal, dismissable via backdrop, ESC, or the close button."
        >
            <Button variant="primary" onclick={() => (panelOpen = true)}>Open details</Button>
            <SidePanel bind:open={panelOpen} title="Deployment d-001">
                <p class="paragraph">
                    Konfidence's <code>&lt;SidePanel&gt;</code> renders a native
                    <code>&lt;dialog&gt;</code> element opened via
                    <code>showModal()</code>, so the browser handles the focus trap,
                    <kbd>Esc</kbd> to close, and the tinted backdrop.
                </p>
                <p class="paragraph">
                    Pass a <code>footer</code> snippet for confirm/dismiss actions.
                </p>
            </SidePanel>
        </Sample>

        <Sample
            title="SidePanel — with footer"
            description="Footer snippet is anchored to the bottom of the panel."
        >
            <Button variant="secondary" onclick={() => (footerOpen = true)}>Open with footer</Button>
            <SidePanel bind:open={footerOpen} title="Confirm deployment">
                <p class="paragraph">
                    You are about to promote <b>kden-api v1.5.0-rc.1</b> to <b>staging</b>.
                    This is a demo dialog — nothing will actually change.
                </p>
                {#snippet footer()}
                    <div class="footer">
                        <Button variant="ghost" onclick={() => (footerOpen = false)}>Cancel</Button>
                        <Button variant="primary" onclick={() => (footerOpen = false)}>Promote</Button>
                    </div>
                {/snippet}
            </SidePanel>
        </Sample>

        <NotYetImplemented
            note="Skeleton v5 provides Dialog primitives. Konfidence has not yet wrapped them."
            planned={["<Dialog> / <ConfirmDialog>", "<SidePanel side='bottom'>", "<MasterDetail> with draggable resizer"]}
        />
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .paragraph {
        margin: 0 0 12px;
        color: var(--text-secondary);
        font-size: var(--text-sm);
        line-height: 1.5;
    }

    .paragraph code,
    .paragraph kbd {
        font-family: var(--font-mono);
        font-size: 0.92em;
        padding: 1px 5px;
        border-radius: 4px;
        background: var(--surface-subtle);
        color: var(--text-primary);
    }

    .footer {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
    }
</style>
