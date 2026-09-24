<script lang="ts">
    import LinkCell from "../LinkCell.svelte";

    interface Props {
        onselect: () => void;
        url?: string;
    }

    let { onselect, url = "/projects" }: Props = $props();
</script>

<table>
    <tbody>
        <tr tabindex="0" onclick={onselect} onkeydown={() => undefined}>
            <td><LinkCell {url} ariaLabel="View related deployments">3</LinkCell></td>
        </tr>
    </tbody>
</table>