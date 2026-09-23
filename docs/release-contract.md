# Alcedema release contract

This is the release policy for maintainers, contributors and automation. It is
suitable for publication; personal agent instructions and deployment inventory
belong outside this public checkout. This policy supersedes upstream release
instructions retained in this fork.

## Version identity

Use independent `MAJOR.MINOR.PATCH` application versions and Git tags
`alcedema-vMAJOR.MINOR.PATCH`. The first release using this scheme is **0.3.0**,
following `v0.2.15-alcedema.2`. Do not rename or replace historical releases.
Preserve upstream `v*` tags. Record the incorporated upstream tag and commit
separately; merging an upstream release does not copy its version number.

The tag prefix identifies the fork and is stripped from the displayed version.
Do not use `-alcedema.N` for future stable releases: a hyphen suffix means a
prerelease in [Semantic Versioning](https://semver.org/). Build metadata does not
provide version ordering. Prereleases need a separately reviewed extension to
this contract and the validator; the current publisher accepts stable tags only.

## Choosing the next version

Before each release, the agent or maintainer must inspect the complete change
set since the previous **fork** release, including merged upstream changes.
Choose by compatibility and user impact, not lines changed, commit count or
upstream numbering. Use the highest applicable impact across all changes:

| Impact | Before 1.0 | From 1.0 onwards |
| --- | --- | --- |
| Breaking: incompatible supported behaviour or interface | Next minor, reset patch | Next major, reset minor and patch |
| Feature: compatible new functionality or deprecation | Next minor, reset patch | Next minor, reset patch |
| Fix: compatible bug, security or performance correction | Next patch | Next patch |
| Maintenance: compatible internal, documentation or build change | Next patch if a release is needed | Next patch if a release is needed |

Compatibility includes documented HTTP APIs, authentication, configuration and
environment variables, command-line behaviour, stored data and upgrade paths,
supported deployment platforms and printing workflows. Removing a supported
option, requiring a new manual migration, dropping a platform or changing an API
contract is breaking. An additive, automatic migration preserving data is not
necessarily breaking. A large refactor preserving behaviour can be maintenance.
A security fix still needs a breaking classification if it breaks a supported
contract. Deprecations must be described before later removal.

Before 1.0, breaking releases must still describe the break and upgrade action.
Moving to **1.0.0** is an explicit stability commitment: document the supported
interfaces and compatibility guarantees and obtain maintainer agreement. Do not
graduate solely because a diff is large. After 1.0, each breaking change requires
a major release. Routine version selection is delegated to the releasing agent;
uncertainty about compatibility must be resolved before publication.

Examples: `0.3.0` → `0.3.1` for fixes, `0.4.0` for a feature or pre-1.0 break;
`1.2.3` → `1.2.4` for fixes, `1.3.0` for features, `2.0.0` for a break.
No release is required for every commit. Never move an existing release tag or
replace published files; corrections get a new version.

## Required decision record and release procedure

1. Fetch complete history and tags from the primary GitLab repository and review
   `git log <previous-tag>..HEAD` and the corresponding diff. Check the latest
   fork release, not the latest upstream tag.
2. Update `release.json`: version, previous fork tag, incorporated upstream tag
   and full SHA, change entries (`impact` and `summary`), and the reason for the
   highest-impact classification. Provide `upgradeNotes` for breaking changes.
   `stabilityRationale` is mandatory for an agreed 1.0 graduation. Summaries must
   cover all user-visible changes, limitations and relevant validation results.
3. Run `python scripts/release_policy.py` and the required tests. It prints the
   candidate version but never creates or pushes a tag. `bump-version.sh` is a
   compatibility wrapper for this validation, with no implicit patch default.
4. Commit the reviewed record alongside the changes. Before tagging, fetch tags
   again, check the previous release is still current, and recheck the full diff.
   The release commit must be on reviewed `main`. Create an annotated
   `alcedema-v<version>` tag only when release publication is authorised. A request
   to edit code or policy alone is not a request to publish or deploy.
5. GitLab validates the decision and tag before publication. Translation checks,
   frontend tests/build, Go tests/vet, upstream-sync tests and policy tests must
   all pass. Build the embedded frontend first. Publish the binary, SHA-256,
   source commit, licence and decision record with version-specific release notes.
6. Verify the published source identity and GitHub mirror tag/branch SHAs.
   Deployment is a separate action with its own validation and rollback.

The checked-in initial record is a **candidate**, not a published release. Its
one-time `transition` impact moves the old fork naming scheme to 0.3.0 without
claiming an application feature. Subsequent releases cannot use that exception.

CI enforces recognised impacts, highest-impact arithmetic, required explanations,
tag/version agreement, upstream ancestry and the latest fork predecessor on tag
pipelines. It cannot prove that a natural-language classification is truthful or
that all changes were described. Reviewing the diff remains mandatory. These
checks gate publication of assets; they cannot prevent an authorised person from
creating a Git tag before CI runs. Release tags must not be rewritten after a
failure; fix code/policy in a new commit and choose a fresh version as appropriate.
