"""Run on Bridge CVM. Read existing secrets; print only curated test outcomes."""
import json,hmac,hashlib,time,secrets,urllib.request,urllib.error,pathlib,sys
cfg={}
for line in pathlib.Path('/opt/hw-bridge/.env').read_text().splitlines():
 k,sep,v=line.partition('=')
 if sep:cfg[k]=v.strip().strip('"').strip("'")
secret=cfg['BRIDGE_SHARED_SECRET'].encode()
def request(url,payload,signed=True,headers=None):
 body=json.dumps(payload,separators=(',',':')).encode()
 if headers is None:
  headers={'Content-Type':'application/json'}
  if signed:
   ts=str(int(time.time()));nonce=secrets.token_hex(16)
   headers.update({'X-HWB-Timestamp':ts,'X-HWB-Nonce':nonce,'X-HWB-Sign':hmac.new(secret,(ts+'\n'+nonce+'\n').encode()+body,hashlib.sha256).hexdigest()})
 req=urllib.request.Request(url,body,headers)
 try:
  with urllib.request.urlopen(req,timeout=20) as r:return r.status,json.load(r),headers
 except urllib.error.HTTPError as e:
  try:result=json.load(e)
  except Exception:result={}
  return e.code,result,headers
php='https://admin.awhaha.com/internal/bridge/'
payload={'username':'fixture-read-only-lookup','reserve_id':'0'*64}
code,result,headers=request(php+'registration_lookup',payload)
print('PHP_LOOKUP',code,'empty_result',result.get('tim_uid')=='')
assert code==200 and result.get('code')==0 and result.get('tim_uid')==''
code,_,_=request(php+'registration_lookup',payload,headers=headers);print('PHP_NONCE_REPLAY',code);assert code==409
code,_,_=request(php+'account_status',{},False);print('PHP_STATUS_UNSIGNED',code);assert code==401
print('NATIVE_INVITE_CONFIGURED',bool(cfg.get('TANGSENG_INVITE_CODE')))
print('REGISTRATION_ENCRYPTION_CONFIGURED',bool(cfg.get('BRIDGE_ENCRYPTION_KEY')))
if '--status' in sys.argv:
 import subprocess
 p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],cwd='/opt/hw-bridge',input="SELECT cursor_json FROM bridge_business_cursor WHERE scope_key LIKE 'account:%';",text=True,capture_output=True)
 assert p.returncode==0
 for line in p.stdout.splitlines():
  row=json.loads(line);key=row.get('key','');state=row.get('aux','');version=row.get('value',0) or row.get('started_at',0) or 1
  if state not in ('confirmed','banned','tenant_disabled','deleted'):continue
  print('STATUS_INPUT',key,state,version)
  code,res,_=request(php+'account_status',{'tim_uid':key.removeprefix('account:'),'state':state,'reason':state,'version':version})
  print('PHP_STATUS_RESULT',code,res.get('code'))
if '--bridge' in sys.argv:
 base='http://127.0.0.1:8090'
 with urllib.request.urlopen(base+'/readyz',timeout=10) as r: print('BRIDGE_READY',r.status,json.load(r).get('status'))
 code,res,headers=request(base+'/api/v1/account/prepare',{'request_id':'fixture-invalid-input','username':'bad','client_ip':'192.0.2.1'})
 print('BRIDGE_PREPARE_VALIDATION',code,res.get('code'));assert code==409 and res.get('code')==1002
 code,_,_=request(base+'/api/v1/account/prepare',{'request_id':'fixture-invalid-input','username':'bad','client_ip':'192.0.2.1'},headers=headers)
 print('BRIDGE_NONCE_REPLAY',code);assert code==409
 print('CLOSURE_INTERNAL_VALIDATION_OK')
