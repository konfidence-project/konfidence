<script lang="ts">
    import { buttonVariants } from "$lib/components/ui/button/index.js";
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

<section
    class="box-border grid min-h-[min(44rem,calc(100vh-4rem))] place-items-center content-center gap-[clamp(0.75rem,2.5vw,1.5rem)] p-[clamp(1.5rem,6vw,4rem)] max-[42rem]:p-4"
    aria-labelledby="error-view-title"
>
    <div
        class="supernova relative mb-[-1.5rem] grid aspect-square w-[min(64vw,34rem)] min-w-64 place-items-center max-[42rem]:mb-[-0.75rem] max-[42rem]:w-[min(78vw,22rem)]"
        aria-hidden="true"
    >
        <span
            class="aura pointer-events-none absolute aspect-square w-[68%] animate-supernova-pulse rounded-full bg-[radial-gradient(circle,var(--warning-background)_0_34%,transparent_68%)] opacity-90 blur-[1.6rem] motion-reduce:animate-none"
        ></span>
        <span
            class="orbit-outer pointer-events-none absolute h-[44%] w-[78%] animate-orbit-drift rounded-full border border-warning/35 [transform:rotate(-16deg)_skew(-8deg)] motion-reduce:animate-none"
        ></span>
        <span
            class="pointer-events-none absolute h-[30%] w-[52%] animate-orbit-drift-reverse rounded-full border border-warning/40 [transform:rotate(22deg)_skew(10deg)] motion-reduce:animate-none"
        ></span>
        <span
            class="pointer-events-none absolute top-[18%] left-[18%] size-[0.45rem] animate-particle-drift rounded-full bg-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--warning)_70%,transparent)] motion-reduce:animate-none"
        ></span>
        <span
            class="pointer-events-none absolute top-[31%] right-[12%] size-[0.32rem] animate-particle-drift rounded-full bg-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--warning)_70%,transparent)] [animation-delay:-1.2s] motion-reduce:animate-none"
        ></span>
        <span
            class="pointer-events-none absolute right-[22%] bottom-[20%] size-[0.56rem] animate-particle-drift rounded-full bg-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--warning)_70%,transparent)] [animation-delay:-2.4s] motion-reduce:animate-none"
        ></span>
        <span
            class="pointer-events-none absolute bottom-[24%] left-[14%] size-[0.28rem] animate-particle-drift rounded-full bg-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--warning)_70%,transparent)] [animation-delay:-3s] motion-reduce:animate-none"
        ></span>
        <span
            class="pointer-events-none absolute top-[11%] left-[64%] size-[0.24rem] animate-particle-drift rounded-full bg-warning shadow-[0_0_1rem_color-mix(in_srgb,var(--warning)_70%,transparent)] [animation-delay:-1.8s] motion-reduce:animate-none"
        ></span>
        <img
            class="error-art relative h-auto w-[78%] animate-art-float drop-shadow-[0_0_1.75rem_color-mix(in_srgb,var(--warning)_46%,transparent)] motion-reduce:animate-none"
            src={errorBurstUrl}
            alt=""
        />
    </div>

    <div
        class="content-card grid w-[min(100%,42rem)] gap-4 rounded-3xl border border-border/80 bg-card/88 p-[clamp(1.25rem,4vw,2.25rem)] shadow-[0_1.25rem_4rem_color-mix(in_srgb,var(--foreground)_18%,transparent)] backdrop-blur-xl max-[42rem]:rounded-[1.125rem]"
    >
        {#if status}
            <span
                class="w-max rounded-full bg-warning-background px-2.5 py-1 text-xs font-bold tracking-[0.08em] text-warning uppercase"
            >Error {status}</span>
        {/if}

        <h1 id="error-view-title">{title}</h1>
        <p class="text-base leading-[1.55] text-muted-foreground">{message}</p>

        {#if detail}
            <pre
                class="m-0 max-h-32 overflow-auto [overflow-wrap:anywhere] rounded-[0.875rem] border bg-background/72 px-4 py-3.5 font-mono text-sm leading-[1.45] whitespace-pre-wrap text-destructive"
            >{detail}</pre>
        {/if}

        <div class="flex flex-wrap items-center gap-3 pt-1">
            <a class={buttonVariants()} href={resolve("/")}>Back to start</a>
        </div>
    </div>
</section>
