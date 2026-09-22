#!/usr/bin/env python3
"""Run only from protected main. Never execute code from an upstream candidate."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import urllib.parse
import urllib.request

def git(*args, cwd=None, check=True):
    result = subprocess.run(['git', '-c', 'core.hooksPath=/dev/null', '-c', 'credential.helper=', *args], cwd=cwd,
                            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if check and result.returncode:
        detail = result.stderr
        for name in ('UPSTREAM_SYNC_TOKEN', 'CI_JOB_TOKEN'):
            secret = os.environ.get(name)
            if secret: detail = detail.replace(secret, '[REDACTED]')
        raise RuntimeError('Git operation failed: ' + detail)
    return result

def integrate(repo, base, release, target):
    """Return conflicts without moving any existing branch, especially main."""
    git('worktree', 'add', '--detach', str(target), base, cwd=repo)
    result = git('-c', 'user.name=CUPS upstream sync', '-c', 'user.email=sync@cups-web.invalid',
                 'merge', '--no-ff', '--no-edit', release, cwd=target, check=False)
    if result.returncode:
        conflicts = git('diff', '--name-only', '--diff-filter=U', cwd=target).stdout.splitlines()
        git('merge', '--abort', cwd=target, check=False)
        if not conflicts:
            raise RuntimeError('Merge failed without conflicts: ' + result.stderr)
        return None, conflicts
    return git('rev-parse', 'HEAD', cwd=target).stdout.strip(), []

def request(url, method='GET', body=None, token=None):
    headers = {'Accept': 'application/json', 'User-Agent': 'cups-web-upstream-sync'}
    if token: headers['PRIVATE-TOKEN'] = token
    if body is not None: headers['Content-Type'] = 'application/json'
    req = urllib.request.Request(url, headers=headers, method=method,
                                 data=None if body is None else json.dumps(body).encode())
    with urllib.request.urlopen(req, timeout=60) as response:
        raw = response.read()
        return json.loads(raw) if raw else None

def upsert(api, resource, marker, body, **query):
    query.update({'search': marker, 'in': 'description', 'state': 'all', 'per_page': 100})
    matches = api(resource + '?' + urllib.parse.urlencode(query))
    existing = next((item for item in matches if marker in (item.get('description') or '')), None)
    if existing:
        return api(f"{resource}/{existing['iid']}", 'PUT', body)
    return api(resource, 'POST', body)

def main():
    if os.environ.get('CI_COMMIT_BRANCH') != 'main' or os.environ.get('CI_COMMIT_REF_PROTECTED') != 'true' or os.environ.get('CI_PIPELINE_SOURCE') != 'schedule':
        raise SystemExit('Upstream synchronization is restricted to scheduled protected main pipelines')
    token = os.environ['UPSTREAM_SYNC_TOKEN']
    api_base = os.environ['CI_API_V4_URL'] + '/projects/' + os.environ['CI_PROJECT_ID'] + '/'
    api = lambda path, method='GET', body=None: request(api_base + path, method, body, token)
    release = request('https://api.github.com/repos/hanxi/cups-web/releases/latest')
    tag = release['tag_name']
    if release.get('draft') or release.get('prerelease') or not re.fullmatch(r'v\d+\.\d+\.\d+', tag):
        raise SystemExit('Latest release is not a stable version')
    repo = Path.cwd()
    git('fetch', '--no-tags', 'https://github.com/hanxi/cups-web.git', 'refs/tags/' + tag)
    sha = git('rev-parse', 'FETCH_HEAD^{commit}').stdout.strip()
    git('fetch', 'origin', 'main')
    base = git('rev-parse', 'origin/main').stdout.strip()
    branch = 'integrate/' + tag
    marker = 'cups-web-upstream:' + tag
    link = f"[Upstream release {tag}]({release['html_url']})\n\nRelease commit: `{sha}`\nBase commit: `{base}`"
    # The secret is exposed only to Git's askpass helper, never a URL or command argument.
    with tempfile.TemporaryDirectory(prefix='cups-sync-') as temporary:
        askpass = Path(temporary) / 'askpass.sh'
        askpass.write_text('#!/bin/sh\ncase "$1" in *Username*) echo oauth2;; *) printf "%s\\n" "$UPSTREAM_SYNC_TOKEN";; esac\n')
        askpass.chmod(0o700)
        os.environ['GIT_ASKPASS'] = str(askpass)
        os.environ['GIT_TERMINAL_PROMPT'] = '0'
        # Remove the runner's included URL rewrites, which otherwise inject its
        # read-only CI_JOB_TOKEN even after changing the origin URL. This
        # checkout is disposable and fetching has already finished.
        git('config', '--local', '--unset-all', 'include.path', check=False)
        project_url = urllib.parse.urlsplit(os.environ['CI_PROJECT_URL'])
        push_url = urllib.parse.urlunsplit((project_url.scheme, 'oauth2@' + project_url.netloc, project_url.path + '.git', '', ''))
        git('remote', 'set-url', 'origin', push_url)
        git('push', 'origin', sha + ':refs/heads/upstream/stable')
        # Preserve the exact upstream release tag, refusing to overwrite an existing tag.
        git('fetch', '--no-tags', 'https://github.com/hanxi/cups-web.git', 'refs/tags/' + tag + ':refs/tags/' + tag)
        git('push', 'origin', 'refs/tags/' + tag)
        if git('merge-base', '--is-ancestor', sha, base, check=False).returncode == 0:
            print(f'{tag} is already included in main; no update')
            return
        existing = api('merge_requests?' + urllib.parse.urlencode({'source_branch': branch, 'state': 'all', 'per_page': 100}))
        if existing:
            mr = existing[0]
            detail = api(f"merge_requests/{mr['iid']}")
            pipeline = detail.get('head_pipeline') or {}
            status = pipeline.get('status', 'not yet available')
            url = pipeline.get('web_url', '')
            api(f"merge_requests/{mr['iid']}", 'PUT', {'description': link + f'\n\nValidation: {status} {url}\n\nManual review and merge required. No automatic deployment.\n\n<!-- {marker} -->'})
            print('Updated existing merge request; main unchanged')
            return
        target = Path(temporary) / 'candidate'
        try:
            candidate, conflicts = integrate(repo, base, sha, target)
            if conflicts:
                description = link + '\n\nMerge conflicts:\n' + '\n'.join('- `' + name + '`' for name in conflicts)
                description += f'\n\nResolve and review manually. Main has not been changed.\n\n<!-- {marker} -->'
                upsert(api, 'issues', marker, {'title': f'Upstream {tag}: merge conflicts', 'description': description})
                print('Conflict report updated; main unchanged')
                return
            git('push', 'origin', candidate + ':refs/heads/' + branch)
            api('merge_requests', 'POST', {'source_branch': branch, 'target_branch': 'main',
                'title': f'Integrate upstream {tag}', 'remove_source_branch': True,
                'description': link + f'\n\nValidation: merge succeeded. Credential-free frontend and Go pipeline pending; inspect the pipeline before merging. The next daily check refreshes this result.\n\nManual review and merge required. No automatic deployment.\n\n<!-- {marker} -->'})
            print('Created integration merge request; main unchanged')
        finally:
            if target.exists(): git('worktree', 'remove', '--force', str(target))

if __name__ == '__main__':
    main()
