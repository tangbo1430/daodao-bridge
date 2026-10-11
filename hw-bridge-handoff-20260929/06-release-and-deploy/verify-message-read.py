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

import sys,re
key=sys.argv[1]
if not re.fullmatch(r'[0-9_]{1,100}',key):raise RuntimeError('Invalid C2C key')
q="SELECT record_json FROM bridge_message_record WHERE tencent_key='%s';"%key
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
if p.returncode:raise RuntimeError('Mapping lookup failed')
rows=[json.loads(raw) for raw in p.stdout.splitlines()]
if len(rows)!=1:raise RuntimeError('Expected single message mapping')
v=rows[0]
print('bridge_message',json.dumps({k:v.get(k) for k in ['tencent_key','tang_id','tang_seq','source','need_receipt','read_synced','tencent_receipt_synced','revoked']}))
r=tx('admin_getroammsg',{'Operator_Account':v['tencent_from'],'Peer_Account':v['tencent_to'],'MinTime':v['occurred_at']-5,'MaxTime':int(time.time()),'MaxCnt':100},'openim')
if r.get('ErrorCode')!=0:raise RuntimeError('Roaming lookup failed')
found=[m for m in r.get('MsgList',[]) if m.get('MsgKey')==key]
if len(found)!=1:raise RuntimeError('Message not found in roaming page')
print('tencent_message',json.dumps({k:found[0].get(k) for k in ['MsgKey','IsPeerRead','MsgFlagBits']}))
if len(sys.argv)>2 and sys.argv[2]=='--expect-read':
 if not v.get('tencent_receipt_synced') or found[0].get('IsPeerRead')!=1:raise RuntimeError('Explicit receipt not confirmed on both sides')
 print('TENCENT_MESSAGE_RECEIPT_OK')
