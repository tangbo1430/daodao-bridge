"""One real non-friend message to our own disposable native reservation."""
import json,sys,pathlib,time,secrets,urllib.request,urllib.parse,urllib.error,hmac,hashlib,base64,zlib,subprocess,os
owner=sys.argv[1]
if not owner.isdigit():raise RuntimeError('Authorized sender UID required')
cfg={}
for line in pathlib.Path('/opt/hw-bridge/.env').read_text().splitlines():
 k,sep,v=line.partition('=')
 if sep:cfg[k]=v.strip().strip('"').strip("'")
native=json.loads(pathlib.Path('/opt/hw-bridge/registration-acceptance-state.json').read_text())
if not native.get('aborted') or not native['username'].startswith('hwqa'):raise RuntimeError('Disposable native target required')
peer=native['native_uid']
statefile=pathlib.Path('/opt/hw-bridge/failure-acceptance-state.json')
def bridge_search():
 url='http://127.0.0.1:8090/api/bridge/search_user?'+urllib.parse.urlencode({'tencent_user_id':owner,'keyword':native['username']})
 headers={'X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),b'',hashlib.sha256).hexdigest()}
 with urllib.request.urlopen(urllib.request.Request(url,headers=headers),timeout=30) as r:body=json.load(r)
 if not any(u.get('user_id')==peer for u in body.get('users',[])):raise RuntimeError('Fixture lookup failed')
def tx(command,payload):
 now=int(time.time());admin=cfg['TENCENT_ADMIN_ACCOUNT'];app=int(cfg['TENCENT_SDK_APP_ID']);ttl=600
 plain=f'TLS.identifier:{admin}\nTLS.sdkappid:{app}\nTLS.time:{now}\nTLS.expire:{ttl}\n'
 sig=base64.b64encode(hmac.new(cfg['TENCENT_SECRET_KEY'].encode(),plain.encode(),hashlib.sha256).digest()).decode()
 token={'TLS.ver':'2.0','TLS.identifier':admin,'TLS.sdkappid':app,'TLS.expire':ttl,'TLS.time':now,'TLS.sig':sig}
 us=base64.b64encode(zlib.compress(json.dumps(token,separators=(',',':')).encode())).decode().replace('+','*').replace('/','-').replace('=','_')
 qs=urllib.parse.urlencode({'sdkappid':app,'identifier':admin,'usersig':us,'random':now,'contenttype':'json'})
 url=cfg.get('TENCENT_REST_BASE_URL','https://adminapisgp.im.qcloud.com')+'/v4/openim/'+command+'?'+qs
 try:
  with urllib.request.urlopen(urllib.request.Request(url,json.dumps(payload).encode(),{'Content-Type':'application/json'}),timeout=30) as r:out=json.load(r)
 except Exception:raise RuntimeError('Tencent test call failed') from None
 if out.get('ErrorCode')!=0:raise RuntimeError('Tencent test rejected code='+str(out.get('ErrorCode')))
 return out
if not statefile.exists():
 bridge_search()
 state={'owner':owner,'peer':peer,'random':secrets.randbelow(2**30),'started':int(time.time())}
 fd=os.open(statefile,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
 with os.fdopen(fd,'w') as f:json.dump(state,f)
 result=tx('sendmsg',{'SyncOtherMachine':2,'From_Account':owner,'To_Account':peer,'MsgRandom':state['random'],'MsgTimeStamp':state['started'],'MsgBody':[{'MsgType':'TIMTextElem','MsgContent':{'Text':'桥接非好友失败提示验收'}}]})
 state['key']=result.get('MsgKey');statefile.write_text(json.dumps(state));os.chmod(statefile,0o600)
 print('NEGATIVE_MESSAGE_ACCEPTED_BY_TENCENT')
else:state=json.loads(statefile.read_text())
if not state.get('key'):raise RuntimeError('Test message outcome uncertain; inspect before retrying')
print('FAILURE_ACCEPTANCE_MESSAGE_KEY',state['key'])
q="SELECT id,status,message_json FROM bridge_outbox_job WHERE created_at>="+str(state['started'])+";"
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],cwd='/opt/hw-bridge',input=q,text=True,capture_output=True)
if p.returncode:raise RuntimeError('Job lookup failed')
original=[];notices=[];jobs=[]
for line in p.stdout.splitlines():
 key,status,raw=line.split('\t',2);m=json.loads(raw);job={'id':key,'status':status,'message':m};jobs.append(job)
 if m.get('source_message_id')==state['key']:original.append(job)
for job in jobs:
 if job['message'].get('failure_notice',{}).get('original_job') in {j['id'] for j in original}:notices.append(job)
print('ORIGINAL_JOB_STATES',[j.get('status') for j in original]);print('FAILURE_NOTICE_STATES',[j.get('status') for j in notices])
if len(original)==1 and original[0].get('status')=='dead' and len(notices)==1 and notices[0].get('status')=='done':print('NONFRIEND_FAILURE_ONCE_VERIFIED')
