import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('sync', Path(__file__).with_name('upstream_sync.py'))
sync = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sync)

class SyncTests(unittest.TestCase):
    def test_merge_and_conflicts_do_not_change_main(self):
        for conflict in (False, True):
            with self.subTest(conflict=conflict), tempfile.TemporaryDirectory() as temporary:
                repo = Path(temporary) / 'repo'; repo.mkdir()
                sync.git('init', '-b', 'main', cwd=repo)
                sync.git('config', 'user.name', 'Test', cwd=repo)
                sync.git('config', 'user.email', 'test@example.invalid', cwd=repo)
                (repo / 'app').write_text('base\n')
                sync.git('add', '.', cwd=repo);sync.git('commit', '-m', 'base', cwd=repo)
                sync.git('branch', 'release', cwd=repo)
                (repo / 'app').write_text('fork\n')
                sync.git('commit', '-am', 'fork', cwd=repo)
                base = sync.git('rev-parse', 'main', cwd=repo).stdout.strip()
                sync.git('switch', 'release', cwd=repo)
                (repo / ('app' if conflict else 'new')).write_text('upstream\n')
                sync.git('add', '.', cwd=repo);sync.git('commit', '-m', 'release', cwd=repo)
                release = sync.git('rev-parse', 'HEAD', cwd=repo).stdout.strip()
                sync.git('switch', 'main', cwd=repo)
                for i in range(2):
                    target=Path(temporary)/f'candidate-{i}'
                    candidate, conflicts=sync.integrate(repo,base,release,target)
                    self.assertEqual(conflicts,['app'] if conflict else [])
                    self.assertEqual(sync.git('rev-parse','main',cwd=repo).stdout.strip(),base)
                    if not conflict:
                        self.assertEqual(sync.git('merge-base','--is-ancestor',release,candidate,cwd=repo,check=False).returncode,0)
                    sync.git('worktree','remove','--force',str(target),cwd=repo)
                self.assertEqual(sync.git('merge-base','--is-ancestor',base,base,cwd=repo,check=False).returncode,0)

    def test_report_is_updated_instead_of_duplicated(self):
        reports=[]
        def api(path,method='GET',body=None):
            if method=='GET': return reports
            if method=='POST': reports.append(dict(body,iid=1));return reports[-1]
            reports[0].update(body);return reports[0]
        for i in range(3):sync.upsert(api,'issues','unique-marker',{'description':f'unique-marker attempt {i}'})
        self.assertEqual(len(reports),1)
        self.assertIn('attempt 2',reports[0]['description'])

if __name__=='__main__':unittest.main()
