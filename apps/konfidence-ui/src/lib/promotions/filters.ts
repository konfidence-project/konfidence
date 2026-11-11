import type { Landscape, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
import {
  approvalGate,
  landscapeLabel,
  referenceLabel,
  statusLabel,
  toPromotionGroup,
} from "./promotions.ts";
import type { Promotion, PromotionGroup, PromotionReference } from "./promotions.ts";

interface PromotionFilter {
  landscape: string;
  stage: string;
  config: string;
  query: string;
}
interface FilterOption {
  value: string;
  label: string;
}

const stageValue = (reference: PromotionReference): string =>
  JSON.stringify([reference.landscape ?? "", reference.name]);

const matchesEndpoints = (
  flow: Pick<Promotion, "source" | "target">,
  filter: Pick<PromotionFilter, "landscape" | "stage">,
): boolean => {
  if (!filter.landscape && !filter.stage) {
    return true;
  }
  return [flow.source, flow.target].some(
    (endpoint) =>
      endpoint.kind === "Stage" &&
      (!filter.landscape || endpoint.landscape === filter.landscape) &&
      (!filter.stage || stageValue(endpoint) === filter.stage),
  );
};

const referenceSearch = (
  reference: PromotionReference,
  landscapes: readonly Landscape[],
): string[] => [reference.name, reference.landscape ?? "", referenceLabel(reference, landscapes)];

const matchesText = (fields: readonly string[], query: string): boolean =>
  fields.some((field) => field.toLowerCase().includes(query));

const filterPromotionGroups = (
  configs: readonly VectorPromotionConfig[],
  landscapes: readonly Landscape[],
  filter: PromotionFilter,
): PromotionGroup[] => {
  const query = filter.query.trim().toLowerCase();
  return configs
    .toSorted((left, right) => left.id.localeCompare(right.id))
    .flatMap((config) => {
      if (filter.config && filter.config !== config.id) {
        return [];
      }
      const parentMatches = matchesText(
        [
          config.id,
          ...referenceSearch(config.source, landscapes),
          ...referenceSearch(config.target, landscapes),
        ],
        query,
      );
      const children = config.promotions.filter(
        (promotion) =>
          matchesEndpoints(promotion, filter) &&
          (parentMatches ||
            matchesText(
              [
                promotion.id,
                promotion.vector,
                statusLabel(promotion),
                approvalGate(promotion)?.setBy ?? "",
                ...referenceSearch(promotion.source, landscapes),
                ...referenceSearch(promotion.target, landscapes),
              ],
              query,
            )),
      );
      return children.length || (parentMatches && matchesEndpoints(config, filter))
        ? [toPromotionGroup(config, children)]
        : [];
    });
};

const toOptions = (map: Map<string, string>): FilterOption[] =>
  [...map]
    .map(([value, label]) => ({ label, value }))
    .toSorted((left, right) => left.label.localeCompare(right.label));

const compatibleOptions = (
  configs: readonly VectorPromotionConfig[],
  landscapes: readonly Landscape[],
  filter: Pick<PromotionFilter, "landscape" | "stage">,
): { landscapes: FilterOption[]; stages: FilterOption[]; configs: FilterOption[] } => {
  const landscapeOptions = new Map(
    landscapes.map((landscape) => [landscape.id, landscapeLabel(landscape.id, landscapes)]),
  );
  const stageOptions = new Map<string, string>();
  const endpoints = configs
    .flatMap((config) => [config, ...config.promotions])
    .flatMap((flow) => [flow.source, flow.target])
    .filter((endpoint) => endpoint.kind === "Stage");
  for (const endpoint of endpoints) {
    if (endpoint.landscape && !landscapeOptions.has(endpoint.landscape)) {
      landscapeOptions.set(endpoint.landscape, landscapeLabel(endpoint.landscape, landscapes));
    }
    if (!filter.landscape || endpoint.landscape === filter.landscape) {
      stageOptions.set(stageValue(endpoint), referenceLabel(endpoint, landscapes));
    }
  }
  return {
    configs: configs
      .filter((config) =>
        [config, ...config.promotions].some((flow) => matchesEndpoints(flow, filter)),
      )
      .map((config) => ({ label: config.id, value: config.id }))
      .toSorted((left, right) => left.label.localeCompare(right.label)),
    landscapes: toOptions(landscapeOptions),
    stages: toOptions(stageOptions),
  };
};

export { compatibleOptions, filterPromotionGroups, stageValue };
export type { PromotionFilter };
