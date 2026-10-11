import os,json,time,hmac,hashlib,base64,zlib,urllib.request,urllib.parse,subprocess
cfg={}
for line in open('/opt/hw-bridge/.env'):
 k,sep,v=line.strip().partition('=')
 if sep: cfg[k]=v.strip().strip('"').strip("'")
now=int(time.time()); app=int(cfg['TENCENT_SDK_APP_ID']); admin=cfg['TENCENT_ADMIN_ACCOUNT']; ttl=3600
content=f'TLS.identifier:{admin}\nTLS.sdkappid:{app}\nTLS.time:{now}\nTLS.expire:{ttl}\n'
sig=base64.b64encode(hmac.new(cfg['TENCENT_SECRET_KEY'].encode(),content.encode(),hashlib.sha256).digest()).decode()
obj={'TLS.ver':'2.0','TLS.identifier':admin,'TLS.sdkappid':app,'TLS.expire':ttl,'TLS.time':now,'TLS.sig':sig}
us=base64.b64encode(zlib.compress(json.dumps(obj,separators=(',',':')).encode())).decode().replace('+','*').replace('/','-').replace('=','_')
query=urllib.parse.urlencode({'sdkappid':app,'identifier':admin,'usersig':us,'random':now,'contenttype':'json'})
def tx(cmd,body,service='group_open_http_svc'):
 req=urllib.request.Request(cfg.get('TENCENT_REST_BASE_URL','https://adminapisgp.im.qcloud.com')+'/v4/'+service+'/'+cmd+'?'+query,data=json.dumps(body).encode(),headers={'Content-Type':'application/json'})
 try: return json.load(urllib.request.urlopen(req,timeout=30))
 except Exception as e: raise RuntimeError(type(e).__name__)

import sys,re,uuid,urllib.error
owner,peer=sys.argv[1:3]
if not all(re.fullmatch(r'[A-Za-z0-9_-]{1,64}',v) for v in (owner,peer)):raise RuntimeError('Invalid test account')
run=uuid.uuid4().hex;audit={'run':run,'owner':owner,'peer':peer,'steps':[]};audit_path='/tmp/hwbridge-group-business-'+run+'.json'
if len(sys.argv)>3:
 if not re.fullmatch(r'[a-f0-9]{32}',sys.argv[3]):raise RuntimeError('Invalid resume request')
 run=sys.argv[3];audit_path='/tmp/hwbridge-group-business-'+run+'.json'
 audit=json.load(open(audit_path))
 if audit['owner']!=owner or audit['peer']!=peer:raise RuntimeError('Resume identity mismatch')
def save():
 fd=os.open(audit_path,os.O_WRONLY|os.O_CREAT|os.O_TRUNC,0o600)
 with os.fdopen(fd,'w') as f:json.dump(audit,f)
def bridge(action,**values):
 payload={'tencent_user_id':owner,**values};body=json.dumps(payload,ensure_ascii=False,separators=(',',':')).encode()
 req=urllib.request.Request('http://127.0.0.1:8090/api/bridge/'+action,data=body,headers={'Content-Type':'application/json','X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()})
 try:
  with urllib.request.urlopen(req,timeout=90) as r:status=r.status;out=json.load(r)
 except urllib.error.HTTPError as e:status=e.code;out=json.load(e)
 summary={'action':action,'field':values.get('field'),'http':status,'code':out.get('code'),'step':out.get('step')}
 audit['steps'].append(summary);save();print(json.dumps(summary),flush=True)
 if status!=200 or out.get('code')!=200:raise RuntimeError('Business operation rejected')
 return out.get('data',out)
def state():
 q="SELECT group_state_json FROM group_map WHERE source_system='tangseng' AND source_group_id='%s';"%gid[3:]
 p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],cwd='/opt/hw-bridge',input=q,text=True,capture_output=True)
 if p.returncode:raise RuntimeError('State lookup failed')
 return json.loads(p.stdout)
def profile():
 r=tx('get_group_info',{'GroupIdList':[gid]})
 if r.get('ErrorCode')!=0:raise RuntimeError('Tencent group query failed')
 return r['GroupInfo'][0]
def member():
 r=tx('get_group_member_info',{'GroupId':gid,'Limit':100,'Offset':0})
 if r.get('ErrorCode')!=0:raise RuntimeError('Tencent member query failed')
 return next((m for m in r.get('MemberList',[]) if m['Member_Account']==peer),None)
def update(field,value='',**extra):return bridge('group_update',group_id=gid,field=field,value=value,**extra)
# Native login uses only encrypted server-side registration material.
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

def sql(query):
 r=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],cwd='/opt/hw-bridge',input=query,text=True,capture_output=True)
 if r.returncode:raise RuntimeError('Query failed')
 return r.stdout.strip()
