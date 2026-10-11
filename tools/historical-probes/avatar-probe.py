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

peer='0cb644b2e1304dc59ffead8af5170ba9'
r=tx('portrait_get',{'To_Account':['74629',peer],'TagList':['Tag_Profile_IM_Nick','Tag_Profile_IM_Image']},'profile')
for user in r.get('UserProfileItem',[]):
 print('tencent',user.get('To_Account'),'result',user.get('ResultCode'))
 for item in user.get('ProfileItem',[]):
  if item['Tag'].endswith('Nick'): print('nickname',item.get('Value'))
  else:
   u=item.get('Value','');print('avatar_present',bool(u),'host',urllib.parse.urlsplit(u).hostname)
   if u:
    try:
     resp=urllib.request.urlopen(u,timeout=30);data=resp.read(4*1024*1024);print('avatar_http',resp.status,'bytes',len(data),'sha256',hashlib.sha256(data).hexdigest())
    except Exception as e: print('avatar_error',type(e).__name__)
for uid in [peer,'f278d20e4ef44eefb00ad403720c9d63']:
 try:
  resp=urllib.request.urlopen('http://10.0.0.100:8090/v1/users/'+uid+'/avatar',timeout=30);data=resp.read(4*1024*1024);print('tang_avatar',uid,resp.status,len(data),hashlib.sha256(data).hexdigest())
 except Exception as e: print('tang_avatar_error',uid,type(e).__name__)
q="SELECT peer_id,status,COALESCE(JSON_UNQUOTE(JSON_EXTRACT(friend_state_json,'$.name')),''),COALESCE(LENGTH(JSON_UNQUOTE(JSON_EXTRACT(friend_state_json,'$.avatar'))),0),(SELECT COUNT(*) FROM bridge_identity_mapping m WHERE m.source_system='tangseng' AND m.source_id=peer_id) AS imported FROM friend_link WHERE owner_id='74629'; SELECT source_system,source_id,display_name,LENGTH(avatar_url) FROM bridge_identity_mapping WHERE source_id IN ('74629','"+peer+"');"
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
print('bridge_mapping',p.stdout)
