import { type VariantProps, tv } from "tailwind-variants";

const tabsListVariants = tv({
  base: "rounded-2xl p-[3px] group-data-horizontal/tabs:h-8 group-data-vertical/tabs:p-1 data-[variant=line]:rounded-none group/tabs-list text-muted-foreground inline-flex w-fit items-center justify-center group-data-[orientation=vertical]/tabs:h-fit group-data-[orientation=vertical]/tabs:flex-col",
  variants: {
    variant: {
      default: "cn-tabs-list-variant-default bg-muted",
      line: "cn-tabs-list-variant-line gap-1 bg-transparent",
    },
  },
  defaultVariants: { variant: "default" },
});

type TabsListVariant = VariantProps<typeof tabsListVariants>["variant"];

export { tabsListVariants };
export type { TabsListVariant };
