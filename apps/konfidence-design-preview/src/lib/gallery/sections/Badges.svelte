<script lang="ts">
    import { Badge } from '@konfidence/design-system/components';
import type { TagColor } from '@konfidence/design-system/components';
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    const colors: TagColor[] = ["red", "orange", "amber", "lime", "green", "teal", "cyan", "blue", "violet", "indigo", "pink", "slate"];
    const code = `<Badge status="green" color="green" showDot={false}>Green</Badge>
<Badge status="green" color="green" variant="subtle" showDot={false}>Green</Badge>
<Badge status="green" color="green" size="md" showDot={false}>Green</Badge>
<Badge status="waiting" animated>Waiting</Badge>
<Badge status="active" showDot={false}>In Progress</Badge>`;
</script>

<span id="status-badges" class="scroll-mt-24"></span>
<Section id="badges" title="Badges" subtitle="Badges display status information using color, a label and an optional dot.">
    <Sample>
        <div class="flex w-full flex-col gap-5">
            <div>
                <p class="mb-3 text-compact"><strong>Appearance</strong> — filled or subtle with a colored border. Every status and all twelve colors support both variants.</p>
                <p class="mb-3 text-meta text-content-secondary">Filled</p>
                <div class="mb-4 flex flex-wrap items-center gap-3" data-filled-palettes>
                    {#each colors as color (color)}
                        <Badge status={color} {color} showDot={false}>{color.charAt(0).toUpperCase() + color.slice(1)}</Badge>
                    {/each}
                </div>
                <p class="mb-3 text-meta text-content-secondary">Subtle</p>
                <div class="flex flex-wrap items-center gap-3" data-subtle-palettes>
                    {#each colors as color (color)}
                        <Badge status={color} {color} variant="subtle" showDot={false}>{color.charAt(0).toUpperCase() + color.slice(1)}</Badge>
                    {/each}
                </div>
            </div>
            <div class="border-t border-outline-subtle pt-4">
                <p class="mb-3 text-compact"><strong>Sizes</strong> — S (default) or M. Both sizes support every color, appearance and dot option.</p>
                <div class="flex flex-col gap-3" data-badge-sizes>
                    <div class="flex flex-wrap items-center gap-3">
                        <span class="w-4 text-meta text-content-secondary">S</span>
                        <Badge status="green" color="green" size="sm" showDot={false}>Green</Badge>
                        <Badge status="green" color="green" size="sm" variant="subtle" showDot={false}>Green</Badge>
                    </div>
                    <div class="flex flex-wrap items-center gap-3">
                        <span class="w-4 text-meta text-content-secondary">M</span>
                        <Badge status="green" color="green" size="md" showDot={false}>Green</Badge>
                        <Badge status="green" color="green" size="md" variant="subtle" showDot={false}>Green</Badge>
                    </div>
                </div>
            </div>
            <div class="border-t border-outline-subtle pt-4">
                <p class="mb-3 text-compact"><strong>Dot (optional)</strong> — no dot, static or animated. Independent of status and appearance.</p>
                <div class="flex flex-wrap items-center gap-3" data-badge-dot-options>
                    <Badge status="waiting" showDot={false}>No dot</Badge>
                    <Badge status="waiting">Static</Badge>
                    <Badge status="waiting" animated>Animated</Badge>
                    <Badge status="healthy" variant="subtle" animated>Healthy</Badge>
                </div>
                <p class="mt-3 text-meta text-content-secondary">Without animated, the dot stays static, including for In Progress. Reduced motion disables the animation.</p>
            </div>
        </div>
    </Sample>
    <pre class="mt-4 overflow-x-auto rounded-base bg-surface-sunken p-4 text-meta"><code>{code}</code></pre>
</Section>
