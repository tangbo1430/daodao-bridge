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

import sys
peer=sys.argv[1]
if not peer or len(peer)>32 or not all(c.isalnum() or c in '_-' for c in peer): raise RuntimeError('Invalid peer ID')
r=tx('portrait_get',{'To_Account':[peer],'TagList':['Tag_Profile_IM_Nick','Tag_Profile_IM_Image']},'profile')
if r.get('ErrorCode')!=0: raise RuntimeError('Tencent profile query failed')
users=[u for u in r.get('UserProfileItem',[]) if u.get('To_Account')==peer and u.get('ResultCode')==0]
if len(users)!=1: raise RuntimeError('Profile missing')
avatar=next((x.get('Value','') for x in users[0].get('ProfileItem',[]) if x.get('Tag')=='Tag_Profile_IM_Image'),'')
if not avatar: raise RuntimeError('Tencent avatar empty')
hashes=[]
for label,u in [('tencent_avatar',avatar),('tang_avatar','http://10.0.0.100:8090/v1/users/'+urllib.parse.quote(peer,safe='')+'/avatar')]:
 try:
  with urllib.request.urlopen(u,timeout=30) as res:
   data=res.read(4*1024*1024+1)
   if len(data)>4*1024*1024: raise RuntimeError('Avatar too large for verification')
   digest=hashlib.sha256(data).hexdigest()
   print(label,'http',res.status,'bytes',len(data),'sha256',digest)
   hashes.append(digest)
 except Exception as e: raise RuntimeError(label+' download failed: '+type(e).__name__) from None
if hashes[0]!=hashes[1]: raise RuntimeError('Avatar content mismatch')
print('AVATAR_SYNC_SHA256_OK')
