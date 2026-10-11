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
peer,expected=sys.argv[1:3]
if not peer or len(peer)>64 or not all(c.isalnum() or c in '_-' for c in peer):raise RuntimeError('Invalid peer ID')
for attempt in range(8):
 r=tx('portrait_get',{'To_Account':[peer],'TagList':['Tag_Profile_IM_Nick','Tag_Profile_IM_Image']},'profile')
 if r.get('ErrorCode')!=0:raise RuntimeError('Tencent profile query failed')
 rows=[u for u in r.get('UserProfileItem',[]) if u.get('To_Account')==peer and u.get('ResultCode')==0]
 if len(rows)!=1:raise RuntimeError('Profile missing')
 fields={p['Tag']:p.get('Value') for p in rows[0].get('ProfileItem',[])}
 actual=fields.get('Tag_Profile_IM_Nick')
 if actual==expected:
  print('PROFILE_NICKNAME_SYNC_OK',actual,'avatar_present',bool(fields.get('Tag_Profile_IM_Image')))
  break
 if attempt==7:raise RuntimeError('Expected nickname not observed within verification window')
 time.sleep(5)