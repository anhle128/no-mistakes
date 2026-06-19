# Quickstart: Validate Current Worktree YOLO Mode

This quickstart is for implementers after `/speckit-tasks` produces tasks and
the feature is implemented.

## Build

```sh
go build ./cmd/no-mistakes
```

## Unit Test Targets

Run focused packages first while iterating:

```sh
go test ./internal/cli ./internal/daemon ./internal/db ./internal/git ./internal/ipc ./internal/tui
go test ./internal/pipeline ./internal/pipeline/steps
```

Then run the required repository checks:

```sh
go test -race ./...
make lint
```

Run tagged e2e coverage after the unit suite is green:

```sh
make e2e
```

## Manual Smoke: AXI Current Mode

1. Create or enter a clean non-default git worktree for an initialized
   no-mistakes repo.
2. Confirm the checkout is clean:

   ```sh
   git status --short
   ```

3. Start a current-worktree run:

   ```sh
   no-mistakes axi run --intent "validate current-worktree execution" --no-worktree --yolo
   ```

4. Confirm:

   - No new run directory is created under `$NM_HOME/worktrees/<repo>/<run>`.
   - Run metadata reports `worktree_mode=current`.
   - The execution directory equals the current git worktree root.
   - AXI output includes `work_dir_label` and `current_worktree_warning`.
   - Any automated fix commits remain in the current checkout.

## Manual Smoke: Root Current-Mode Guidance

From a clean non-default branch:

```sh
no-mistakes --no-worktree --yolo
```

Confirm the root command fails before run creation with recovery guidance to use
`no-mistakes axi run --intent "..." --no-worktree`; the root command has no
`--intent` flag and must not start with empty, generic, or inferred intent.

## Rejection Checks

Each case must reject before pipeline execution:

```sh
# default branch
git switch main
no-mistakes axi run --intent "validate current-worktree execution" --no-worktree --yolo

# detached head
git checkout --detach HEAD
no-mistakes axi run --intent "validate current-worktree execution" --no-worktree --yolo

# tracked dirty file
echo dirty >> README.md
no-mistakes axi run --intent "validate current-worktree execution" --no-worktree --yolo

# untracked non-ignored file
touch untracked-current-mode-check.txt
no-mistakes axi run --intent "validate current-worktree execution" --no-worktree --yolo
```

Ignored-only files should not block:

```sh
mkdir -p tmp
printf 'tmp/\n' >> .git/info/exclude
touch tmp/ignored-only
no-mistakes axi run --intent "validate current-worktree execution" --no-worktree --yolo
```

## Rendering Checks

Inspect all relevant surfaces:

```sh
no-mistakes status
no-mistakes runs
no-mistakes axi status
no-mistakes attach --run <run-id>
```

Confirm:

- Current-mode runs use the plain label "uses this checkout".
- Isolated runs use the plain label "disposable no-mistakes checkout".
- Full absolute paths are absent from normal output unless verbose/debug output
  is explicitly requested.
- Failed, cancelled, stale-recovered, setup-failed, and missing-base states show
  incomplete/degraded evidence when appropriate.

## Isolated Regression Checks

Run existing flows without `--no-worktree`:

```sh
no-mistakes axi run --intent "isolated regression" --yes
git push no-mistakes HEAD
no-mistakes -y
```

Confirm disposable worktree creation and cleanup still match existing behavior.
