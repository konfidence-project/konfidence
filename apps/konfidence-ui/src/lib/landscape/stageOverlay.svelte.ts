import { pushState } from "$app/navigation";
import { page } from "$app/state";

import type { Stage } from "#lib/konfidence-api/types.ts";
import { stageDetailsUrl } from "#lib/landscape/url.ts";

interface StageOverlayOptions {
  embedded: () => boolean;
  projectId: () => string;
  stages: () => readonly Stage[];
}

type StageOverlayReference = NonNullable<App.PageState["stageOverlay"]>;

const openStageOverlay = (
  event: MouseEvent,
  href: string,
  reference: StageOverlayReference,
): void => {
  if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
    return;
  }

  event.preventDefault();
  // SvelteFlow proxies node data; copy the primitive fields into a structured-cloneable history value.
  pushState(href, {
    stageOverlay: {
      landscapeId: reference.landscapeId,
      projectId: reference.projectId,
      stageId: reference.stageId,
    },
  });
};

class StageOverlay {
  readonly #embedded: () => boolean;
  readonly #projectId: () => string;
  readonly #stages: () => readonly Stage[];

  constructor({ embedded, projectId, stages }: StageOverlayOptions) {
    this.#embedded = embedded;
    this.#projectId = projectId;
    this.#stages = stages;
  }

  get selected(): Stage | undefined {
    const reference = page.state.stageOverlay;
    if (!reference || reference.projectId !== this.#projectId()) {
      return undefined;
    }
    return this.#stages().find(
      (stage) => stage.id === reference.stageId && stage.landscapeId === reference.landscapeId,
    );
  }

  get selectedHref(): string | undefined {
    const stage = this.selected;
    return stage ? this.hrefFor(stage) : undefined;
  }

  hrefFor = (stage: Stage): string =>
    stageDetailsUrl({
      embedded: this.#embedded(),
      landscapeId: stage.landscapeId,
      projectId: this.#projectId(),
      stageId: stage.id,
    });

  referenceFor = (stage: Stage): StageOverlayReference => ({
    landscapeId: stage.landscapeId,
    projectId: this.#projectId(),
    stageId: stage.id,
  });

  close = (): void => {
    globalThis.history.back();
  };
}

const useStageOverlay = (options: StageOverlayOptions): StageOverlay => new StageOverlay(options);

export { StageOverlay, openStageOverlay, useStageOverlay };
export type { StageOverlayOptions, StageOverlayReference };
