"""Probe ONLY the saved disposable reservation. Emit strict allow-listed data."""
import json,pathlib,hashlib,base64,subprocess,urllib.request,urllib.error,re,sys
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
cfg={}
for line in pathlib.Path('/opt/hw-bridge/.env').read_text().splitlines():
 k,sep,v=line.partition('=')
 if sep:cfg[k]=v.strip().strip('"').strip("'")
state=json.loads(pathlib.Path('/opt/hw-bridge/registration-acceptance-state.json').read_text())
key=hashlib.sha256(('registration\0'+state['request_id']+'\0').encode()).hexdigest()
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],cwd='/opt/hw-bridge',input="SELECT cursor_json FROM bridge_business_cursor WHERE scope_key='registration:"+key+"';",text=True,capture_output=True)
if p.returncode:raise RuntimeError('Reservation lookup unavailable')
row=json.loads(json.loads(p.stdout)['aux']);encrypted=row['password_cipher'];raw=base64.urlsafe_b64decode(encrypted+'='*((-len(encrypted))%4))
password=AESGCM(hashlib.sha256(('hw-bridge-registration-v1\0'+cfg['BRIDGE_ENCRYPTION_KEY']).encode()).digest()).decrypt(raw[:12],raw[12:],None).decode()
body={'username':row['username'],'password':password,'flag':2,'device':{'device_id':'hwbridge-'+(row['tsdd_uid'] or row['username']),'device_name':'华为互通','device_model':'hw-bridge'}}
variant='plan-device'
if '--short-device' in sys.argv:body['device']['device_id']='hwbridge'+row['tsdd_uid'];variant='40-character-device'
if '--ascii-device' in sys.argv:body['device']['device_name']='Huawei Bridge';variant+='-ascii-name'
url=cfg['TANGSENG_BASE_URL'].rstrip('/')+cfg.get('TANGSENG_LOGIN_ENDPOINT','/v1/user/usernamelogin')
req=urllib.request.Request(url,json.dumps(body).encode(),{'Content-Type':'application/json'})
try:
 with urllib.request.urlopen(req,timeout=15) as r:status=r.status;response=json.load(r)
except urllib.error.HTTPError as e:status=e.code;response=json.load(e)
def clean(v):
 if isinstance(v,dict):
  out={}
  for k,x in v.items():
   if k in ('password','token','invite_code','authorization','vercode'):out[k]='[redacted]'
   elif k in ('uid','app_id','username','name','device_id'):out[k]='fixture-'+type(x).__name__
   elif k in ('status','code','msg','message','data'):out[k]=clean(x)
  return out
 if isinstance(v,list):return [clean(x) for x in v]
 if isinstance(v,str):
  for s in (password,cfg.get('TANGSENG_INVITE_CODE',''),row['username'],row['tsdd_uid']):
   if s:v=v.replace(s,'[redacted]')
  return re.sub(r'https?://\S+','[redacted-url]',v)
 return v
print(json.dumps({'provenance':'real native login for dedicated registration acceptance reservation','variant':variant,'reservation_status':row['status'],'stored_uid_present':bool(row['tsdd_uid']),'http_status':status,'response':clean(response)},ensure_ascii=False,indent=2))