def cursor(key):return json.loads(sql("SELECT cursor_json FROM bridge_business_cursor WHERE scope_key='%s'"%key))
reg=json.loads(cursor('registration:'+cursor('account-registration:'+owner)['aux'])['aux'])
raw=base64.urlsafe_b64decode(reg['password_cipher']+'='*(-len(reg['password_cipher'])%4))
password=AESGCM(hashlib.sha256(('hw-bridge-registration-v1\0'+cfg['BRIDGE_ENCRYPTION_KEY']).encode()).digest()).decrypt(raw[:12],raw[12:],None).decode()
def native(method,path,body=None):
 req=urllib.request.Request(cfg['TANGSENG_BASE_URL'].rstrip('/')+path,method=method,data=None if body is None else json.dumps(body).encode(),headers={'Content-Type':'application/json','token':token})
 try:
  with urllib.request.urlopen(req,timeout=30) as r:return json.load(r)
 except Exception as e:raise RuntimeError('Native request failed: '+type(e).__name__) from None
token=''
login=native('POST','/v1/user/usernamelogin',{'username':reg['username'],'password':password,'flag':2,'device':{'device_id':'hwbridge'+reg['tsdd_uid'],'device_name':'华为互通','device_model':'hw-bridge'}})
login=login.get('data',login);assert login['uid']==reg['tsdd_uid'];token=login['token'];del password
gid=None
try:
 g=bridge('group_create',members=[peer],name='群设置验收-'+run[:6],request_id=run);gid=g['group_id'];audit['group_id']=gid;save();print('TEMP_GROUP',gid,flush=True)
 for mode in ['facade','native']:
  for value in [1,0]:
   if mode=='facade':update('invite',str(value))
   else:native('PUT','/v1/groups/'+gid[3:],{'invite':str(value)})
   g=bridge('group_sync',group_id=gid)
   assert g['invite']==value and native('GET','/v1/groups/'+gid[3:])['invite']==value
   assert profile()['ApplyJoinOption']=='DisableApply','Do not open Tencent-only joining'
   print('INVITE_SETTING_READBACK_OK',mode,value,flush=True)
 for value in [1,0]:
  update('forbidden',str(value));g=bridge('group_sync',group_id=gid)
  assert g['forbidden']==value and profile()['MuteAllMember']==('On' if value else 'Off')
  print('MUTE_SETTING_READBACK_OK',value,flush=True)
 for count,(mode,field,role) in enumerate([('facade','manager','Admin'),('facade','member','Member'),('native','manager','Admin'),('native','member','Member')],1):
  if mode=='facade':update(field,members=[peer])
  else:native('POST' if field=='manager' else 'DELETE','/v1/groups/'+gid[3:]+'/managers',[peer])
  for _ in range(15):
   n=native('POST','/v1/message/channel/sync',{'channel_id':gid[3:],'channel_type':2,'start_message_seq':0,'end_message_seq':0,'limit':100,'pull_mode':0,'device_uuid':'hwbridge-settings-audit'})
   notices=[m for m in n.get('messages',[]) if m.get('payload',{}).get('bridge_event_id','').startswith('bridge-group-role:')]
   if len(notices)==count and member()['Role']==role:break
   time.sleep(2)
  else:raise RuntimeError('Missing native role notice')
  print('ROLE_AND_NOTICE_OK',mode,field,flush=True)
 before=profile()['NextMsgSeq']
 for _ in range(3):bridge('group_sync',group_id=gid)
 assert profile()['NextMsgSeq']==before
 print('REPEATED_SYNC_NO_TENCENT_NOTICE_LOOP_OK',flush=True)
 member_path='/members?limit=1000&page=1' if state().get('group_type')==1 else '/membersync?version=0&limit=1000'
 ms=native('GET','/v1/groups/'+gid[3:]+member_path)
 me=next(m for m in ms if m['uid']==reg['tsdd_uid'])
 path=me.get('avatar') or 'users/'+me['uid']+'/avatar'
 assert path=='users/'+reg['tsdd_uid']+'/avatar','Unexpected member avatar route'
 with urllib.request.urlopen(cfg['TANGSENG_BASE_URL'].rstrip('/')+'/v1/'+path,timeout=30) as r:
  data=r.read();assert r.status==200 and r.headers.get_content_type().startswith('image/')
  print('NATIVE_GROUP_MEMBER_AVATAR_OK',len(data),hashlib.sha256(data).hexdigest(),flush=True)
 audit['result']='passed';save()
finally:
 if not gid:
  # A native group can exist even when the create response reports a failed
  # mirror update. Recover only this run's persisted request before cleanup.
  for raw_state in sql("SELECT group_state_json FROM group_map WHERE source_system='tangseng'").splitlines():
   saved=json.loads(raw_state)
   if saved.get('request_owner')==owner and saved.get('request_id')==run:
    gid=saved['tencent_id'];break
 if gid:bridge('group_disband',group_id=gid);audit['cleanup']='passed';save();print('TEMP_GROUP_CLEANED',gid,flush=True)
