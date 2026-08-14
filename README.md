# Abstrax Deploy Plugin

Official Abstrax CLI plugin for **zero-downtime GitHub deployments**.

Binary: `abstrax-deploy` → `abstrax deploy …`  
Trust level: `official`  
OS: Debian/Ubuntu and RHEL-family (Rocky/Alma/RHEL). Nginx only.

## What it does

Projects created with `abstrax project add` get infrastructure (user, path, nginx, runtime). This plugin deploys **application code** into a Capistrano/Deployer-style layout:

```text
{project.path}/
  deploy.json
  releases/{YYYYMMDDHHMMSS}/
  current -> releases/{id}
  shared/
```

Each `deploy now`:

1. Shallow-clones the configured repository into a new release directory
2. Writes `.abstrax-release.json`, then **deletes `.git`** (immutable tree)
3. Symlinks configured shared paths
4. Runs hooks (`after_clone` → `before_activate`)
5. Health-checks that the release and `public_dir` exist
6. Atomically flips `current`
7. Runs `after_activate` hooks
8. Prunes old releases (`keep_releases`, default 5)

There is **no** in-place `git pull`, no bare/mirror cache in v1, no PHP-FPM reload, and no automatic supervisor restarts. Restart services only via hooks if you need to.

Nginx PHP blocks in Abstrax already use `$realpath_root`, so flipping `current` does not require reloading PHP-FPM.

## Install (local)

```bash
cd plugins/deploy
make build
sudo cp bin/abstrax-deploy /usr/local/lib/abstrax/plugins/
abstrax deploy version
abstrax-deploy plugin metadata | jq .
```

## Happy path

```bash
# Project already exists (infra only)
sudo abstrax project add example.com --domains=example.com --php --public-dir=public

# One-shot setup (init + key + optional first deploy)
sudo abstrax deploy setup example.com \
  --repository=git@github.com:acme/app.git \
  --branch=main \
  --preset=laravel \
  --no-first-deploy

# Add the printed public key as a GitHub Deploy Key (read-only)
sudo abstrax deploy key example.com --show

# Deploy
sudo abstrax deploy now example.com --yes

# Inspect
abstrax deploy status example.com
abstrax deploy list example.com
```

## Commands

| Command | Root? | Description |
|---------|-------|-------------|
| `deploy setup <project>` | yes | Guided init + configure + key + optional first deploy |
| `deploy init <project>` | yes | Scaffold dirs + `deploy.json` + set web root to `current/{public_dir}` |
| `deploy configure <project>` | yes (writes) | Show or update config |
| `deploy key <project>` | yes (writes) | Create/rotate deploy key; `--show` / `--fingerprint` are read-only |
| `deploy now <project>` | yes | Full release pipeline |
| `deploy rollback <project> [id]` | yes | Flip `current`; re-runs `after_activate` hooks |
| `deploy list <project>` | no* | List releases |
| `deploy status <project>` | no* | Config + current release |
| `deploy hooks <project> [phase]` | yes (writes) | List/set/append/clear hooks |

\*Read-only when filesystem permissions allow.

Globals: `--json`, `--json-stream`, `--yes`, `--dry-run`, `--verbose`, `--quiet`, `--no-color`.

There is **no** `deploy release` command.

## Presets

| Preset | Shared | Default hooks |
|--------|--------|---------------|
| `laravel` | `.env`, `storage` | `composer install --no-dev --optimize-autoloader`; `$ABSTRAX_CLI_PHP artisan migrate --force` |
| `node` | none | `npm ci && npm run build` (app must define a `build` script) |
| `ruby` | none | `bundle install --deployment --without development test` |
| `static` | none | none |
| `none` | none | none |

Laravel includes migrate. No preset restarts services. For Node/Ruby workers, add restarts to `after_activate`, for example:

```bash
sudo abstrax deploy hooks example.com after_activate \
  --append='abstrax project service restart example.com example-worker --yes'
```

## GitHub deploy keys

1. `sudo abstrax deploy key <project>` creates `~/.ssh/abstrax_deploy_<project>` for the project user
2. Print the pubkey: `abstrax deploy key <project> --show`
3. GitHub → repository **Settings → Deploy keys → Add deploy key** (read-only is enough)
4. `known_hosts` is updated for `github.com` automatically when the key is created

Rotate with `sudo abstrax deploy key <project> --rotate --yes` and update GitHub.

## Shallow clone behaviour

- Branches: `git clone --depth 1 --branch <ref>`
- Tags: shallow clone by tag name, with fetch fallback
- SHAs: `git init` + `git fetch --depth 1` (deeper fetch if needed)
- After metadata is written, `.git` is removed so releases are plain trees

Provider seam: `provider: "github"` in v1. Other hosts can be added later without rewriting the release engine.

## Debian/Ubuntu and RHEL

This plugin does not hard-code distro nginx/PHP socket paths. It uses Abstrax APIs:

- `abstrax project inspect <project> --json`
- `abstrax project modify <project> --public-dir=current/{public_dir} --json`

`$ABSTRAX_CLI_PHP` resolves versioned CLI binaries when possible (`php8.5` on Debian/Ubuntu, `/opt/remi/php85/root/usr/bin/php` on Remi/RHEL-family).

## Hooks and environment

Hooks are shell strings run with `bash -lc` as the project user when isolated. **cwd** is the release path.

Injected env vars:

- `ABSTRAX_PROJECT`, `ABSTRAX_PROJECT_PATH`, `ABSTRAX_RELEASE_PATH`
- `ABSTRAX_CURRENT_PATH`, `ABSTRAX_SHARED_PATH`
- `ABSTRAX_BRANCH`, `ABSTRAX_REF`, `ABSTRAX_RELEASE_ID`
- `ABSTRAX_CLI_PHP` (PHP runtimes)

Rollback policy: re-runs `after_activate` so user-defined restart hooks still apply.

## Agent / JSON output

```bash
sudo abstrax deploy now example.com --yes --json-stream
```

NDJSON progress lines (`type=progress`) then a final `type=result` line, matching Abstrax core. Use `--json` for a single result object. Do not combine `--json` and `--json-stream`.

Conceptual future agent actions: `deploy.setup`, `deploy.configure`, `deploy.now`, `deploy.rollback`, `deploy.status`.

## Development

```bash
make test
make build
make vet
```

## Non-goals (v1)

- No Saturn-style in-place git deploys
- No `--migrate` from legacy trees
- No bare git cache
- No Apache
- No automatic PHP-FPM / supervisor restarts
- No web UI / TUI
