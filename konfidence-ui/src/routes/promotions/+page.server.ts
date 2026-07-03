import { error } from "@sveltejs/kit";

const ENHANCE_YOUR_CALM = 420;

const load = () => {
  error(ENHANCE_YOUR_CALM, "Enhance your calm");
};

export { load };
