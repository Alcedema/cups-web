"""Acceptance checks for a disposable instance only; never submit a print job."""
import argparse
import http.cookiejar
import json
import urllib.error
import urllib.request

class Browser:
    def __init__(self, base):
        self.base=base;self.jar=http.cookiejar.CookieJar()
        self.client=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
    def call(self,path,body=None,method=None,csrf=True,headers=None,status=200):
        assert path not in ('/api/print','/api/compose') and not path.endswith('/run') and not path.endswith('/reprint')
        headers=dict(headers or {})
        if body is not None:headers['Content-Type']='application/json'
        if csrf:
            token=next((c.value for c in self.jar if c.name=='csrf_token'),'')
            if token:headers['X-CSRF-Token']=token
        req=urllib.request.Request(self.base+path,data=None if body is None else json.dumps(body).encode(),method=method,headers=headers)
        try:
            with self.client.open(req,timeout=45) as response:code=response.status;raw=response.read()
        except urllib.error.HTTPError as error:code=error.code;raw=error.read()
        assert code==status,(path,code,raw[:300])
        try:return json.loads(raw)
        except ValueError:return raw.decode()
    def login(self,name,password):
        self.call('/api/login',{'username':name,'password':password});self.call('/api/csrf')

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--base',required=True)
    parser.add_argument('--disposable',action='store_true',required=True)
    args=parser.parse_args()
    a=Browser(args.base)
    settings=a.call('/api/public-settings')
    assert settings['defaultLanguage']=='en' and settings['supportedLanguages']==['en','zh-CN']
    a.call('/api/me/preferences',{'language':'en'},'PUT',status=401)
    a.login('admin','admin')
    a.call('/api/me/preferences',{'language':'zh-CN'},'PUT',csrf=False,status=403)
    for value in [None,'de','EN']:
        a.call('/api/me/preferences',{'language':value},'PUT',status=400)
    a.call('/api/me/preferences',{'language':'zh-CN'},'PUT')
    b=Browser(args.base);b.login('admin','admin')
    assert b.call('/api/me')['effectiveLanguage']=='zh-CN'
    a.call('/api/logout',{})
    a.login('admin','admin')
    assert a.call('/api/me')['language']=='zh-CN'
    a.call('/api/admin/users',{'username':'language-test-user','password':'original-password','role':'user'})
    u=Browser(args.base);u.login('language-test-user','original-password')
    assert u.call('/api/me')['effectiveLanguage']=='en'
    u.call('/api/me/preferences',{'language':'en'},'PUT')
    assert a.call('/api/me')['language']=='zh-CN'
    key=u.call('/api/api-keys',{'name':'disposable-test','expiresInDays':1})
    token=key.get('key') or key.get('token')
    assert token, 'API key response did not contain a key'
    api=Browser(args.base)
    for path,body in [('/api/me/preferences',{'language':'zh-CN'}),('/api/me/password',{'currentPassword':'original-password','newPassword':'valid-password'})]:
        api.call(path,body,'PUT',headers={'Authorization':'Bearer '+token},status=403)
    for current,password in [('wrong','valid-password'),('original-password','short'),('original-password','a'*73),('original-password','字'*25)]:
        u.call('/api/me/password',{'currentPassword':current,'newPassword':password},'PUT',status=400)
    u.call('/api/me/password',{'currentPassword':'original-password','newPassword':'valid-password'},'PUT',csrf=False,status=403)
    u.call('/api/me/password',{'currentPassword':'original-password','newPassword':'字'*24},'PUT')
    u.call('/api/me',status=401)
    u.login('language-test-user','字'*24)
    me=u.call('/api/me');assert me['role']=='user' and me['language']=='en'
    a.call('/api/admin/settings',{'guestMode':True},'PUT')
    guest=Browser(args.base);guest.call('/api/session');guest.call('/api/csrf')
    assert guest.call('/api/me')['effectiveLanguage']=='en'
    for path,body in [('/api/me/preferences',{'language':'zh-CN'}),('/api/me/password',{'currentPassword':'guest','newPassword':'new-password'})]:
        guest.call(path,body,'PUT',status=403)
    a.call('/api/admin/settings',{'guestMode':False},'PUT')
    print('PASS: defaults, two browsers, independent users, logout persistence, CSRF, API-key/guest rejection, Unicode password limits, session clearing and account preservation. No print job submitted.')

if __name__=='__main__':main()
