import { error } from "@sveltejs/kit";

import type { StageList } from "$lib/stages";

const fetchStages = async (fetch: typeof globalThis.fetch) => {
    const response = await fetch("/api/stages");

    if (!response.ok) {
        error(response.status, "Failed to load stages");
    }

    return ((await response.json()) as StageList).items;
};

export const load = ({ fetch }) => {
    return {
        stages: fetchStages(fetch),
    };
};
