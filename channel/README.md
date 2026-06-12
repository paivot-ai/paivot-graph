# The stable channel

`stable.json` is the single source of truth for what a Paivot installation
should look like. The bootstrap installer and `pvg update` both read this file
from `main` of this repository and converge the machine onto the pinned
combination: tool binaries (pvg, nd, vlt), Claude Code plugins (paivot-graph,
nd), and skills (vlt-skill). One file, one tested combo, no version skew.

## Schema

```json
{
  "schema": 1,
  "channel": "stable",
  "updated": "YYYY-MM-DD",
  "tools": {
    "<name>": {"repo": "<owner/repo>", "version": "vX.Y.Z"}
  },
  "plugins": {
    "<name>": {"marketplace": "<owner/repo>", "version": "X.Y.Z"}
  },
  "skills": {
    "<name>": {"repo": "<owner/repo>", "version": "vX.Y.Z"}
  }
}
```

- `tools` entries point at GitHub releases. `version` is the release tag
  (v-prefixed). Binaries are downloaded as `<name>_<X.Y.Z>_<os>_<arch>`
  archives and verified against the release's `checksums.txt`.
- `plugins` entries point at GitHub-source Claude Code marketplaces. `version`
  matches the plugin manifest (`.claude-plugin/plugin.json`) committed in that
  repository -- no v prefix, because plugin manifests don't use one.
- `skills` entries point at a repository tag whose skill `SKILL.md` frontmatter
  carries the matching `version:`.

## How a combo gets stamped

Every change to `channel/**` runs the `channel-verify` CI workflow, which
smoke-tests the pinned combination before it lands:

1. `stable.json` must parse and declare `schema: 1`.
2. Every `tools` entry must have a published GitHub release at the pinned tag,
   with a downloadable `linux_amd64` asset and `checksums.txt`. The asset is
   downloaded, sha256-verified, and executed -- `<tool> --version` must report
   the pinned version.
3. Every `plugins` entry must match the plugin manifest committed at HEAD of
   the marketplace repository's default branch.
4. Every `skills` entry must match the `version:` frontmatter of the skill's
   `SKILL.md` at the pinned tag.

A combo that fails any check never reaches `main`. A combo on `main` is, by
construction, a combination that CI installed and verified.

## The already-released rule

`stable.json` may only reference versions that are ALREADY released and
published. The manifest pins history; it never announces the future. The
release order is therefore:

1. Cut and publish the release in the tool/plugin repository (release gates in
   each repo enforce their own version sync).
2. Update `stable.json` to point at the new version.
3. CI verifies the combo; merge.

If you update `stable.json` first, CI fails -- the release tag does not exist
yet. That failure is the system working as intended.

## Rollback

Because the channel is just a file in git, rollback is a ref, not a procedure:

```bash
pvg update --to <git-ref>
```

This fetches `channel/stable.json` at `<git-ref>` (a commit SHA, tag, or
branch of this repository) and converges the machine onto that combination.
Every combo that was ever on `main` passed the same CI smoke test, so any
historical ref is a known-good target.

To hold a machine at a specific combo and suppress update nudges:

```bash
pvg update --pin <git-ref>
```

Updates are pin + nudge, never silent: `pvg` notices when `main` has moved
past the installed combo and tells you, but nothing changes until you run
`pvg update` yourself.
