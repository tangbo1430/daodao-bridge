import json,urllib.request,urllib.error
cfg={}
for line in open('/opt/hw-bridge/.env'):
 k,sep,v=line.strip().partition('=')
 if sep:cfg[k]=v.strip().strip('"').strip("'")
base=cfg['TANGSENG_BASE_URL'].rstrip('/')
def call(method,path,body=None,token=''):
 req=urllib.request.Request(base+path,method=method,data=None if body is None else json.dumps(body).encode(),headers={'Content-Type':'application/json','token':token})
 try:
  with urllib.request.urlopen(req,timeout=25) as r:
   raw=r.read();return r.status,json.loads(raw) if raw else None
 except urllib.error.HTTPError as e:return e.code,None
 except Exception as e:raise RuntimeError(type(e).__name__) from None
login=None
for key in ['TANGSENG_SHADOW_PASSWORD','TANGSENG_LEGACY_SHADOW_PASSWORD']:
 if not cfg.get(key):continue
 code,obj=call('POST',cfg.get('TANGSENG_LOGIN_ENDPOINT','/v1/user/usernamelogin'),{'username':'tx_74629','password':cfg[key],'flag':2,'device':{'device_id':'hwbridge-tx_74629','device_name':'Huawei Bridge','device_model':'bridge'}})
 if code==200 and obj:
  candidate=obj.get('data',obj)
  if candidate.get('token') and candidate.get('uid')=='f278d20e4ef44eefb00ad403720c9d63':login=candidate;break
if not login:raise RuntimeError('Existing shadow authentication failed')
token=login['token'];peer='0cb644b2e1304dc59ffead8af5170ba9'
import os,sys
path='/tmp/hwbridge-receipt-ui-test.json'
action=sys.argv[1]
code,v=call('GET','/v1/users/'+peer,token=token)
if code!=200 or not isinstance(v,dict) or v.get('receipt') not in (0,1):raise RuntimeError('Receipt preflight failed')
if action=='enable':
 if os.path.exists(path):raise RuntimeError('Existing test setting backup must be restored first')
 fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
 with os.fdopen(fd,'w') as f:json.dump({'owner':login['uid'],'peer':peer,'receipt':v['receipt']},f)
 wanted=1
elif action=='restore':
 saved=json.load(open(path))
 if saved['owner']!=login['uid'] or saved['peer']!=peer:raise RuntimeError('Backup identity mismatch')
 wanted=saved['receipt']
else:raise RuntimeError('Invalid action')
code,_=call('PUT','/v1/users/'+peer+'/setting',{'receipt':wanted},token)
print('setting_write_http',code)
if code!=200:raise RuntimeError('Setting update rejected')
code,v=call('GET','/v1/users/'+peer,token=token)
print('receipt_verified',code,v.get('receipt') if v else None)
if code!=200 or v.get('receipt')!=wanted:raise RuntimeError('Setting verification failed')
if action=='restore':os.rename(path,path+'.restored')