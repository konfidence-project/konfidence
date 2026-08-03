<script lang="ts">
    import ErrorView from "$lib/components/ErrorView.svelte";
    import { HTTP_FORBIDDEN } from "$lib/http-status";
    import { page } from "$app/state";
    import { t } from "$lib/stores/i18n.svelte";

    const errorCode = $derived.by(() => {
        const err = page.error;
        if (err && typeof err === "object" && "code" in err) {
            const { code } = err as { code?: unknown };
            if (typeof code === "string") {
                return code;
            }
        }
        return undefined;
    });

    const title = $derived.by(() => {
        if (page.status === HTTP_FORBIDDEN) {
            return t("ERROR_ACCESS_DENIED_TITLE");
        }
        return t("ERROR_APP_UNAVAILABLE_TITLE");
    });
    const message = $derived.by(() => {
        if (errorCode) {
            return t(`ERROR_CODE_${errorCode}`);
        }
        if (page.status === HTTP_FORBIDDEN) {
            return t("ERROR_ACCESS_DENIED_MESSAGE");
        }
        return t("ERROR_APP_UNAVAILABLE_MESSAGE");
    });
</script>

<ErrorView
    {title}
    status={page.status}
    error={page.error}
    {message}
/>
