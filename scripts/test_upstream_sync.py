import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import os
import urllib.parse

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


    def test_complete_scheduled_workflow_is_repeatable(self):
        for scenario in ('included', 'clean', 'conflict'):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                repo = root / 'working'; repo.mkdir()
                origin = root / 'origin.git'
                upstream = root / 'upstream.git'
                real_git = sync.git
                real_git('init', '-b', 'main', cwd=repo)
                real_git('config', 'user.name', 'Test', cwd=repo)
                real_git('config', 'user.email', 'test@example.invalid', cwd=repo)
                (repo / 'app').write_text('base\n')
                real_git('add', '.', cwd=repo); real_git('commit', '-m', 'base', cwd=repo)
                real_git('branch', 'release', cwd=repo)
                (repo / 'app').write_text('fork\n')
                real_git('commit', '-am', 'fork', cwd=repo)
                real_git('switch', 'release', cwd=repo)
                (repo / ('app' if scenario == 'conflict' else 'new')).write_text('upstream\n')
                real_git('add', '.', cwd=repo); real_git('commit', '-m', 'upstream', cwd=repo)
                real_git('tag', 'v1.0.0', cwd=repo)
                release = real_git('rev-parse', 'HEAD', cwd=repo).stdout.strip()
                real_git('clone', '--bare', str(repo), str(upstream))
                real_git('switch', 'main', cwd=repo)
                if scenario == 'included':real_git('merge', '--no-edit', 'release', cwd=repo)
                base = real_git('rev-parse', 'main', cwd=repo).stdout.strip()
                real_git('clone', '--bare', str(repo), str(origin))
                real_git('remote', 'add', 'origin', str(origin), cwd=repo)
                reports = {'issues': [], 'merge_requests': []}
                def api(url, method='GET', body=None, token=None):
                    if url.startswith('https://api.github.com/'):
                        return {'tag_name':'v1.0.0', 'html_url':'https://github.com/hanxi/cups-web/releases/tag/v1.0.0'}
                    path = url.split('/projects/123/')[1].split('?')[0]
                    resource = path.split('/')[0]
                    items = reports[resource]
                    if method == 'GET':
                        if '/' in path:return dict(items[0],head_pipeline={'status':'success','web_url':'https://gitlab.example/pipeline/1'})
                        return items
                    if method == 'POST':
                        items.append(dict(body,iid=1));return items[-1]
                    items[0].update(body);return items[0]
                def local_git(*args, **kwargs):
                    args = [str(upstream) if value == 'https://github.com/hanxi/cups-web.git' else value for value in args]
                    if args[:3] == ['remote','set-url','origin']:args[3] = str(origin)
                    return real_git(*args, **kwargs)
                previous = Path.cwd()
                try:
                    os.chdir(repo)
                    with patch.dict(os.environ, {'CI_COMMIT_BRANCH':'main','CI_COMMIT_REF_PROTECTED':'true','CI_PIPELINE_SOURCE':'schedule','UPSTREAM_SYNC_TOKEN':'test-only','CI_API_V4_URL':'https://gitlab.example/api/v4','CI_PROJECT_ID':'123','CI_PROJECT_URL':'https://gitlab.example/example/cups-web'}), patch.object(sync,'request',side_effect=api), patch.object(sync,'git',side_effect=local_git):
                        sync.main(); sync.main()
                finally:os.chdir(previous)
                self.assertEqual(real_git('rev-parse','main',cwd=origin).stdout.strip(),base)
                self.assertEqual(real_git('rev-parse','upstream/stable',cwd=origin).stdout.strip(),release)
                self.assertEqual(len(reports['merge_requests']), 1 if scenario == 'clean' else 0)
                self.assertEqual(len(reports['issues']), 1 if scenario == 'conflict' else 0)
                if scenario == 'clean':self.assertIn('Validation: success', reports['merge_requests'][0]['description'])
                if scenario == 'conflict':
                    self.assertIn('`app`',reports['issues'][0]['description'])
                    self.assertIn(release,reports['issues'][0]['description'])

if __name__=='__main__':unittest.main()
