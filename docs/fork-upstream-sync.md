# Fork upstream sync

Operator notes for `adityad1997/warmbly`. This file is fork-only; it is not part of the customer docs site.

## What runs automatically

The **Sync upstream release** workflow (`.github/workflows/sync-upstream-release.yml`) every four hours (and on manual **Run workflow**):

1. Reads the latest stable (non-draft, non-prerelease) release from `warmbly/warmbly`
2. Skips if this fork already has that tag
3. Merges that upstream tag into `main` (keeping fork patches)
4. On success: pushes `main` and the same `vX.Y.Z` tag
5. Existing `release.yml` then builds `ghcr.io/adityad1997/warmbly/*` and creates the GitHub Release

The VPS at `/var/www/warmbly` is pointed at that registry and `RELEASES_GITHUB_REPO=adityad1997/warmbly`. Apply a new release with the admin **Update Warmbly** dialog (image mode). Host cron auto-apply is not installed yet.

## When a conflict issue opens

The workflow opens an issue titled `Upstream sync conflict: vX.Y.Z` and does not push a tag.

```bash
cd /path/to/warmbly   # this fork clone
git remote add upstream https://github.com/warmbly/warmbly.git   # once
git fetch upstream
git checkout main
git merge vX.Y.Z                  # or: git merge upstream/main
# resolve conflicts; keep fork-only patches
git add -A && git commit
git push origin main
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

Wait for **Release** to finish on GitHub, confirm packages for any new image names are still Public, then use **Update Warmbly** on the VPS (or set `WARMBLY_TAG` and `docker compose pull && up -d`).

## Manual sync without waiting for the schedule

Actions → **Sync upstream release** → **Run workflow**.

If the latest upstream tag already exists on the fork, the run exits successfully with a skip message.
