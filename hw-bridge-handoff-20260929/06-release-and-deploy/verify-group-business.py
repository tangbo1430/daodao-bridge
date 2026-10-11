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
gid=None
try:
 g=bridge('group_create',members=[peer],name='业务验收-'+run[:8],request_id=run);gid=g['group_id'];audit['group_id']=gid;save();print('test_group',gid,flush=True)
 update('name','验收群-'+run[:8]);assert profile()['Name']=='验收群-'+run[:8]
 update('notice','业务验收：完成后自动解散');assert profile()['Notification']=='业务验收：完成后自动解散'
 update('manager',members=[peer]);assert member()['Role']=='Admin'
 update('member',members=[peer]);assert member()['Role']=='Member'
 update('mute_member','600',members=[peer]);m=member();assert max(m.get('ShutUpUntil',0),m.get('MuteUntil',0))>time.time()+500
 update('mute_member','0',members=[peer]);m=member();assert max(m.get('ShutUpUntil',0),m.get('MuteUntil',0))<=time.time()
 update('forbidden','1');assert profile()['MuteAllMember']=='On'
 update('forbidden','0');assert profile()['MuteAllMember']=='Off'
 update('invite','1');assert state()['invite']==1
 update('invite','0');assert state()['invite']==0
 bridge('group_remove',group_id=gid,members=[peer]);assert member() is None
 bridge('group_invite',group_id=gid,members=[peer]);assert member() is not None
 source=cfg['TANGSENG_BASE_URL'].rstrip('/')+'/v1/users/'+peer+'/avatar'
 update('avatar',source)
 face=profile().get('FaceUrl');assert face and urllib.parse.urlparse(face).hostname=='admin.awhaha.com'
 hashes=[]
 for u in (face,cfg['TANGSENG_BASE_URL'].rstrip('/')+'/v1/groups/'+gid[3:]+'/avatar'):
  with urllib.request.urlopen(u,timeout=30) as r:hashes.append(hashlib.sha256(r.read()).hexdigest())
 assert hashes[0]==hashes[1];print('GROUP_AVATAR_SHA256_OK',hashes[0],flush=True)
 before=profile()['NextMsgSeq']
 for _ in range(2):bridge('group_sync',group_id=gid)
 after=profile()['NextMsgSeq'];assert before==after
 audit['result']='passed';print('GROUP_ROLES_MUTE_INVITE_AVATAR_OK',flush=True)
finally:
 if gid:
  bridge('group_disband',group_id=gid)
  assert state()['status']==2 and state()['mirror_ready']==False
  assert profile().get('ErrorCode')==10010
  audit['cleanup']='passed';save();print('TEMP_GROUP_DISBANDED',gid,flush=True)