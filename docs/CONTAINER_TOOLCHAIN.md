# Running Paivot inside a toolchain container

Some projects run their build, lint, and test toolchain inside a container --
for example Elixir/`mix` inside `docker compose`. Paivot's vault-backed lints
and tests shell out to `pvg nd list --json`, `pvg issues list --json`, and
`nd ... --json`. If `pvg` and `nd` are installed only on the host, those calls
die with `:enoent` *inside* the container, and the affected lint/test suites
fail for the wrong reason. (This has masked a genuine product finding behind
sixteen dead lint tests.)

The fix is to make `pvg` and `nd` resolvable on `PATH` inside the container, and
to make the live nd vault reachable from inside the container. Both `pvg` and
`nd` are static Go binaries, so the same executable runs unmodified inside a
glibc Linux container as long as the binary matches the container's CPU
architecture.

## Two supported wirings

### A. Install into the toolchain image at build time (recommended)

This is portable, reproducible, and architecture-correct: the image always
carries binaries built for its own architecture, and they travel with the image
wherever it runs. Fetch `pvg` + `nd` for the container's architecture during the
image build with the channel installer:

```dockerfile
# in the toolchain image build
RUN curl -fsSL https://raw.githubusercontent.com/paivot-ai/pvg/main/install.sh | sh
```

The channel installer pins to the channel-stable versions, which keeps the
container's `pvg`/`nd` in sync with the versions the project expects.

If the build environment cannot reach GitHub, download the
`*_linux_<arch>.tar.gz` release assets for `pvg` and `nd` (matching the
container architecture -- `amd64` or `arm64`), then `COPY` the extracted
binaries into the image:

```dockerfile
COPY pvg /usr/local/bin/pvg
COPY nd  /usr/local/bin/nd
RUN chmod +x /usr/local/bin/pvg /usr/local/bin/nd
```

### B. Read-only bind-mount the host binaries (quick local dev)

For fast local iteration you can mount the host's binaries into the container
instead of rebuilding the image:

```yaml
services:
  toolchain:
    volumes:
      - .:/app                                       # repo, including .git/ (carries the live nd vault)
      - /usr/local/bin/pvg:/usr/local/bin/pvg:ro
      - /usr/local/bin/nd:/usr/local/bin/nd:ro
```

**Architecture caveat:** bind-mounting only works when the host and the
container share the same OS/architecture (for example a `linux/amd64` host
running a `linux/amd64` container). On a macOS/ARM host running a `linux/amd64`
container, the host binary will **not** run inside the container -- it is the
wrong architecture. Use wiring (A) in that case.

## Vault resolution inside the container

Having the binaries on `PATH` is necessary but not sufficient. The live nd vault
lives under the git common dir (for example `.git/paivot/nd-vault/`, the shared
vault referenced by `.vault/.nd-shared.yaml`). It is reachable inside the
container **only if** both of the following hold:

1. The container bind-mounts the repo root **including `.git/`** (the snippet in
   wiring B mounts `.:/app`, which carries `.git/`). The live vault is gitignored
   and is not part of git history, so it travels with the working tree, not with
   a fresh clone.
2. `.vault/.nd-shared.yaml` is committed, so `nd` resolves the shared vault under
   the git common dir rather than looking for a branch-local `.vault`.

This is the same one-time setup the PM-isolation flow already documents -- see
[PM Isolation in commands/piv-loop.md](../commands/piv-loop.md) and
[docs/LIVE_SOR.md](LIVE_SOR.md) for the full rationale. In short:

```bash
pvg nd root --ensure
git add .vault/.nd-shared.yaml
git commit -m "chore(paivot): share live nd vault across worktrees"
```

With both wirings in place, `pvg nd list --json`, `pvg issues list --json`, and
`pvg lint` all run from inside the container against the same live vault the host
uses.

## Acceptance check

A fresh project can self-verify the wiring. From inside the container, all three
of these should succeed against the project vault:

```bash
pvg version
nd --version
pvg nd list --json
```

If `pvg version` or `nd --version` fail, the binaries are not on `PATH` (or are
the wrong architecture -- revisit the wiring choice). If those succeed but
`pvg nd list --json` fails to find the vault, the repo root including `.git/` is
not mounted, or `.vault/.nd-shared.yaml` is not committed.
