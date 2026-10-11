"""Dedicated registration account only: create group, later verify owner cleanup."""
import json,sys,hmac,hashlib,time,secrets,urllib.request,urllib.parse,urllib.error,base64,zlib,pathlib,subprocess,os
owner,survivor,phase=sys.argv[1:4]
if not owner.isdigit() or not survivor.isdigit() or owner==survivor:raise RuntimeError('Invalid dedicated account parameters')
cfg={}
for line in pathlib.Path('/opt/hw-bridge/.env').read_text().splitlines():
 k,sep,v=line.partition('=')
 if sep:cfg[k]=v.strip().strip('"').strip("'")
def sql(q):
 p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],cwd='/opt/hw-bridge',input=q,text=True,capture_output=True)
 if p.returncode:raise RuntimeError('State lookup failed')
 return p.stdout.strip()
index=json.loads(sql("SELECT cursor_json FROM bridge_business_cursor WHERE scope_key='account-registration:"+owner+"';"))
reservation=json.loads(json.loads(sql("SELECT cursor_json FROM bridge_business_cursor WHERE scope_key='registration:"+index['aux']+"';"))['aux'])
if not reservation['username'].startswith('hwqa') or reservation['tim_uid']!=owner:raise RuntimeError('Not a dedicated registration fixture')
def tx(command,payload,service='group_open_http_svc'):
 now=int(time.time());app=int(cfg['TENCENT_SDK_APP_ID']);admin=cfg['TENCENT_ADMIN_ACCOUNT'];ttl=600
 sign=base64.b64encode(hmac.new(cfg['TENCENT_SECRET_KEY'].encode(),f'TLS.identifier:{admin}\nTLS.sdkappid:{app}\nTLS.time:{now}\nTLS.expire:{ttl}\n'.encode(),hashlib.sha256).digest()).decode()
 obj={'TLS.ver':'2.0','TLS.identifier':admin,'TLS.sdkappid':app,'TLS.expire':ttl,'TLS.time':now,'TLS.sig':sign}
 sig=base64.b64encode(zlib.compress(json.dumps(obj,separators=(',',':')).encode())).decode().replace('+','*').replace('/','-').replace('=','_')
 query=urllib.parse.urlencode({'sdkappid':app,'identifier':admin,'usersig':sig,'random':now,'contenttype':'json'})
 url=cfg.get('TENCENT_REST_BASE_URL','https://adminapisgp.im.qcloud.com')+'/v4/'+service+'/'+command+'?'+query
 try:
  with urllib.request.urlopen(urllib.request.Request(url,json.dumps(payload).encode(),{'Content-Type':'application/json'}),timeout=30) as r:out=json.load(r)
 except Exception:raise RuntimeError('Tencent request unavailable') from None
 if out.get('ErrorCode')!=0:raise RuntimeError('Tencent operation rejected code='+str(out.get('ErrorCode')))
 return out
def bridge(action,payload):
 body=json.dumps(payload,separators=(',',':')).encode();headers={'Content-Type':'application/json','X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()}
 try:
  with urllib.request.urlopen(urllib.request.Request('http://127.0.0.1:8090/api/bridge/'+action,body,headers),timeout=90) as r:out=json.load(r)
 except urllib.error.HTTPError as e:
  out=json.load(e);raise RuntimeError('Bridge operation rejected code='+str(out.get('code'))+' step='+str(out.get('step'))) from None
 if out.get('code')!=200:raise RuntimeError('Bridge business operation rejected')
 return out.get('data',out)
statefile=pathlib.Path('/opt/hw-bridge/disposable-owner-'+owner+'.json')
if phase=='create':
 if statefile.exists():state=json.loads(statefile.read_text())
 else:
  state={'owner':owner,'survivor':survivor,'request_id':secrets.token_hex(16)}
  fd=os.open(statefile,os.O_CREAT|os.O_EXCL|os.O_WRONLY,0o600)
  with os.fdopen(fd,'w') as f:json.dump(state,f)
 tx('account_import',{'Identifier':owner,'Nick':'桥接注销验收账号'},'im_open_login_svc')
 peer=sys.argv[4]
 if not peer.isalnum() or len(peer)!=32:raise RuntimeError('Authorized seed friend required')
 group=bridge('group_create',{'tencent_user_id':survivor,'members':[peer,owner],'name':'桥接注销验收临时群','request_id':state['request_id']})
 state['group_id']=group.get('tencent_group_id') or group.get('tencent_id') or group.get('group_id')
 if not state['group_id']:
  raise RuntimeError('Group mapping ID unavailable')
 statefile.write_text(json.dumps(state));os.chmod(statefile,0o600)
 bridge('group_remove',{'tencent_user_id':survivor,'group_id':state['group_id'],'members':[peer]})
 bridge('group_update',{'tencent_user_id':survivor,'group_id':state['group_id'],'field':'owner','value':owner})
 profile=tx('get_group_info',{'GroupIdList':[state['group_id']]})['GroupInfo'][0]
 assert profile['Owner_Account']==owner
 print('DISPOSABLE_OWNER_GROUP_CREATED',state['group_id'])
elif phase=='verify':
 state=json.loads(statefile.read_text());gid=state['group_id']
 account=json.loads(sql("SELECT cursor_json FROM bridge_business_cursor WHERE scope_key='account:"+owner+"';"))
 print('DISPOSABLE_ACCOUNT_STATE',account.get('aux'))
 if account.get('aux')!='deleted':raise RuntimeError('Cleanup not complete')
 assert reservation['status']=='deleted' and reservation['password_cipher']==''
 profile=tx('get_group_info',{'GroupIdList':[gid]})['GroupInfo'][0]
 members=tx('get_group_member_info',{'GroupId':gid,'Limit':100,'Offset':0}).get('MemberList',[])
 assert profile['Owner_Account']==survivor and all(m['Member_Account']!=owner for m in members)
 print('DISPOSABLE_OWNER_TRANSFER_AND_CREDENTIAL_ERASE_OK')
 bridge('group_disband',{'tencent_user_id':survivor,'group_id':gid})
 state['cleaned']=True;statefile.write_text(json.dumps(state));print('DISPOSABLE_TEMP_GROUP_DISBANDED')
else:raise RuntimeError('Unknown fixture phase')
