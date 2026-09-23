# Alcedema CUPS Web

Development, merge requests, CI and releases: <https://gitlab.com/Alcedema/cups-web>.
GitHub mirror: <https://github.com/Alcedema/cups-web>, a direct fork of
<https://github.com/hanxi/cups-web>. This fork starts at upstream v0.2.15 and
retains the upstream Git history, MIT licence and author attribution.

## Language and accounts

Set `DEFAULT_LANGUAGE=en` (the default) or `DEFAULT_LANGUAGE=zh-CN` in the service
environment. Invalid values log a warning and fall back to English. Restart the
service after changing the environment. Browser language is never consulted.

Account settings is available in desktop navigation and the mobile menu. Select
**Use server default**, **English**, or **简体中文**, then save. Preferences are stored
on the account in SQLite, apply immediately, and survive logout, restart and
switching browsers. Guests always use the server default and cannot edit the
shared account. Printer names, document content, protocol values and unrecognised
external diagnostics remain unchanged.

Password changes require the current password and matching new passwords. The
minimum is eight Unicode characters and the maximum is 72 UTF-8 bytes. A successful
change signs out the current browser session. Roles and language are preserved.

The additive `users.language` migration uses an empty string to inherit the
server default. It does not reset passwords, session-signing keys or history.

### API

* `GET /api/public-settings` adds `defaultLanguage` and `supportedLanguages`.
* `GET /api/me` adds saved `language` and resolved `effectiveLanguage`.
* `PUT /api/me/preferences`: `{"language":""}`, `{"language":"en"}` or
  `{"language":"zh-CN"}`.
* `PUT /api/me/password`: `{"currentPassword":"…","newPassword":"…"}`.

Both writes require a browser session and a valid CSRF cookie/header pair.
Guest and API-key requests are rejected. Existing `error` fields and HTTP status
codes are retained; application errors additionally include stable `code` and
`params` fields where available. Password fields are never logged.

Translation catalogues are `frontend/src/locales/en.json` (British English) and
`zh-CN.json`. Existing messages use content-derived stable keys to avoid changing
them on every upstream merge. New account messages use descriptive keys. Both
catalogues must have identical keys and interpolation parameters. The translation
check also rejects untranslated interface strings in source. Backend error
recognition lives in `internal/messages/errors.json`; external diagnostics are
not edited. Nuxt UI, document language and date/number formatting follow the
account language.

## Development and releases

Tooling is pinned in `.gitlab-ci.yml`, `go.mod`, `frontend/package.json` and
`frontend/package-lock.json`. Use npm 11.6.2 with Node 22.22.0:

```sh
cd frontend
npx --yes npm@11.6.2 ci
npm run check:translations
npm test
npm run build
cd ..
go test ./...
go vet ./...
python scripts/test_upstream_sync.py
mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags '-s -w -X main.Version=dev' \
  -o bin/cups-web-linux-amd64 ./cmd/server
```

The frontend must be built before Go because it is embedded in the binary.
Follow the [release contract](release-contract.md) and record the version decision
in `release.json`. Independent release tags `alcedema-vX.Y.Z` publish the Linux
amd64 binary, `RELEASE.json`, `SHA256SUMS` and `SOURCE_COMMIT` in GitLab's package registry, linked from the GitLab release.
The footer identifies the Alcedema fork, links issue reports to GitLab, and credits
hanxi/cups-web. Its MIT licence link serves the complete original copyright and
licence notice embedded in the binary. Releases also include `LICENSE.txt`. The
version is displayed in the footer. CI release publication
uses its short-lived job token, never the synchronization credential.

## Upstream maintenance and mirroring

Develop on GitLab `main`. GitLab pushes branches and tags to the direct GitHub
fork using a repository-specific SSH deploy key. **Keep divergent refs** is
enabled: GitHub-only divergent work causes a mirror failure and needs manual
resolution. GitHub Actions are disabled. Issues, merge requests and release assets
stay on GitLab; they are not Git objects and are not mirrored.

A daily pipeline runs at **07:00 UTC** on protected `main`. It fetches stable
releases directly from hanxi/cups-web, preserves release tags and updates
`upstream/stable`. A new release is merged in a disposable worktree. A clean
merge is pushed to `integrate/vX.Y.Z` with one merge request; conflicts are
reported in one issue per release with filenames and commit identity. Repeated
runs update the existing report. Closed merge requests are not silently reopened.
The sync job never changes `main`, merges a merge request, deploys a release, or
executes fetched upstream code. Candidate branches are built in separate jobs
without synchronization credentials. The next daily run refreshes the MR's
pipeline validation status. Review the complete pipeline before merging.

`UPSTREAM_SYNC_TOKEN` is a project-only Developer token with `api` and
`write_repository`, masked, hidden and protected, scoped to environment
`upstream-sync`. Only the trusted scheduled job declares that environment.
Merge request pipelines cannot access protected variables. Main only permits
maintainers to push or merge; the sync token cannot do either. Rotate the token before its configured expiry; keep the operational schedule private.

## Native LXC installation

Deploy the Linux binary under a dedicated service account, with persistent state
and service configuration outside the source checkout. CUPS and CUPS Web are
separate services; retain the configured addresses and ports during an upgrade.
Keep actual hostnames, container IDs, filesystem inventory and backup locations
in a private operational runbook. Deployment is currently manual.

Before replacing a production binary, validate the candidate with disposable
state, including a legacy-database migration and rollback. Stop CUPS Web, back up
its entire persistent directory (including SQLite sidecar files), environment,
systemd configuration, symlink and previous binary. Install the verified binary,
set `DEFAULT_LANGUAGE=en`, and restart CUPS Web. Verify users, printer discovery,
conversion and both ports without submitting a print job. If validation fails,
stop CUPS Web and restore the previous binary, service configuration and complete
pre-upgrade state before restarting. Do not alter the CUPS scheduler or queues.
