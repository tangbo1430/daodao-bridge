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
code,v=call('GET','/v1/users/'+peer,token=token)
print('shadow_to_peer_profile',code, json.dumps({k:v.get(k) for k in ['uid','receipt']}) if v else None)
for label,start,end in [('incident_range',20,22),('new_message_range',35,37)]:
 code,v=call('POST','/v1/message/channel/sync',{'channel_id':peer,'channel_type':1,'start_message_seq':start,'end_message_seq':end,'limit':10,'pull_mode':1,'device_uuid':'hwbridge-contract-audit'},token)
 print(label,'http',code)
 if v:
  for m in v.get('messages',[]):print('message',json.dumps({k:m.get(k) for k in ['message_idstr','message_seq','setting','revoke']}))