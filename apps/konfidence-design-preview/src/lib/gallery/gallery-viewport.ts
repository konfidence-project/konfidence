interface GalleryViewport {
  pause: () => void;
  settleNavigation: () => void;
  destroy: () => void;
}

const visibleComponent = (gallery: HTMLElement): HTMLElement | undefined => {
  const line =
    (document.querySelector(".preview-topbar")?.getBoundingClientRect().bottom ?? 0) + 32;
  return [...gallery.querySelectorAll<HTMLElement>("[data-catalog-status]")].find((element) => {
    const bounds = element.getBoundingClientRect();
    return (
      !element.hidden &&
      bounds.height > 0 &&
      bounds.bottom > line &&
      bounds.top < window.innerHeight
    );
  });
};

const listenForManualScroll = (interruptNavigation: () => void): (() => void) => {
  const lifecycle = new AbortController();
  const keyboardScroll = (event: KeyboardEvent): void => {
    if (
      event.defaultPrevented ||
      !["PageDown", "PageUp", "Home", "End", "ArrowDown", "ArrowUp", " "].includes(event.key)
    ) {
      return;
    }
    if (
      event.target instanceof Element &&
      event.target.closest(
        'input, textarea, [contenteditable="true"], [role="tablist"], [role="listbox"]',
      )
    ) {
      return;
    }
    interruptNavigation();
  };
  const scrollbar = (event: PointerEvent): void => {
    if (event.clientX >= document.documentElement.clientWidth) {
      interruptNavigation();
    }
  };
  for (const event of ["wheel", "touchmove"] as const) {
    globalThis.addEventListener(event, interruptNavigation, {
      passive: true,
      signal: lifecycle.signal,
    });
  }
  globalThis.addEventListener("keydown", keyboardScroll, { signal: lifecycle.signal });
  globalThis.addEventListener("pointerdown", scrollbar, { signal: lifecycle.signal });
  return () => lifecycle.abort();
};

// Track the top visible component only after scrolling settles. Native anchor
// animations must not replace their destination with cards passed along the way.
const watchGalleryViewport = (
  gallery: HTMLElement,
  onVisible: (id: string) => void,
): GalleryViewport => {
  let timer: ReturnType<typeof setTimeout> | undefined = undefined;
  let paused = true;
  const schedule = (): void => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      if (paused) {
        paused = false;
        return;
      }
      const component = visibleComponent(gallery);
      if (component) {
        onVisible(component.id);
      }
    }, 150);
  };
  const stopIntent = listenForManualScroll(() => {
    if (paused) {
      paused = false;
      clearTimeout(timer);
    }
  });
  globalThis.addEventListener("scroll", schedule, { passive: true });
  schedule();
  return {
    destroy() {
      clearTimeout(timer);
      globalThis.removeEventListener("scroll", schedule);
      stopIntent();
    },
    pause() {
      paused = true;
      clearTimeout(timer);
    },
    settleNavigation() {
      paused = true;
      schedule();
    },
  };
};

export { watchGalleryViewport };
export type { GalleryViewport };
