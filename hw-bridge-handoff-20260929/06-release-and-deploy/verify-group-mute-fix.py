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
def tx(cmd,body):
 req=urllib.request.Request(cfg.get('TENCENT_REST_BASE_URL','https://adminapisgp.im.qcloud.com')+'/v4/group_open_http_svc/'+cmd+'?'+query,data=json.dumps(body).encode(),headers={'Content-Type':'application/json'})
 try: return json.load(urllib.request.urlopen(req,timeout=30))
 except Exception as e: raise RuntimeError(type(e).__name__)

import sys
group=sys.argv[1]
owner=sys.argv[2]
def snapshot():
 r=tx('get_group_info',{'GroupIdList':[group],'ResponseFilter':{'GroupBaseInfoFilter':['MuteAllMember','NextMsgSeq']}})
 if r.get('ErrorCode')!=0 or not r.get('GroupInfo') or r['GroupInfo'][0].get('ErrorCode')!=0: raise RuntimeError('Tencent group read failed')
 g=r['GroupInfo'][0]
 return {'next_seq':g.get('NextMsgSeq'),'mute_all':g.get('MuteAllMember')}
before=snapshot()
print('before',json.dumps(before))
for i in range(3):
 body=json.dumps({'tencent_user_id':owner,'group_id':group},separators=(',',':')).encode()
 signature=hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()
 req=urllib.request.Request('http://127.0.0.1:8090/api/bridge/group_sync',data=body,headers={'Content-Type':'application/json','X-Bridge-Signature':signature})
 with urllib.request.urlopen(req,timeout=90) as res:
  data=json.load(res)
  print('sync',i+1,'http',res.status,'code',data.get('code'))
  if data.get('code')!=200: raise RuntimeError('Bridge group sync failed')
after=snapshot()
print('after',json.dumps(after))
if before!=after: raise RuntimeError('Group changed during no-op sync; inspect new messages')
print('GROUP_SYNC_NO_NEW_TIPS_OK')
