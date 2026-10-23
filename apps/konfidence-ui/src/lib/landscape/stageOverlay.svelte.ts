import { goto, pushState, replaceState } from "$app/navigation";
import { page } from "$app/state";

import type { Stage } from "#lib/konfidence-api/types.ts";
import { stageDetailsUrl } from "#lib/landscape/url.ts";
import { projectLandscapeUrl } from "#lib/projects/url.ts";

interface StageOverlayOptions {
  embedded: () => boolean;
  projectId: () => string;
  stages: () => readonly Stage[];
}

type StageOverlayReference = NonNullable<App.PageState["stageOverlay"]>;

const setStageOverlay = (
  href: string,
  reference: StageOverlayReference,
  replace: boolean,
): void => {
  const state: App.PageState = {
    overlayOrigin: page.state.overlayOrigin ?? "landscape",
    stageOverlay: reference,
  };
  if (replace) {
    void replaceState(href, state);
  } else {
    void pushState(href, state);
  }
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

  open = (stage: Stage, replace: boolean): void => {
    setStageOverlay(this.hrefFor(stage), this.referenceFor(stage), replace);
  };

  close = (): void => {
    if (page.state.overlayOrigin === "landscape") {
      globalThis.history.back();
      return;
    }
    void goto(projectLandscapeUrl(this.#projectId(), this.#embedded()), {
      replace: true,
      reset: false,
    });
  };
}

const useStageOverlay = (options: StageOverlayOptions): StageOverlay => new StageOverlay(options);

export { StageOverlay, useStageOverlay };
export type { StageOverlayOptions, StageOverlayReference };
