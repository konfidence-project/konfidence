<script lang="ts">
    import { Tag } from '@konfidence/design-system/components';
import type { TagColor } from '@konfidence/design-system/components';
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    const examples: { color: TagColor; label: string; icon: string }[] = [
        { color: "teal", icon: "navigation-right-arrow", label: "Promotion" },
        { color: "amber", icon: "list", label: "Migration" },
        { color: "green", icon: "accept", label: "Test" },
        { color: "blue", icon: "upload-to-cloud", label: "Activation" },
        { color: "slate", icon: "key", label: "Deploy" },
        { color: "violet", icon: "switch-classes", label: "Feature Toggle" },
        { color: "red", icon: "refresh", label: "Rollback" },
        { color: "orange", icon: "alert", label: "Incident" },
        { color: "lime", icon: "alert", label: "Build" },
        { color: "pink", icon: "accept", label: "Approval" },
        { color: "cyan", icon: "search", label: "Scan" },
        { color: "indigo", icon: "action-settings", label: "Config" },
    ];
    let feedback = $state("Select a tag to try the interaction.");
    const code = `<Tag color="teal" icon="navigation-right-arrow">Promotion</Tag>
<Tag mono size="sm">US-30</Tag>
<Tag color="teal" size="md" interactive onclick={openLatest}>Latest waiting: v2.14.0</Tag>
<Tag color="teal" href="/promotions">Promotions</Tag>`;
</script>

<span id="foundation-tags" class="scroll-mt-24"></span>
<Section id="tags" title="Tags" subtitle="Tags primarily display metadata, such as categories, IDs and version details. Twelve colors, optional icons and monospace text, with size and interaction as independent options.">
    <Sample>
        <div class="flex w-full flex-col gap-5">
            <div class="flex flex-wrap items-center gap-3" data-tag-palettes>
                {#each examples as example (example.color)}
                    <Tag color={example.color} icon={example.icon}>{example.label}</Tag>
                {/each}
            </div>
            <div class="border-t border-outline-subtle pt-4">
                <p class="mb-3 text-compact font-semibold">Icon and content</p>
                <div class="flex flex-wrap items-center gap-3">
                    <Tag color="teal" icon="navigation-right-arrow">Promotion</Tag>
                    <Tag color="teal">Promotion</Tag>
                    <Tag mono>US-30</Tag>
                    <Tag mono>sha:9f3c1ba</Tag>
                </div>
            </div>
            <div class="border-t border-outline-subtle pt-4" id="tag-sizes">
                <p class="mb-3 text-compact"><strong>Sizes</strong> — S: 20px · M: 24px (default), both with 12px text.</p>
                <div class="flex flex-col gap-3">
                    <div class="flex flex-wrap items-center gap-3">
                        <span class="w-4 text-meta text-content-secondary">S</span>
                        <Tag mono size="sm">US-30</Tag>
                        <Tag color="teal" size="sm" icon="navigation-right-arrow">Promotion</Tag>
                    </div>
                    <div class="flex flex-wrap items-center gap-3">
                        <span class="w-4 text-meta text-content-secondary">M</span>
                        <Tag mono>US-30</Tag>
                        <Tag color="teal" icon="navigation-right-arrow">Promotion</Tag>
                    </div>
                </div>
            </div>
            <div class="border-t border-outline-subtle pt-4">
                <p class="mb-3 text-compact"><strong>Interaction (optional)</strong> — Hover, pressed and keyboard focus states; buttons for actions, links for navigation.</p>
                <div class="flex flex-wrap items-center gap-3">
                    <Tag mono size="sm" interactive onclick={() => feedback = "Region US-30 selected"}>US-30</Tag>
                    <Tag color="teal" interactive wrap onclick={() => feedback = "Latest waiting promotion v2.14.0 opened"}>Latest waiting: <span class="font-mono">v2.14.0</span> · 5 min ago</Tag>
                    <Tag color="teal" icon="navigation-right-arrow" interactive onclick={() => feedback = "Promotion selected"}>Promotion</Tag>
                    <Tag color="teal" interactive disabled>Latest waiting: unavailable</Tag>
                    <Tag color="teal" size="sm" href="#tag-sizes">View sizes</Tag>
                </div>
                <p class="mt-3 text-meta text-content-secondary"><output aria-live="polite" data-tag-feedback>{feedback}</output></p>
            </div>
        </div>
    </Sample>
    <pre class="mt-4 overflow-x-auto rounded-base bg-surface-sunken p-4 text-meta"><code>{code}</code></pre>
</Section>
