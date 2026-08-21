#!/usr/bin/env bash
# Transform raw crd-ref-docs markdown into the VitePress-ready reference doc.
# Reads raw markdown on stdin, writes the transformed doc to stdout.
#
#   crd-ref-docs ... --output-path=<tmp> \
#     && hack/transform-crd-docs.sh < <tmp>/out.md > api/docs/crd.md
#
# Three passes:
#   1. Prepend VitePress frontmatter.
#   2. Drop the generated `XxxList` wrapper types (noise for API consumers) and
#      demote the remaining per-type headings H4 -> H3.
#   3. Inject the validated sample from test/data/samples under each Kind that
#      has one, as a `#### Example` fenced yaml block.
#
# Samples are the same fixtures `make validate` checks against the CRD schema,
# so every embedded example is guaranteed schema-correct.
#
# Portable to bash 3.2 (macOS): no associative arrays — awk owns the Kind->sample
# map, built by scanning the samples dir for `kind:` declarations.
set -euo pipefail

SAMPLES_DIR="${SAMPLES_DIR:-test/data/samples}"

# --- Pass 1: frontmatter -----------------------------------------------------
cat <<'FRONT'
---
title: CRD
description: Custom Resource Definition specifications for Konfidence Kubernetes resources.
outline: [2, 3]
editLink: true
lastUpdated: true
---

FRONT

# --- Passes 2 + 3: strip List types, H4->H3, inject samples ------------------
# Build a Kind<TAB>sample-path map into a temp file (portable: no bash 4 assoc
# arrays, no multi-line awk -v which BWK awk on macOS rejects).
MAP=$(mktemp)
trap 'rm -f "$MAP"' EXIT
for f in "$SAMPLES_DIR"/*.yaml; do
  [ -e "$f" ] || continue
  kind=$(awk '/^kind:[[:space:]]/ { sub(/^kind:[[:space:]]+/, ""); print; exit }' "$f")
  [ -n "${kind:-}" ] && printf '%s\t%s\n' "$kind" "$f" >> "$MAP"
done

awk -v mapfile="$MAP" '
  BEGIN {
    while ((getline line < mapfile) > 0) {
      idx = index(line, "\t")
      k = substr(line, 1, idx - 1)
      p = substr(line, idx + 1)
      sample[k] = p                        # last sample wins per Kind
    }
    close(mapfile)
    pending = ""
  }

  # Flush a pending sample block before the next section boundary.
  function flush_sample(   l) {
    if (pending == "") return
    printf "#### Example\n\n```yaml\n"
    while ((getline l < sample[pending]) > 0) print l
    close(sample[pending])
    print "```"
    print ""
    pending = ""
  }

  # A "#### XxxList" heading opens a wrapper block we skip until the next "#### ".
  /^#### [A-Za-z]+List$/ { flush_sample(); skip = 1; next }

  # Drop link-list references to the stripped List types (in the "Resource Types"
  # index and the per-type "Appears in:" blocks) so we leave no dangling anchors.
  /^- \[[A-Za-z]+List\]\(#[a-z]+list\)$/ { next }

  /^#### / {
    flush_sample()
    skip = 0
    sub(/^#### /, "### ")
    print
    kind = $0
    sub(/^### /, "", kind)
    if (kind in sample) pending = kind
    next
  }

  # H2 boundaries also flush any pending sample.
  /^## / { flush_sample() }

  skip { next }
  { print }

  END { flush_sample() }
' | awk 'NF { last = NR } { lines[NR] = $0 } END { for (i = 1; i <= last; i++) print lines[i] }'
# ^ trim trailing blank lines so the file ends with exactly one newline. The
#   sample-injection emits a blank line after every Example block, so without this
#   the file's trailing whitespace would vary with content — making check-generate's
#   regenerate-then-diff produce spurious whitespace-only drift. Canonical output
#   keeps that byte-comparison meaningful.
