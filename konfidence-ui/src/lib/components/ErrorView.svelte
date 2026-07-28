<script lang="ts">
    import errorBurstUrl from "$lib/assets/gpt-image-2-kn700szf3pnqy26t64gtqqgz0s89tnyh-removebg-preview.png";
    import { resolve } from "$app/paths";

    const {
        title = "Something went wrong",
        message = "The requested content could not be loaded.",
        error,
        status,
    } = $props<{
        title?: string;
        message?: string;
        error?: unknown;
        status?: number;
    }>();

    const detail = $derived.by(() => {
        if (error instanceof Error) {
            return error.message;
        }

        if (error && typeof error === "object" && "message" in error) {
            return String(error.message);
        }

        return undefined;
    });
</script>

<section class="error-view grid min-h-[min(44rem,calc(100vh-4rem))] place-items-center content-center gap-[clamp(0.75rem,2.5vw,1.5rem)] p-[clamp(1.5rem,6vw,4rem)] max-[42rem]:p-4" role="alert" aria-live="assertive" aria-labelledby="error-view-title">
    <div class="supernova relative mb-[-1.5rem] grid aspect-square w-[min(64vw,34rem)] min-w-64 place-items-center max-[42rem]:mb-[-0.75rem] max-[42rem]:w-[min(78vw,22rem)]" aria-hidden="true">
        <span class="aura pointer-events-none absolute aspect-square w-[68%] animate-supernova-pulse rounded-full bg-[radial-gradient(circle,color-mix(in_srgb,var(--app-warning)_34%,transparent)_0_34%,transparent_67%)] opacity-90 blur-[1.6rem]"></span>
        <span class="orbit-outer pointer-events-none absolute h-[44%] w-[78%] animate-orbit-drift rounded-full border border-app-warning/36 [transform:rotate(-16deg)_skew(-8deg)]"></span>
        <span class="pointer-events-none absolute h-[30%] w-[52%] animate-orbit-drift-reverse rounded-full border border-app-warning/42 [transform:rotate(22deg)_skew(10deg)]"></span>
        <span class="pointer-events-none absolute top-[18%] left-[18%] size-[0.45rem] animate-particle-drift rounded-full bg-app-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--app-warning)_70%,transparent)]"></span>
        <span class="pointer-events-none absolute top-[31%] right-[12%] size-[0.32rem] animate-particle-drift rounded-full bg-app-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--app-warning)_70%,transparent)] [animation-delay:-1.2s]"></span>
        <span class="pointer-events-none absolute right-[22%] bottom-[20%] size-[0.56rem] animate-particle-drift rounded-full bg-app-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--app-warning)_70%,transparent)] [animation-delay:-2.4s]"></span>
        <span class="pointer-events-none absolute bottom-[24%] left-[14%] size-[0.28rem] animate-particle-drift rounded-full bg-app-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--app-warning)_70%,transparent)] [animation-delay:-3s]"></span>
        <span class="pointer-events-none absolute top-[11%] left-[64%] size-[0.24rem] animate-particle-drift rounded-full bg-app-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--app-warning)_70%,transparent)] [animation-delay:-1.8s]"></span>
        <img class="error-art relative h-auto w-[78%] animate-art-float drop-shadow-xl" src={errorBurstUrl} alt="" />
    </div>

    <div class="content-card grid w-[min(100%,42rem)] gap-4 rounded-3xl border border-app-border/78 bg-app-card/88 p-[clamp(1.25rem,4vw,2.25rem)] shadow-2xl backdrop-blur-xl max-[42rem]:rounded-[1.125rem]">
        {#if status}
            <span class="w-max rounded-full bg-app-warning/18 px-[0.625rem] py-1 text-xs font-bold tracking-[0.08em] text-app-warning uppercase">Error {status}</span>
        {/if}

        <h1 id="error-view-title" class="m-0">{title}</h1>
        <p class="m-0 text-base leading-[1.55] text-app-muted">{message}</p>

        {#if detail}
            <pre class="m-0 max-h-32 overflow-auto whitespace-pre-wrap rounded-[0.875rem] border border-app-border bg-app-bg/72 px-4 py-[0.875rem] font-mono text-sm leading-[1.45] text-app-error">{detail}</pre>
        {/if}

        <div class="flex flex-wrap items-center gap-3 pt-1">
            <a class="action-link btn preset-filled-primary-500 min-h-10 px-4 font-semibold" href={resolve("/")}>Back to start</a>
        </div>
    </div>
</section>
