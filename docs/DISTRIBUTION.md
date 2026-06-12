# Distribution: One-Command Install, Channel-Pinned Updates

## Overview

Paivot ships as a coordinated set of artifacts: tool binaries (pvg, nd, vlt),
Claude Code plugins (paivot-graph, nd), and skills (vlt-skill). Versions of
these artifacts are NOT independent -- the plugin's hooks shell out to pvg, the
agents call nd and vlt, and a skew between any two of them is a bug class we
have hit repeatedly. Distribution v2 eliminates the skew by pinning the whole
set in one manifest: the stable channel.

```bash
# Install (one command, everything)
curl -fsSL https://raw.githubusercontent.com/paivot-ai/pvg/main/install.sh | sh

# Update (one command, converges onto the current pinned combo)
pvg update
```

## The channel manifest

`channel/stable.json` in this repository is the single source of truth for
what an installation should look like. Both the bootstrap installer and
`pvg update` read it from `main` and converge the machine onto the pinned
combination. The schema and the verification rules are documented in
[channel/README.md](../channel/README.md).

Key properties:

- **One tested combo.** Every change to `channel/**` runs the
  `channel-verify` CI workflow, which downloads each pinned tool release,
  sha256-verifies it against the release's `checksums.txt`, runs
  `<tool> --version`, and cross-checks plugin manifests and skill frontmatter.
  A combo on `main` is, by construction, a combination CI installed and
  verified.
- **Already-released only.** `stable.json` may only reference versions that
  are already published. Release first in the artifact repo, pin after. If a
  pinned tag does not exist, CI fails -- by design.
- **Rollback is a ref.** `pvg update --to <git-ref>` fetches `stable.json`
  at that ref and converges onto it. Any historical `main` combo is a
  known-good target.

## Pin + nudge, never silent

`pvg` notices when the channel has moved past the installed combo and nudges
the user, but it never updates on its own. Nothing changes until the user runs
`pvg update`. To hold a machine at a specific combo and suppress the nudge,
use `pvg update --pin <git-ref>`.

This is deliberate: agents run unattended for hours, and a mid-session binary
swap is exactly the kind of surprise the methodology exists to prevent.

## GitHub-source marketplaces

Plugins install from GitHub-source marketplaces (`paivot-ai/paivot-graph` and
`paivot-ai/nd`), not from local directory clones. Consumer machines need no
repository checkouts -- the only clone-based flow left is the development
install (`git clone` + `make install`), which registers the checkout as a
local marketplace for plugin development.

## Version sync inside this repo

Three files in this repo must always carry the same version: `VERSION`,
`.claude-plugin/plugin.json`, and `.claude-plugin/marketplace.json`. They are
bumped together with `make bump v=X.Y.Z` -- never by hand. Two gates enforce
this:

| Gate | When | What it checks |
|---|---|---|
| `make channel-check` | Locally, before pushing channel changes | `stable.json` parses, its paivot-graph pin matches `VERSION`, and the pvg pin matches the plugin version |
| `version-sync` CI workflow | Push to `main` / PRs touching `VERSION` or `.claude-plugin/**` | All three version files agree |
| `channel-verify` CI workflow | Push / PRs touching `channel/**` | The pinned combo actually exists and works (see above) |

The pvg binary version must always match the paivot-graph plugin version --
they release in lockstep.

## Release order

1. Release the artifact (pvg, nd, vlt, or this plugin). Each repo's release
   gates enforce its own internal version sync.
2. Update `channel/stable.json` to pin the new version; bump `updated`.
3. Run `make channel-check` locally.
4. Push. `channel-verify` smoke-tests the combo; merge when green.

`make fetch-tools` remains for development clones; for consumers, `pvg update`
supersedes it.

---

## Related

- [channel/README.md](../channel/README.md) -- channel schema, stamping, rollback
- [SEEDING.md](SEEDING.md) -- what to re-seed after an upgrade (`pvg seed --force`)
