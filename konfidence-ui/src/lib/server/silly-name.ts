/**
 * Generates a silly-but-friendly display name used by the stubbed login flow.
 *
 * Once a real IdP is plugged in, this module is no longer needed — the user's
 * name will come from the ID token claims.
 */

const ADJECTIVES = [
  "Whimsical",
  "Perplexed",
  "Caffeinated",
  "Bewildered",
  "Grumpy",
  "Nonchalant",
  "Sassy",
  "Sleepy",
  "Wobbly",
  "Curious",
  "Fluffy",
  "Cranky",
  "Melancholic",
  "Bouncy",
  "Suspicious",
  "Overconfident",
  "Bashful",
  "Peculiar",
  "Jittery",
  "Ambitious",
  "Reluctant",
  "Zesty",
  "Mischievous",
  "Dignified",
] as const;

const NOUNS = [
  "Panda",
  "Toaster",
  "Otter",
  "Penguin",
  "Cactus",
  "Waffle",
  "Sloth",
  "Narwhal",
  "Muffin",
  "Platypus",
  "Hedgehog",
  "Yak",
  "Meerkat",
  "Walrus",
  "Squirrel",
  "Kraken",
  "Dumpling",
  "Pineapple",
  "Llama",
  "Goblin",
  "Wizard",
  "Pickle",
  "Robot",
  "Bagel",
] as const;

const pickRandom = <Item>(list: readonly Item[]): Item => {
  const index = Math.floor(Math.random() * list.length);
  return list[index]!;
};

/**
 * Returns a silly display name, e.g. `Whimsical Panda`.
 */
const generateSillyName = (): string => `${pickRandom(ADJECTIVES)} ${pickRandom(NOUNS)}`;

export default generateSillyName;
