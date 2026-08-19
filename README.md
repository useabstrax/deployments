# Abstrax Deploy Plugin

Zero-downtime GitHub deployments for Abstrax projects.

Binary: `abstrax-deploy` → `abstrax deploy …`  
Trust level: `official`  
OS: Debian/Ubuntu and RHEL-family. Nginx only.

Full user docs: [useabstrax.com/docs/plugins/official/deploy](https://useabstrax.com/docs/plugins/official/deploy)

## What it does

`abstrax project add` creates infrastructure (user, path, nginx, runtime). This plugin deploys application code into:

```text
{project.path}/
  deploy.json
  releases/{YYYYMMDDHHMMSS}/
  current -> releases/{id}
  shared/
```

Each `deploy now` shallow-clones the repo, strips `.git`, links shared paths, runs hooks, health-checks, flips `current`, then prunes old releases. There is no in-place `git pull` and no automatic PHP-FPM or supervisor restart.

## Install

```bash
sudo abstrax plugin install deploy
abstrax deploy version
```

Local build:

```bash
go build -o bin/abstrax-deploy ./cmd/abstrax-deploy
sudo cp bin/abstrax-deploy /usr/local/lib/abstrax/plugins/
```

Tagged releases (`v*`) publish linux-amd64 / linux-arm64 archives and `plugin-manifest.json`.

## Quick start

```bash
sudo abstrax project add example.com --domains=example.com --php --public-dir=public

sudo abstrax deploy setup example.com \
  --repository=git@github.com:acme/app.git \
  --branch=main \
  --preset=laravel \
  --no-first-deploy

abstrax deploy key example.com --show   # add as a GitHub deploy key (read-only)
sudo abstrax deploy now example.com --yes

abstrax deploy status example.com
abstrax deploy list example.com
```

## Commands

| Command | Root? | Description |
|---------|-------|-------------|
| `deploy setup <project>` | yes | Init + configure + key + optional first deploy |
| `deploy init <project>` | yes | Scaffold dirs + `deploy.json`; set web root to `current/{public_dir}` |
| `deploy configure <project>` | yes (writes) | Show or update config |
| `deploy key <project>` | yes (writes) | Create/rotate deploy key; `--show` / `--fingerprint` are read-only |
| `deploy now <project>` | yes | Full release pipeline |
| `deploy rollback <project> [id]` | yes | Flip `current`; re-runs `after_activate` |
| `deploy list <project>` | no* | List releases |
| `deploy status <project>` | no* | Config + current release |
| `deploy hooks <project> [phase]` | yes (writes) | List/set/append/clear hooks |

\*Read-only when filesystem permissions allow.

Globals: `--json`, `--json-stream`, `--yes`, `--dry-run`, `--verbose`, `--quiet`, `--no-color`.

`--ref` is a branch, `tags/v1.0.0`, or SHA. A bare name such as `v1.2.3` is treated as a branch.

## Presets

| Preset | `public_dir` | Shared | Default hooks |
|--------|--------------|--------|---------------|
| `laravel` | `public` | `.env`, `storage` | Composer install via `abstrax composer run`; `$ABSTRAX_CLI_PHP artisan migrate --force` |
| `node` | `.` | none | `npm ci && npm run build` |
| `ruby` | `.` | none | `bundle install --deployment --without development test` |
| `static` / `none` | `.` | none | none |

Laravel setup/init/deploy also scaffolds `shared/storage` and a minimal `shared/.env` (generated `APP_KEY`) when missing or empty. Install the [Composer plugin](https://useabstrax.com/docs/plugins/official/composer) for Laravel: `sudo abstrax plugin install composer && sudo abstrax composer setup`.

Hooks that call `abstrax` run as root so plugins are found; pass `--project` / `--path` so Composer still drops to the project user. Other hooks run as the project user.

No preset restarts services. Add those to `after_activate`.

## Development

```bash
go test -race ./...
go vet ./...
go build -o bin/abstrax-deploy ./cmd/abstrax-deploy
```
