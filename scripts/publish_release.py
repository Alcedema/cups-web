"""Publish CI-built files with the short-lived CI job token; no sync credential."""
import json
import hashlib
import os
from pathlib import Path
import urllib.error
import urllib.request
from release_policy import release_notes, validate_history

base=os.environ['CI_API_V4_URL']+'/projects/'+os.environ['CI_PROJECT_ID']
tag=os.environ['CI_COMMIT_TAG']
record=json.loads(Path('release.json').read_text(encoding='utf-8'))
validate_history(record, tag)
if Path('bin/SOURCE_COMMIT').read_text(encoding='utf-8').strip() != os.environ['CI_COMMIT_SHA']:
    raise ValueError('Build source identity does not match this release')
if json.loads(Path('bin/RELEASE.json').read_text(encoding='utf-8')) != record:
    raise ValueError('Build release decision does not match this release')
headers={'JOB-TOKEN':os.environ['CI_JOB_TOKEN']}
asset_names=('cups-web-linux-amd64','SHA256SUMS','SOURCE_COMMIT','LICENSE.txt','RELEASE.json')
assets=[Path('bin') / name for name in asset_names]
if any(not asset.is_file() for asset in assets):
    raise ValueError('A required release asset is missing')
expected_checksum=hashlib.sha256(assets[0].read_bytes()).hexdigest()+'  cups-web-linux-amd64'
if Path('bin/SHA256SUMS').read_text(encoding='utf-8').strip() != expected_checksum:
    raise ValueError('Binary checksum does not match SHA256SUMS')
# Refuse to overwrite an already published release. Unexpected API failures also
# stop publication; only a confirmed 404 permits creation.
try:
    req=urllib.request.Request(base+'/releases/'+tag,headers=headers)
    with urllib.request.urlopen(req,timeout=60) as response: response.read()
except urllib.error.HTTPError as error:
    if error.code != 404: raise
    error.close()
else:
    raise ValueError('Release already exists; published releases are immutable')
links=[]
for file in assets:
    url=base+'/packages/generic/cups-web/'+tag+'/'+file.name
    req=urllib.request.Request(url,data=file.read_bytes(),method='PUT',headers=headers)
    with urllib.request.urlopen(req,timeout=120) as response: response.read()
    links.append({'name':file.name,'url':url,'link_type':'package'})
body={'name':'Alcedema CUPS Web '+record['version'],'tag_name':tag,'description':release_notes(record,os.environ['CI_COMMIT_SHA']),'assets':{'links':links}}
req=urllib.request.Request(base+'/releases',data=json.dumps(body).encode(),method='POST',headers=dict(headers,**{'Content-Type':'application/json'}))
with urllib.request.urlopen(req,timeout=60) as response: response.read()
print('Published '+tag)
