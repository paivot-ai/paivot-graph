# Per-Story Environment Isolation

Paivot runs developers in parallel: a wave of stories, each in its own git
worktree on its own branch. When those stories' integration tests need shared
infrastructure -- a database, a message broker, a cluster -- the worktrees alone
do not isolate them. They still all point at the same Postgres, the same broker,
the same fixed host port, and the wave serializes on that contention (or worse,
two stories corrupt each other's state).

This convention lets a project give each story its OWN isolated environment, so a
parallel wave actually runs in parallel. It is **opt-in** and
**harness-agnostic**.

## Principle: Paivot owns the contract, the project owns the engine

The methodology never names Docker, Kubernetes, or ports -- those are
project-specific engines. Paivot defines a small contract; the project supplies
an executable that satisfies it however it likes (compose project, k8s namespace,
a fresh schema, a remote tenant). A project with no shared infrastructure ships
nothing and nothing changes.

## The contract: `.paivot/envr`

A project opts in by placing an executable script at `.paivot/envr` -- a sibling
of the existing `.paivot/config.yaml`. It is called `envr` (not `env`) on
purpose: `.env` is too common and would be confused with a dotenv file.

The token is always the **story id**. Paivot already guarantees one story id per
concurrent worktree, so two concurrent environments never share a token.

| Invocation | Contract |
|------------|----------|
| `.paivot/envr up <token>` | Provision an isolated environment keyed by `<token>`. Print connection details to **stdout** as `KEY=VALUE` lines (one per line; `#`-prefixed comment lines allowed). **Idempotent** -- re-running for the same token is safe and prints the same details. Exit `0` on success, non-zero on failure. |
| `.paivot/envr down <token>` | Tear down the environment for `<token>`. **Idempotent** -- safe if it is already gone. Exit `0`. |

**stdout is the interface.** `up` emits the environment as `KEY=VALUE` lines; the
dispatcher captures that stdout verbatim and injects it into the developer
prompt. Put only connection details on stdout. Send logs/diagnostics to stderr.

```
# example .paivot/envr up <token> output
# postgres provisioned for this story
DATABASE_URL=postgres://app:app@127.0.0.1:54113/app
REDIS_URL=redis://127.0.0.1:54114/0
```

**Graceful degradation.** If `.paivot/envr` does not exist or is not executable,
the methodology behaves exactly as it does today: the dispatcher discovers shared
infrastructure once and injects one connection string into all developers (see
[CONTAINER_TOOLCHAIN.md](CONTAINER_TOOLCHAIN.md) and the Infrastructure Context
section of [../commands/piv-loop.md](../commands/piv-loop.md)). Nothing about the
contract requires `pvg` -- it is plain prompt-driven methodology.

## Lifecycle: bracketed around the worktree lifecycle

The dispatcher already creates one worktree per story and removes it on close
(see [PARALLEL_DEV_WORKTREES.md](PARALLEL_DEV_WORKTREES.md)). Environment
provisioning brackets that SAME lifecycle:

1. **On prepare** -- immediately after `pvg worktree add` for the story, if
   `.paivot/envr` is executable, the dispatcher runs `.paivot/envr up <story-id>`,
   captures its `KEY=VALUE` stdout, and injects it into THAT developer's prompt as
   the story's **ISOLATED INFRASTRUCTURE**. This REPLACES the single shared
   connection string for that developer.
2. **On close** -- accepted, abandoned, or otherwise cleaned up -- wherever the
   dispatcher runs `pvg worktree remove` for the story, it also runs
   `.paivot/envr down <story-id>`. Because `down` is idempotent, this is safe even
   if the environment was never brought up.

```bash
# provision (dispatcher, right after `pvg worktree add`)
[ -x .paivot/envr ] && .paivot/envr up STORY_ID      # stdout -> developer prompt

# tear down (dispatcher, alongside `pvg worktree remove`)
[ -x .paivot/envr ] && .paivot/envr down STORY_ID
```

Because the token is the story id, the dispatcher never runs two concurrent
stories on the same token -- that uniqueness is already guaranteed by one story
id per worktree.

## Harness-agnostic

The `.paivot/envr` contract is identical across all Paivot variants (graph,
codex, opencode, pi). Only the dispatcher/developer **prompt wording** adapts per
harness -- the script, the `up`/`down` verbs, the token-is-story-id rule, and the
`KEY=VALUE` stdout are the same everywhere. The contract does not depend on `pvg`
or on any harness-specific tooling; a project's `.paivot/envr` works unchanged no
matter which Paivot harness drives it.

## Worked example A: Kubernetes namespace per story

A namespace is a natural per-story boundary -- create one on `up`, delete it on
`down`. `kubectl` calls are idempotent enough that re-running `up` for the same
token is safe.

```sh
#!/usr/bin/env sh
# .paivot/envr -- Kubernetes namespace per story
set -eu

cmd=$1
token=$2
ns="pv-${token}"

case "$cmd" in
  up)
    # idempotent: create only if absent
    kubectl get namespace "$ns" >/dev/null 2>&1 \
      || kubectl create namespace "$ns" >&2
    # (apply your manifests into -n "$ns" here, then wait for readiness)
    echo "KUBE_NAMESPACE=${ns}"
    ;;
  down)
    # idempotent: ignore "not found"
    kubectl delete namespace "$ns" --ignore-not-found >&2
    ;;
  *)
    echo "usage: envr up|down <token>" >&2
    exit 2
    ;;
esac
```

The developer receives `KUBE_NAMESPACE=pv-<story-id>` and runs its integration
tests against that namespace -- no two stories share one.

## Worked example B: docker-compose with dynamic host ports

`COMPOSE_PROJECT_NAME=<token>` namespaces container, network, and volume names so
two stories' stacks coexist -- but it does **NOT** namespace published host
ports. If `docker-compose.yml` pins `ports: ["5432:5432"]`, both stories still
fight over host port 5432 and the second `up` fails. The fix: do NOT pin host
ports in compose; let Docker assign ephemeral ones, then DISCOVER the assigned
port with `docker compose port` and emit it as `KEY=VALUE`.

In `docker-compose.yml`, expose without a fixed host port:

```yaml
services:
  db:
    image: postgres:16
    ports:
      - "5432"          # host port assigned dynamically -- do NOT pin "5432:5432"
```

```sh
#!/usr/bin/env sh
# .paivot/envr -- docker-compose, project-per-story, dynamic host ports
set -eu

cmd=$1
token=$2
export COMPOSE_PROJECT_NAME="$token"   # isolates container/network/volume names

case "$cmd" in
  up)
    # idempotent: compose up is safe to re-run for the same project
    docker compose up -d >&2
    # COMPOSE_PROJECT_NAME alone does NOT isolate host ports -- discover them
    hostport=$(docker compose port db 5432 | cut -d: -f2)
    echo "DATABASE_URL=postgres://postgres:postgres@127.0.0.1:${hostport}/postgres"
    ;;
  down)
    # idempotent: -v removes volumes; no error if nothing is up
    docker compose down -v >&2
    ;;
  *)
    echo "usage: envr up|down <token>" >&2
    exit 2
    ;;
esac
```

Each story gets its own compose project AND its own discovered host port, so
parallel `up` calls never collide.

## Testing it

1. Drop the Kubernetes example above into `.paivot/envr` and `chmod +x` it (point
   it at a throwaway cluster -- e.g. kind or minikube).
2. Run a two-story wave so the dispatcher prepares both worktrees and calls
   `.paivot/envr up <story-id>` for each.
3. Confirm each developer prompt carries a distinct
   `KUBE_NAMESPACE=pv-<story-id>`, and that `kubectl get namespace` shows one
   `pv-*` namespace per in-flight story.
4. Let the stories close and confirm `kubectl get namespace` shows the `pv-*`
   namespaces removed (the dispatcher ran `.paivot/envr down <story-id>`
   alongside worktree removal).

If both developers land in the same namespace, the token is not being threaded as
the story id; if a namespace lingers after close, `down` is not idempotent or the
teardown was skipped.
