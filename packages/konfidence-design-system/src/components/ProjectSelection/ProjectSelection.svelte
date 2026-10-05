<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    interface Props {
        name: string;
        class?: unknown;
        [key: string]: unknown;
    }

    let { name, class: className, ...rest }: Props = $props();
</script>

<button type="button" class={[
    "project-switch flex cursor-pointer items-center gap-2 rounded-base border border-outline-subtle bg-surface-subtle px-2.5 py-1.5 text-compact font-semibold text-content-primary hover:bg-surface-sunken focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-outline-focus",
    typeof className === "string" && className.length > 0 && className,
]} aria-label="Change project" {...rest}>
    <span class="project-switch__name min-w-0 flex-1 truncate text-left">{name}</span>
    <ui5-icon class="size-3.5 text-current" name="slim-arrow-down"></ui5-icon>
</button>
