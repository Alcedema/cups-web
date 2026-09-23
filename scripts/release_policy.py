"""Validate the recorded release decision. Never create, push or modify a tag."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess

VERSION = r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
FORK_TAG = re.compile(r"alcedema-v" + VERSION)
LEGACY_TAG = re.compile(r"v" + VERSION + r"-alcedema\.([1-9][0-9]*)")
TRANSITION_FROM = "v0.2.15-alcedema.2"
ROOT = Path(__file__).resolve().parent.parent


def version(value):
    if not isinstance(value, str) or not re.fullmatch(VERSION, value):
        raise ValueError("Version must be MAJOR.MINOR.PATCH without a suffix or leading zeros")
    return tuple(map(int, value.split(".")))


def required_text(record, key):
    value = record.get(key)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"Missing non-empty {key}")
    return value


def validate_record(record):
    target = version(record.get("version"))
    previous = required_text(record, "previousTag")
    required_text(record, "rationale")
    if not re.fullmatch("v" + VERSION, required_text(record, "upstreamTag")):
        raise ValueError("upstreamTag must be a stable upstream vMAJOR.MINOR.PATCH tag")
    if not re.fullmatch(r"[0-9a-f]{40}", required_text(record, "upstreamCommit")):
        raise ValueError("upstreamCommit must be a full commit SHA")
    changes = record.get("changes")
    if not isinstance(changes, list) or not changes:
        raise ValueError("At least one classified change is required")
    impacts = set()
    for change in changes:
        if not isinstance(change, dict):
            raise ValueError("Each change must contain impact and summary")
        required_text(change, "summary")
        impact = required_text(change, "impact")
        if impact not in {"breaking", "feature", "fix", "maintenance", "transition", "stable"}:
            raise ValueError(f"Unknown change impact: {impact}")
        impacts.add(impact)
    if "breaking" in impacts:
        required_text(record, "upgradeNotes")
    if previous == TRANSITION_FROM and "transition" in impacts:
        if "stable" in impacts or target != (0, 3, 0):
            raise ValueError("The naming transition must be to 0.3.0")
        return "alcedema-v" + record["version"]
    match = FORK_TAG.fullmatch(previous)
    if not match:
        raise ValueError("previousTag must name the previous Alcedema release")
    major, minor, patch = map(int, match.groups())
    if "transition" in impacts:
        raise ValueError("The transition exception is only available for the first release")
    if "stable" in impacts:
        if major != 0:
            raise ValueError("Stability graduation is only available before 1.0")
        required_text(record, "stabilityRationale")
        expected = (1, 0, 0)
    elif "breaking" in impacts and major > 0:
        expected = (major + 1, 0, 0)
    elif impacts & {"breaking", "feature"}:
        expected = (major, minor + 1, 0)
    else:
        expected = (major, minor, patch + 1)
    if target != expected:
        raise ValueError("Recorded impacts require version " + ".".join(map(str, expected)))
    return "alcedema-v" + record["version"]


def git(*args, cwd=ROOT):
    return subprocess.check_output(["git", *args], cwd=cwd, stderr=subprocess.PIPE).decode("utf-8").strip()


def validate_history(record, tag="", cwd=ROOT):
    expected = validate_record(record)
    for ref in (record["previousTag"], record["upstreamTag"]):
        git("merge-base", "--is-ancestor", "refs/tags/" + ref, "HEAD", cwd=cwd)
    upstream = git("rev-parse", "refs/tags/" + record["upstreamTag"] + "^{commit}", cwd=cwd)
    if upstream != record["upstreamCommit"]:
        raise ValueError("Upstream tag does not match upstreamCommit")
    if not tag:
        return expected
    if tag != expected:
        raise ValueError(f"Release tag must match the decision record: {expected}")
    if git("rev-parse", "refs/tags/" + tag + "^{commit}", cwd=cwd) != git("rev-parse", "HEAD", cwd=cwd):
        raise ValueError("Release tag must point to the checked-out commit")
    git("merge-base", "--is-ancestor", "HEAD", "refs/remotes/origin/main", cwd=cwd)
    # Full tag history is required. Upstream tags never affect fork ordering.
    fork_tags = []
    legacy_tags = []
    for candidate in git("tag", "--list", cwd=cwd).splitlines():
        if candidate == tag:
            continue
        match = FORK_TAG.fullmatch(candidate)
        if match:
            fork_tags.append((tuple(map(int, match.groups())), candidate))
        match = LEGACY_TAG.fullmatch(candidate)
        if match:
            legacy_tags.append((tuple(map(int, match.groups())), candidate))
    latest = max(fork_tags or legacy_tags, default=((), ""))[1]
    if latest != record["previousTag"]:
        raise ValueError(f"previousTag is stale; latest fork tag is {latest}")
    return expected


def release_notes(record, commit):
    changes = "\n".join(f"- {c['summary']} ({c['impact']})" for c in record["changes"])
    return (f"Alcedema CUPS Web {record['version']}\n\n{changes}\n\n"
            f"Version decision: {record['rationale']}\n\n"
            f"Upgrade notes: {record.get('upgradeNotes') or 'No special upgrade steps recorded.'}\n\n"
            f"Stability commitment: {record.get('stabilityRationale') or 'Unchanged.'}\n\n"
            f"Previous fork release: `{record['previousTag']}`\n\n"
            f"Based on [hanxi/cups-web {record['upstreamTag']}]"
            f"(https://github.com/hanxi/cups-web/releases/tag/{record['upstreamTag']}) "
            f"(`{record['upstreamCommit']}`).\n\nSource commit: `{commit}`\n\n"
            "Development: https://gitlab.com/Alcedema/cups-web")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", default=os.environ.get("CI_COMMIT_TAG", ""))
    args = parser.parse_args()
    record = json.loads((ROOT / "release.json").read_text(encoding="utf-8"))
    tag = args.tag
    # Original upstream tags remain mirrored but must never publish fork assets.
    if tag and re.fullmatch("v" + VERSION, tag):
        tag = ""
    try:
        expected = validate_history(record, tag)
    except (ValueError, subprocess.CalledProcessError) as error:
        parser.exit(1, f"Release policy failed: {error}\n")
    print(f"Validated candidate {record['version']} ({expected}); no tag created or pushed.")


if __name__ == "__main__":
    main()
