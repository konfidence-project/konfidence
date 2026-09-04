---
theme: default
title: Seamless kden CLI installation
info: |
  Implementation plan for issue #941: a curl | sh installer for the kden CLI,
  and a SOTA `kden version` that surfaces the update path (re-run the installer).
canvasWidth: 980
aspectRatio: 16/9
transition: slide-left
class: text-center
mdc: true
---

# Seamless `kden` installation

### `curl | sh` · client-safe · any Unix · no package manager

<div class="opacity-70 mt-6 text-sm">issue #941 · implementation plan</div>

<!--
Implementation plan for #941. Every artifact name, ldflags path and API quirk is
verified against this repo's .goreleaser.yaml, cmd/kden, and hack/ (Sep 2026).
SOTA claims are grounded in the real install scripts / cobra source of the named tools.
-->

---
layout: default
---

# The ask & the shape

<div class="grid grid-cols-2 gap-6 mt-2 text-sm">
<div>

**Goal (#941)** — one command, any Unix, no brew:

```sh
curl -fsSL <url>/install.sh | sh
```

- install from a **GitHub release** …or **build from a git ref**
- update = **re-run the installer** (CLI never self-upgrades — see Phase 2)

</div>
<div v-click>

**Three phases**

- **0** fix the version-injection bug *(blocker)*
- **1** `hack/install.sh` — release or git-ref
- **2** SOTA `kden version` — surfaces the update path, doesn't self-upgrade

Package managers (brew/deb/rpm) = out of scope for v1.

</div>
</div>

<div v-click class="mt-5 text-sm opacity-80">
Already there: <code>.goreleaser.yaml</code> publishes <code>kden-cli-&lt;os&gt;-&lt;arch&gt;.tar.gz</code> + <code>checksums.txt</code>. No release-side change needed for Phase 1.
</div>

<!--
name_template (.goreleaser.yaml:87-97): lowercased .Os, amd64->x86_64, else verbatim
(arm64 stays arm64). checksums.txt ships in every release. darwin/amd64 is in the build
`ignore` list — no Intel-Mac artifact. No signs:/cosign block yet.
-->

---
layout: default
---

# Phase 0 — fix the version bug (blocker)

goreleaser injects the version into a package that **doesn't hold the var**.

```yaml
# .goreleaser.yaml:63 — targets cmd/version.version …
-X github.com/konfidence-project/konfidence/cmd/version.version={{.Version}}
```
```go
// …but the var lives in cmd/kden/cmd/version/version.go:10
var version = "0.0.1"   // package .../cmd/kden/cmd/version
```

<div v-click class="mt-3 text-sm">

Paths don't match → injection **silently no-ops** → every release prints `0.0.1`. `kden version` would lie about which build it is, and any server-driven skew check (future) would compare against a bogus version.

**Fix:** point kden at `pkg/build.Version` (already wired for the other two binaries), delete the local var. One source of truth, smaller diff.

</div>

<!--
Verified: `ls cmd/version/` -> not found. Only version var is at
cmd/kden/cmd/version/version.go:10; ldflags target cmd/version.version. Real mismatch.
Root-cause fix reuses pkg/build.Version (.goreleaser.yaml:45,:76 already inject it for
konfidence + api). Also switch to cmd.OutOrStdout() so the version test captures output.
-->

---
layout: default
---

# Phase 1 — `hack/install.sh` (POSIX `sh`)

```sh {all|1-2|4-6|8-10}
os=$(uname -s | tr A-Z a-z)                       # linux | darwin
case $(uname -m) in x86_64|amd64) a=x86_64;; arm64|aarch64) a=arm64;; *) die;; esac
asset="kden-cli-${os}-${a}.tar.gz"                # matches goreleaser name_template

tag=${KDEN_VERSION:-$(latest_release)}            # /releases/latest, fallback /releases?per_page=1
http_get "$REPO/releases/download/$tag/$asset" > "$tmp/$asset"   # curl OR wget, -f fails on 4xx
verify_sha256 "$asset"                            # checksums.txt; fatal on mismatch

tar -xzf "$tmp/$asset" -C "$tmp"
install -m755 via mv "$tmp/kden" "${KDEN_INSTALL_DIR:-$HOME/.local/bin}/kden"   # no sudo, atomic
# print PATH export line if dir not on $PATH — never edit rc files
```

<div class="grid grid-cols-2 gap-4 mt-3 text-xs opacity-80">
<div v-click>

`#!/bin/sh` + `set -eu` — bash isn't guaranteed (Alpine, macOS bash 3.2). `aarch64→arm64` is the load-bearing map. Rosetta Intel-Mac → probe `hw.optional.arm64`, else clear error.

</div>
<div v-click>

`prerelease: auto` + no stable release yet → the `/releases?per_page=1` fallback is **mandatory**. Honor `GITHUB_TOKEN` for CI rate limits.

</div>
</div>

<!--
uname -m isn't normalized (linux arm64=aarch64, mac arm64=arm64). checksums.txt verify:
sha256sum on linux, shasum -a256 on mac — detect. curl -f prevents an HTML 404 body
landing in the tarball. ~/.local/bin default = no sudo + the standard user bin dir.
-->

---
layout: default
---

# Phase 1 — two independent axes

Fetching the **script** and choosing the **install source** are decoupled.

<div class="grid grid-cols-2 gap-5 mt-3 text-sm">
<div v-click>

**Script source — always `main`**

- the `konfidence.cloud` stub always fetches `.../main/hack/install.sh`
- users always get the newest installer logic (fixes, arch detection) — no per-release URL churn
- same as Flux (`fluxcd.io/install.sh` = latest `main`)

</div>
<div v-click>

**Install source — chosen at runtime**

- the script (whatever its version) resolves: release → `KDEN_VERSION` → `KDEN_GIT_REF`
- an always-`main` script installs **any** release or ref
- version installed ≠ version of the script that installed it

</div>
</div>

<div v-click class="mt-4 p-3 rounded bg-green-500 bg-opacity-10 text-sm">
So we <b>never</b> need to fetch the script from a pinned ref. Always <code>main</code> is correct <b>forever</b> — self-updating installer, install target picked by env var. Only users who want a byte-reproducible, auditable install pin the <i>script</i> URL to a tag too (belt-and-suspenders, not the norm).
</div>

<!--
The core insight from the discussion: script-version and installed-version are separate
axes. Always-main script = Flux/Istio model (their vanity URLs always serve latest main
and still install pinned versions via FLUX_VERSION/ISTIO_VERSION). The only knock on
always-main is byte-reproducibility (piped bytes can change between runs) — the answer
is "pin the script to a tag if you care", not "make everyone pin". This MOOTS the earlier
"pin the stub to a tag once releases exist" note — the stub tracks main permanently.
-->

---
layout: default
---

# Phase 1 — source switch: release **or** git ref

```sh {all|1-2|4-5|7-8}
# default: latest release
curl -fsSL https://konfidence.cloud/install.sh | sh

# pin the installed release
curl -fsSL https://konfidence.cloud/install.sh | KDEN_VERSION=v0.3.0 sh

# install from any git ref (branch/tag/sha) — contributor opt-in, needs Go
curl -fsSL https://konfidence.cloud/install.sh | KDEN_GIT_REF=my-branch sh
```

<div v-click class="mt-3 text-sm">

Resolution: **`KDEN_GIT_REF`** (force source build) → **`KDEN_VERSION`** (pinned release) → **latest release**.

`KDEN_GIT_REF` is a **permanent contributor knob** — "install `kden` from my branch / an unreleased `main`" — not a pre-release crutch. It also covers unpublished platforms (e.g. darwin/amd64).

Source path: `git clone --depth 1 --branch <ref>` + `go build -ldflags "…pkg/build.Version=<ref>" ./cmd/kden` (not `go install` — monorepo subpath drops the version ldflag). Skips checksum; trust anchor is the ref.

</div>

<div v-click class="mt-3 p-2 rounded bg-blue-500 bg-opacity-10 text-xs">
<b>Resolution never silently compiles.</b> <code>latest_release</code> already falls back from <code>/releases/latest</code> to the newest release <i>including prereleases</i> (so the <code>alpha</code> tag resolves). If it still finds nothing, that's a clear <b>error</b> — not an implicit build from <code>main</code>. Source builds are always the explicit <code>KDEN_GIT_REF</code> opt-in.
</div>

<!--
CHANGED: removed the auto-build-from-main fallback entirely. It rested on a false
premise ("no release published yet") — but alpha IS a published release, and
latest_release's /releases?per_page=1 branch already finds prereleases. So the
fallback was dead code that, if ever hit, would silently compile from main (needs Go,
not what the installer user asked for). Now: empty resolution => die with a clear
message pointing at KDEN_VERSION / KDEN_GIT_REF. KDEN_GIT_REF stays the permanent,
explicit source-build opt-in (contributors, "try my branch", unpublished arch).
go install of ./cmd/kden won't apply -X version -> clone+build with pkg/build.Version.
--branch handles tags/branches; sha needs clone-then-checkout. No artifact => no checksum.
-->

---
layout: default
---

# Phase 2 — why the CLI must **not** self-upgrade

The CLI is a **client** of the API + controllers. It must never move ahead of them.

<div class="grid grid-cols-2 gap-5 mt-3 text-sm">
<div v-click>

**The hazard we avoid**

- `kden upgrade` / an auto-notifier nudges the user to a newer CLI
- their **API/controllers are still older** → version skew
- new CLI speaks a contract the server doesn't → broken UX, confusing errors

</div>
<div v-click>

**So we removed both**

- ❌ `kden upgrade` (self-replace)
- ❌ the "new version available" notifier
- ✅ update = **re-run the installer**, a deliberate act the user times

</div>
</div>

<div v-click class="mt-4 p-3 rounded bg-green-500 bg-opacity-10 text-sm">
This is the <b>Istio / Flux / Helm</b> model — and here it's the <b>right</b> call, not a UX compromise: a client racing ahead of its server is the actual danger. Re-running <code>curl | sh</code> keeps the human in control of when the CLI moves.
</div>

<!--
The reversal from the earlier plan, and WHY: kden talks to the Konfidence API and
controllers. If the CLI self-upgrades (or nags the user to), it can jump ahead of a
cluster still running older controllers/API — version skew that breaks the user. The
safe posture: the CLI never upgrades itself. Update is re-running the installer, timed
by the user (who knows their cluster's version). Removed cmd/kden/cmd/upgrade and
internal/kden/release entirely; go-github + semver revert to indirect. This is the
Camp-1 (Istio/Flux/Helm) model — correct here precisely because we're a client.
-->

---
layout: default
---

# Phase 2 — SOTA `kden version` (respects `--output`)

Version output surfaces the update path; it doesn't *perform* it.

```sh {all|1-3|5-6|8-9}
$ kden version                 # default json — matches every other command
{ "version": "v0.4.0", "commit": "abc1234",
  "goVersion": "go1.26", "platform": "linux/arm64", "buildDate": "…" }

$ kden version --output pretty # aligned human block
kden v0.4.0 · commit abc1234 · linux/arm64 · built …

# to stderr, released builds only — never pollutes stdout:
To update, re-run the installer:  curl -fsSL https://konfidence.cloud/install.sh | sh
```

<div v-click class="mt-3 text-sm opacity-80">
Fixes a real bug: the old <code>version</code> printed a hardcoded line and <b>ignored <code>--output</code></b>. Now json/yaml go through the shared <code>output.ResolveFormat</code>; pretty is a direct block. The update hint lives on <b>stderr</b>, so <code>kden version | jq</code> stays clean and <code>dev</code> builds print no hint.
</div>

<!--
version now builds an Info{Version,Commit,GoVersion,Platform,BuildDate} and routes
json/yaml through internal/kden/output (same path as project-list); pretty is written
directly (a version is scalar, not tabular — no bubbletea table). The update hint is
the ONLY thing on stderr and only for non-dev builds, so machine consumers piping
stdout get pure json/yaml. build.Date is a new pkg/build var injected via goreleaser
ldflags for all three binaries. This is the gh/istioctl idiom: tell the user how to
update, don't do it for them.
-->

---
layout: default
---

# Future — let the **server** drive the "update" signal

The safe direction is server → client, never client racing ahead.

<div class="grid grid-cols-2 gap-5 mt-3 text-sm">
<div v-click>

**When it's actually needed**

- the API bumps its contract version
- an **older CLI** calls a newer API
- API responds with a version-skew signal (header / field)

</div>
<div v-click>

**What the CLI does then**

- surfaces: *"your kden is too old for this API — update"*
- points at the same `curl | sh` one-liner
- the user updates deliberately, now with a real reason

</div>
</div>

<div v-click class="mt-4 p-3 rounded bg-blue-500 bg-opacity-10 text-sm">
Not built now (YAGNI — the API contract isn't versioned yet). Noted as the correct future shape: the <b>server</b> knows the compatibility window, so the server tells the client to catch up — the inverse of a client that upgrades itself into incompatibility.
</div>

<!--
The one legitimate "please update" case is the OPPOSITE of self-upgrade: server tells
an old client it's too old. That's safe because the server owns the compatibility
window. Mechanism (future): API returns a min-CLI-version / skew header; the kden API
client checks it and prints an update prompt pointing at the installer. Deliberately
not implemented — no versioned API contract exists yet, and building the seam now would
be speculative. This slide records the direction so the removed notifier isn't mistaken
for "we never want any update signal" — we want it, but server-driven.
-->

---
layout: default
---

# Hosting the `curl | sh` URL

The docs site already gives us a clean URL — no new infra, no redirect trick.

<div class="grid grid-cols-2 gap-6 mt-3 text-sm">
<div v-click>

**The clean one — `konfidence.cloud`**

```sh
curl -fsSL https://konfidence.cloud/install.sh | sh
```
`konfidence-docs` is **VitePress on GitHub Pages** at `konfidence.cloud`. Its `public/install.sh` is a **one-line bootstrap** that forwards to the real script — not a copy.

</div>
<div v-click>

**What lives at that URL** (`public/install.sh`)

```sh
#!/bin/sh
exec sh -c "$(curl -fsSL https://raw.\
githubusercontent.com/konfidence-project/\
konfidence/main/hack/install.sh)"
```
Real installer stays in **one place** (`konfidence/hack/install.sh`). Stub ~never changes → no sync, no drift.

</div>
</div>

<div v-click class="mt-4 p-3 rounded bg-green-500 bg-opacity-10 text-sm">
<b>Why a stub, not a redirect or a copy:</b> GitHub Pages is static — it <b>can't</b> issue a 302, and a <code>meta-refresh</code> serves HTML that would pipe garbage into <code>sh</code>. A bootstrap that <i>fetches-and-execs</i> the raw file is the "redirect from the docs domain" done the only way a static host allows: one source of truth, pretty URL, two hops. The stub tracks <code>main</code> <b>permanently</b> — the installed version is chosen at runtime, not by which script you fetched.
</div>

<!--
Verified live: konfidence-docs Pages → cname konfidence.cloud, VitePress 2.0.0-alpha.19,
public/ served verbatim (favicon.ico proves it). public/install.sh is a tiny bootstrap
that curls the real script from the main repo's raw URL and execs it — no COPY, no drift.
GitHub Pages can't do a real 302 (static host) and a meta-refresh returns HTML that
breaks curl|sh — so fetch-and-exec is the correct "forward from the docs domain".
Stub tracks main FOREVER (not "pin to a tag later"): script-version and installed-version
are independent axes — install target is picked by KDEN_VERSION/KDEN_GIT_REF at runtime.
Trade-off: two network hops. Reproducibility-minded users pin the script URL to a tag.
-->

---
layout: default
---

# SOTA reality check

<div class="text-xs mt-2">

| Tool | Install | Checksum | Self-update cmd | Notify on run |
|---|---|---|---|---|
| istioctl / Flux / Helm / Argo **← us** | `curl \| sh` | ✅ *(Istio ❌)* | ❌ re-run script | ❌ |
| gh | pkg mgr | pkg mgr | ❌ → `brew upgrade` | ✅ stderr, 24h, TTY+CI gated |
| krew | curl+sha | ✅ | plugins only | ✅ ~40% of runs |
| Deno / rustup | `curl \| sh` | ✅ | ✅ self-replace | ❌ |

</div>

<div v-click class="mt-3 p-2 rounded bg-red-500 bg-opacity-10 text-sm">
<b>Myth-buster:</b> <code>istioctl upgrade</code> upgrades the <b>cluster control plane</b>, not the CLI. <code>krew upgrade</code> = plugins. <b>None of the cluster-facing tools self-updates or notifies</b> — because, like kden, they're clients of a server whose version they must not outrun.
</div>

<div v-click class="mt-2 text-sm opacity-80">
Our lineage: installer ← Flux/Helm · <b>no self-update, no notifier ← Istio/Flux/Helm/Argo</b> (deliberate, we're a client). Deno/rustup self-replace because they answer to no server; we don't get that luxury.
</div>

<!--
Grounded in fetched source: istio downloadIstioCandidate.sh (no verify), flux install/
flux.sh (sha256), helm get-helm-3, k3s get.k3s.io, deno.land/install.sh + `deno upgrade`,
sh.rustup.rs, cli/cli internal/update/update.go, sigs.k8s.io/krew root.go (0.4 sampling).
Everyone converges on curl|sh + version pin + sha256, and nobody silently self-updates.
-->

---
layout: default
---

# Files & rollout

<div class="grid grid-cols-2 gap-5 mt-2 text-sm">
<div>

| File | Phase |
|---|---|
| `.goreleaser.yaml` | 0 — ldflags → `pkg/build` (+ `Date`) |
| `pkg/build/build.go` | 0 — add `Date` var |
| `cmd/kden/cmd/version/version.go` | 2 — `--output`-aware version |
| `hack/install.sh` | 1 — the installer |
| `cmd/kden/cmd/root.go` | 2 — *un*wire notifier/upgrade |
| `../konfidence-docs` install page | 3 — upstream PR |

</div>
<div v-click>

**Sequence**

- **0 → 1 → 2**; 0 gates 2 (version must be right to display)
- **1 stands alone** — satisfies #941's install ask on its own
- **Removed:** `cmd/kden/cmd/upgrade/`, `internal/kden/release/` (no self-upgrade)
- CI: `shellcheck --shell=sh`, smoke matrix (ubuntu+macos), version-output table tests
- **Releases:** `alpha` is a published release, so `latest_release` resolves it via the prerelease-including fallback — no special-casing, no build-from-`main`

</div>
</div>

<!--
Phase 2 is now version output only — no upgrade/notifier. cmd/kden/cmd/upgrade and
internal/kden/release are DELETED, not added. root.go change is removing the notifier
wiring. version.go routes json/yaml through internal/kden/output and honors --output.
build.Date is a new pkg/build var. archiveName() lives in install.sh only now. Windows:
installer is Unix-only; Windows users grab the .exe archive manually.
-->

---
layout: default
---

# Phase 3 — upstream the install docs

The installer is only "seamless" if the docs tell users the one command. PR to `../konfidence-docs`.

<div class="grid grid-cols-2 gap-5 mt-2 text-sm">
<div v-click>

**What ships in the docs PR**

- an **Install** page: the one-liner + `KDEN_VERSION` / `KDEN_GIT_REF` env knobs
- how to update: **re-run the installer** (CLI doesn't self-upgrade) + `kden version`
- the `public/install.sh` bootstrap stub (same PR — it's what makes the URL real)

</div>
<div v-click>

**Why it's a separate repo PR**

- `konfidence-docs` is its own repo (VitePress → `konfidence.cloud`)
- the stub + the page land together → the pretty URL and its docs are atomic
- links back with `Resolves <#941 url>`; cross-repo so merging closes the issue

</div>
</div>

<div v-click class="mt-4 p-3 rounded bg-green-500 bg-opacity-10 text-sm">
Order: land <code>hack/install.sh</code> here (Phase 1) <b>first</b> — the stub fetches it from <code>main</code>, so the raw URL must resolve before the docs PR merges. Then the docs PR adds the stub + the page in one shot.
</div>

<!--
The docs PR is the user-facing half of #941 — an installer nobody's told about isn't
"seamless". It lives in the sibling konfidence-docs repo (separate VitePress site,
cname konfidence.cloud). Bundle the public/install.sh bootstrap stub WITH the install
page in that one PR so the clean URL and its documentation are atomic. Sequencing:
hack/install.sh must exist on main BEFORE the stub goes live, or the bootstrap curls a
404. Use "Resolves <full issue url>" (cross-repo) so merging the docs PR can close #941.
-->

---
layout: center
class: text-center
---

# End state

<div class="text-left max-w-2xl mx-auto mt-4 text-sm">

```sh
curl -fsSL https://konfidence.cloud/install.sh | sh
kden version              # → real release version, commit, platform (Phase 0+2)
kden version --output yaml # respects --output like every other command
#   To update, re-run the installer: curl … | sh   ← stderr, released builds only
curl -fsSL https://konfidence.cloud/install.sh | sh   # update = re-run, user-timed
```

</div>

<div v-click class="mt-8 opacity-80 text-sm">
Phase 1 satisfies #941. Phase 0 makes the version honest; Phase 2 surfaces the update path without ever letting the client outrun its server.
</div>

<div class="abs-br m-6 text-xs opacity-50">
Resolves https://github.com/konfidence-project/konfidence-project/issues/941
</div>
