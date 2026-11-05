/** Compact presentation only: filtering, identity and details retain the original reference. */
const vectorLabel = (reference: string): string => {
  const value = reference.trim();
  const slash = value.lastIndexOf("/");
  const schemeDelimiter = "://";
  const scheme = value.indexOf(schemeDelimiter);
  // A scheme/host without a repository path is not an image basename.
  if (scheme !== -1 && !value.includes("/", scheme + schemeDelimiter.length)) {
    return value;
  }
  return slash !== -1 && slash < value.length - 1 ? value.slice(slash + 1) : value;
};

export { vectorLabel };
