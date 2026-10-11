"""Create one disposable native reservation with the server invite, then abort.
No existing accounts are changed. The normal orphan policy retains/labels it.
No passwords, tokens, invitation values or raw native bodies are printed.
"""
import json,hmac,hashlib,time,secrets,urllib.request,urllib.error,pathlib,os
cfg={}
for line in pathlib.Path('/opt/hw-bridge/.env').read_text().splitlines():
 k,sep,v=line.partition('=')
 if sep:cfg[k]=v.strip().strip('"').strip("'")
if not cfg.get('TANGSENG_INVITE_CODE'):raise RuntimeError('Native invitation unavailable')
def call(action,p):
 body=json.dumps(p,separators=(',',':')).encode();ts=str(int(time.time()));nonce=secrets.token_hex(16)
 headers={'Content-Type':'application/json','X-HWB-Timestamp':ts,'X-HWB-Nonce':nonce,'X-HWB-Sign':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),(ts+'\n'+nonce+'\n').encode()+body,hashlib.sha256).hexdigest()}
 req=urllib.request.Request('http://127.0.0.1:8090/api/v1/account/'+action,body,headers)
 try:
  with urllib.request.urlopen(req,timeout=30) as r:return r.status,json.load(r)
 except urllib.error.HTTPError as e:return e.code,json.load(e)
state_path=pathlib.Path('/opt/hw-bridge/registration-acceptance-state.json')
if state_path.exists():
 state=json.loads(state_path.read_text())
else:
 state={'request_id':secrets.token_hex(16),'username':'hwqa'+time.strftime('%m%d%H%M')+secrets.token_hex(3),'nick':'桥接注册事务验收'}
 fd=os.open(state_path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
 with os.fdopen(fd,'w') as f:json.dump(state,f)
if state.get('aborted'):
 print('NATIVE_REGISTRATION_ALREADY_ABORTED');raise SystemExit(0)
p={**state,'invite_code':cfg['TANGSENG_INVITE_CODE'],'client_ip':'127.0.0.1'}
code,r=call('prepare',p)
print('NATIVE_PREPARE',code,'business_code',r.get('code'))
if code!=200 or r.get('code')!=0:
 print('NATIVE_PREPARE_REQUIRES_FOLLOWUP');raise SystemExit(2)
data=r['data'];state['reserve_id']=data['reserve_id'];state['native_uid']=data['tsdd_uid']
state_path.write_text(json.dumps(state));os.chmod(state_path,0o600)
try:
 code,again=call('prepare',p)
 assert code==200 and again['data']['reserve_id']==data['reserve_id'] and again['data']['tsdd_uid']==data['tsdd_uid']
 print('NATIVE_PREPARE_IDEMPOTENCY_OK')
finally:
 code,result=call('abort',{'reserve_id':data['reserve_id']})
 print('NATIVE_ABORT',code,result.get('code'))
 if code==200 and result.get('code')==0:
  state['aborted']=True;state_path.write_text(json.dumps(state));os.chmod(state_path,0o600)
 else:raise RuntimeError('Disposable reservation abort remains pending')
code,result=call('abort',{'reserve_id':data['reserve_id']})
assert code==200 and result.get('code')==0
print('NATIVE_REGISTRATION_PREPARE_ABORT_OK')
