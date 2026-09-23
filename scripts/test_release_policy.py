import copy
from contextlib import chdir
import hashlib
import json
import os
from pathlib import Path
import runpy
import subprocess
import tempfile
import unittest
from unittest.mock import MagicMock, patch
from urllib.error import HTTPError

from release_policy import ROOT, git, release_notes, validate_history, validate_record


class ReleasePolicyTests(unittest.TestCase):
    def setUp(self):
        self.record = {
            "version": "0.3.0", "previousTag": "v0.2.15-alcedema.2",
            "upstreamTag": "v0.2.15", "upstreamCommit": "a" * 40,
            "changes": [{"impact": "transition", "summary": "Independent version series"}],
            "rationale": "A documented compatibility assessment",
            "upgradeNotes": "Documented upgrade steps", "stabilityRationale": "",
        }

    def decision(self, previous, version, *impacts):
        record = copy.deepcopy(self.record)
        record.update(previousTag="alcedema-v" + previous, version=version,
                      changes=[{"impact": impact, "summary": "A described change"} for impact in impacts],
                      stabilityRationale="Maintainer-agreed public interface commitment")
        return record

    def test_version_impact_matrix(self):
        cases = [
            ("0.3.0", "0.3.1", ["fix"]),
            ("0.3.9", "0.4.0", ["feature", "fix"]),
            ("0.3.9", "0.4.0", ["breaking", "maintenance"]),
            ("0.4.0", "1.0.0", ["stable"]),
            ("1.2.3", "1.2.4", ["maintenance"]),
            ("1.2.3", "1.3.0", ["feature", "fix"]),
            ("1.2.3", "2.0.0", ["breaking", "feature", "fix"]),
        ]
        for previous, version, impacts in cases:
            with self.subTest(version=version, impacts=impacts):
                self.assertEqual(validate_record(self.decision(previous, version, *impacts)), "alcedema-v" + version)

    def test_rejects_under_over_bumps_and_unexplained_changes(self):
        cases = [self.decision("1.2.3", "1.2.4", "breaking"),
                 self.decision("0.3.0", "2.0.0", "fix"),
                 self.decision("0.3.0", "0.3.0", "fix"),
                 self.decision("1.2.3", "1.3.0", "maintenance"),
                 self.decision("1.2.3", "1.2.4-alcedema.1", "fix"),
                 self.decision("1.2.3", "01.2.4", "fix"),
                 self.decision("1.2.3", "1.2.4", "unknown"),
                 self.decision("0.3.0", "0.3.1", "transition")]
        for field in ("rationale", "upstreamTag", "upstreamCommit"):
            record = self.decision("0.3.0", "0.3.1", "fix")
            record[field] = ""
            cases.append(record)
        for impact, field, version in [("breaking", "upgradeNotes", "0.4.0"),
                                        ("stable", "stabilityRationale", "1.0.0")]:
            record = self.decision("0.3.0", version, impact)
            record[field] = ""
            cases.append(record)
        for record in cases:
            with self.subTest(record=record), self.assertRaises(ValueError):
                validate_record(record)

    def test_transition_is_explicit_and_one_time(self):
        record = self.decision("0.2.15", "0.3.0", "transition")
        record["previousTag"] = "v0.2.15-alcedema.2"
        self.assertEqual(validate_record(record), "alcedema-v0.3.0")
        record["version"] = "0.3.1"
        with self.assertRaises(ValueError):
            validate_record(record)

    def test_history_and_tag_guards(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            def run(*args):
                return git(*args, cwd=repo)
            run("init", "-b", "main")
            run("config", "user.name", "Test")
            run("config", "user.email", "test@example.invalid")
            run("-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "base")
            upstream = run("rev-parse", "HEAD")
            run("tag", "v0.2.15")
            run("tag", "alcedema-v0.3.0")
            run("tag", "v99.0.0")
            run("-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "candidate")
            run("update-ref", "refs/remotes/origin/main", "HEAD")
            run("tag", "alcedema-v0.3.1")
            record = self.decision("0.3.0", "0.3.1", "fix")
            record.update(upstreamCommit=upstream, upstreamTag="v0.2.15")
            before = run("show-ref")
            self.assertEqual(validate_history(record, "alcedema-v0.3.1", repo), "alcedema-v0.3.1")
            self.assertEqual(run("show-ref"), before)
            for tag in ("alcedema-v0.3.2", "v0.3.1-alcedema.1"):
                with self.assertRaises(ValueError):
                    validate_history(record, tag, repo)
            record["upstreamCommit"] = "0" * 40
            with self.assertRaises(ValueError):
                validate_history(record, "alcedema-v0.3.1", repo)
            record["upstreamCommit"] = upstream
            run("tag", "alcedema-v0.4.0")
            with self.assertRaises(ValueError):
                validate_history(record, "alcedema-v0.3.1", repo)
            run("tag", "-d", "alcedema-v0.4.0")
            run("update-ref", "refs/remotes/origin/main", upstream)
            with self.assertRaises(subprocess.CalledProcessError):
                validate_history(record, "alcedema-v0.3.1", repo)

    def test_notes_include_actual_changes_and_provenance(self):
        record = self.decision("0.3.0", "0.3.1", "fix")
        notes = release_notes(record, "a" * 40)
        for value in ("0.3.1", "alcedema-v0.3.0", "A described change", record["rationale"],
                      record["upstreamCommit"], "a" * 40):
            self.assertIn(value, notes)

    def test_publication_guards_and_asset_allowlist(self):
        for scenario in ("valid", "existing", "checksum", "source", "record", "missing"):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as directory, chdir(directory):
                record = self.decision("0.3.0", "0.3.1", "fix")
                Path("release.json").write_text(json.dumps(record), encoding="utf-8")
                Path("bin").mkdir()
                binary = b"disposable test binary"
                Path("bin/cups-web-linux-amd64").write_bytes(binary)
                Path("bin/SHA256SUMS").write_text(hashlib.sha256(binary).hexdigest() + "  cups-web-linux-amd64\n", encoding="utf-8")
                Path("bin/SOURCE_COMMIT").write_text("a" * 40, encoding="utf-8")
                Path("bin/RELEASE.json").write_text(json.dumps(record), encoding="utf-8")
                Path("bin/LICENSE.txt").write_text("Test licence", encoding="utf-8")
                Path("bin/private-note.txt").write_text("Must never be uploaded", encoding="utf-8")
                if scenario == "checksum":
                    Path("bin/cups-web-linux-amd64").write_bytes(b"different binary")
                if scenario == "source":
                    Path("bin/SOURCE_COMMIT").write_text("b" * 40, encoding="utf-8")
                if scenario == "record":
                    Path("bin/RELEASE.json").write_text("{}", encoding="utf-8")
                if scenario == "missing":
                    Path("bin/LICENSE.txt").unlink()
                calls = []
                def request(req, **kwargs):
                    calls.append(req)
                    if req.get_method() == "GET" and scenario != "existing":
                        raise HTTPError(req.full_url, 404, "Not found", {}, None)
                    return MagicMock()
                environment = dict(CI_API_V4_URL="https://example.invalid/api/v4", CI_PROJECT_ID="1",
                                   CI_COMMIT_TAG="alcedema-v0.3.1", CI_COMMIT_SHA="a" * 40,
                                   CI_JOB_TOKEN="test-job-token")
                with patch.dict(os.environ, environment), patch("release_policy.validate_history"), \
                        patch("urllib.request.urlopen", side_effect=request), patch("builtins.print"):
                    if scenario == "valid":
                        runpy.run_path(str(ROOT / "scripts/publish_release.py"))
                        self.assertEqual([req.get_method() for req in calls], ["GET"] + ["PUT"] * 5 + ["POST"])
                        self.assertTrue(all("private-note" not in req.full_url for req in calls))
                        body = json.loads(calls[-1].data)
                        self.assertEqual(body["name"], "Alcedema CUPS Web 0.3.1")
                        self.assertIn(record["rationale"], body["description"])
                    else:
                        with self.assertRaises(ValueError):
                            runpy.run_path(str(ROOT / "scripts/publish_release.py"))
                        self.assertFalse(any(req.get_method() in {"PUT", "POST"} for req in calls))


if __name__ == "__main__":
    unittest.main()
