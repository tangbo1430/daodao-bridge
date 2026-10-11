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

peer='0cb644b2e1304dc59ffead8af5170ba9'; owner='74629'; group='dd_7825169c0398401ca16ed59d02e0542b'
q="SELECT record_json FROM bridge_message_record WHERE tencent_from='%s' AND tencent_to IN ('%s','%s') ORDER BY occurred_at DESC LIMIT 5;"%(peer,owner,group)
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
if p.returncode: raise RuntimeError('Query failed')
for raw in p.stdout.splitlines():
 v=json.loads(raw)
 print('record',json.dumps({k:v.get(k) for k in ['tencent_key','tang_id','tang_seq','channel_type','occurred_at','read_synced','revoked']},ensure_ascii=False))
r=tx('group_msg_get_simple',{'GroupId':group,'ReqMsgNumber':5,'WithRecalledMsg':1})
print('group_history',r.get('ErrorCode'))
for v in r.get('RspMsgList',[]): print('group_state',json.dumps({k:v.get(k) for k in ['MsgSeq','IsPlaceMsg','MsgTimeStamp','From_Account']}))
r=tx('admin_getroammsg',{'Operator_Account':owner,'Peer_Account':peer,'MinTime':1790773900,'MaxTime':int(time.time()),'MaxCnt':20},'openim')
print('c2c_history',r.get('ErrorCode'),'count',len(r.get('MsgList',[])))
for v in r.get('MsgList',[]): print('c2c_state',json.dumps({k:v.get(k) for k in ['MsgKey','MsgFlagBits','IsPeerRead']}))