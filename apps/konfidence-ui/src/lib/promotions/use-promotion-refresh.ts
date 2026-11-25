import { onMount } from "svelte";

// Temporary, page-local lifecycle. Replace with the planned shared refresh
// mechanism rather than growing interval preferences or another query cache here.
const REFRESH_INTERVAL_MS = 10_000;

const usePromotionRefresh = (reload: () => void): void => {
  onMount(() => {
    let interval: ReturnType<typeof setInterval> | undefined = undefined;
    const stop = (): void => {
      clearInterval(interval);
      interval = undefined;
    };
    const start = (): void => {
      interval = setInterval(reload, REFRESH_INTERVAL_MS);
    };
    const visibilityChanged = (): void => {
      stop();
      if (document.visibilityState === "visible") {
        reload();
        start();
      }
    };
    if (document.visibilityState === "visible") {
      start();
    }
    document.addEventListener("visibilitychange", visibilityChanged);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", visibilityChanged);
    };
  });
};

export { usePromotionRefresh };
