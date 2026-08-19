# Changelog

All notable changes to the Abstrax Deploy plugin are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-08-19

### Added

- **Laravel shared scaffold** - On setup/init/deploy, ensures `shared/storage/{app,framework,logs}/…` and a minimal `shared/.env` (generated `APP_KEY`, file session/cache drivers). Non-empty `.env` files are never overwritten.
- **Composer plugin hooks** - Laravel preset `after_clone` uses `abstrax composer run --project=… --path=… install …`. Setup/init/configure warn with a copyable `sudo abstrax plugin install composer && sudo abstrax composer setup` when the Composer plugin is missing.
- **Action IDs** - Plugin metadata commands include `action` values such as `plugin.deploy.now` for Abstrax `--action` dispatch.

### Changed

- **Help and usage** - Usage text shows `abstrax deploy …` instead of the `abstrax-deploy` binary name.

### Fixed

- **Release ownership before hooks** - After clone and shared linking, the release (and `shared/`) are chowned to the project user before hooks run, so `composer install` and similar can create directories like `vendor/`.
- **Hook error output** - Failed hooks now include both stderr and stdout (tail), so Artisan errors are visible after Composer progress.
- **Setup scaffold ownership** - `deploy setup` chowns `deploy.json`, `releases/`, and `shared/` to the project user.

## [0.1.0] - 2026-08-14

First release of the official Abstrax Deploy plugin (`abstrax-deploy` → `abstrax deploy …`).

### Added

- **Zero-downtime layout** - Per-project `releases/`, atomic `current` symlink, `shared/`, and plugin-owned `deploy.json`.
- **`deploy setup`** - Guided one-shot init + configure + deploy key + optional first deploy (`--repository`, `--branch`, `--preset`, `--public-dir`, `--keep`, `--no-first-deploy`, `--yes`).
- **`deploy init`** - Scaffold directories and `deploy.json`; set project web root to `current/{public_dir}` via Abstrax APIs.
- **`deploy configure`** - Show or update repository, branch, preset, shared paths, public dir, and keep count.
- **`deploy key`** - Create GitHub deploy keys for the project user; `--show`, `--fingerprint`, and `--rotate --yes`.
- **`deploy now`** - Full pipeline: shallow clone → strip `.git` → shared links → hooks → health check → activate → prune.
- **`deploy rollback`** - Point `current` at a previous (or explicit) release; re-runs `after_activate` hooks.
- **`deploy list` / `deploy status`** - Release listing and status (readable without root when permissions allow).
- **`deploy hooks`** - List, `--set`, `--append`, or `--clear` for `after_clone`, `before_activate`, and `after_activate`.
- **Presets** - `laravel` (composer + migrate), `node`, `ruby`, `static`, and `none`.
- **Git strategy** - Fresh shallow clone per release (branch/tag/SHA with deeper fetch fallback); `.git` removed after metadata is written. No bare mirror in v1.
- **GitHub provider seam** - v1 copy and key flow are GitHub-oriented; provider interface allows other hosts later.
- **Machine output** - `--json` and `--json-stream` (NDJSON progress + final result) for scripts and automation.
- **Cross-distro support** - Debian/Ubuntu and RHEL-family; uses Abstrax project APIs rather than hard-coded nginx/PHP paths.
- **Plugin metadata** - Protocol v1 `plugin metadata` listing all commands; `requires_abstrax >=0.1.0`.

### Notes

- Mutating commands require root (same spirit as `abstrax project add`).
- No PHP-FPM reload after activate (nginx uses `$realpath_root`).
- No automatic supervisor/service restarts — use hooks only.
- No `deploy release` alias; use `deploy now`.
- Non-goals deferred: in-place git deploys, `--migrate`, bare git cache, Apache, GitLab/Bitbucket-specific UX.
