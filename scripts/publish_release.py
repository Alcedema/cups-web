"""Publish CI-built files with the short-lived CI job token; no sync credential."""
import json
import os
from pathlib import Path
import urllib.error
import urllib.request

base=os.environ['CI_API_V4_URL']+'/projects/'+os.environ['CI_PROJECT_ID']
tag=os.environ['CI_COMMIT_TAG']
headers={'JOB-TOKEN':os.environ['CI_JOB_TOKEN']}
links=[]
for file in sorted(Path('bin').iterdir()):
    url=base+'/packages/generic/cups-web/'+tag+'/'+file.name
    req=urllib.request.Request(url,data=file.read_bytes(),method='PUT',headers=headers)
    with urllib.request.urlopen(req,timeout=120) as response: response.read()
    links.append({'name':file.name,'url':url,'link_type':'package'})
body={'name':tag,'tag_name':tag,'description':'Bilingual CUPS Web with account preferences and password changes.\n\nSource commit: `'+os.environ['CI_COMMIT_SHA']+'`\n\nDevelopment: https://gitlab.com/Alcedema/cups-web\nUpstream: https://github.com/hanxi/cups-web','assets':{'links':links}}
req=urllib.request.Request(base+'/releases',data=json.dumps(body).encode(),method='POST',headers=dict(headers,**{'Content-Type':'application/json'}))
with urllib.request.urlopen(req,timeout=60) as response: response.read()
print('Published '+tag)
