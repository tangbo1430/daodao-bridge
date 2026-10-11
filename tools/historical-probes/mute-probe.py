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
group='dd_7825169c0398401ca16ed59d02e0542b'
for cmd,body in [('get_group_info',{'GroupIdList':[group],'ResponseFilter':{'GroupBaseInfoFilter':['MuteAllMember','LastMsgTime','NextMsgSeq','Name']}}),('get_group_member_info',{'GroupId':group,'Limit':100,'Offset':0})]:
 r=tx(cmd,body)
 if cmd=='get_group_member_info':
  print(cmd,json.dumps({'ErrorCode':r.get('ErrorCode'),'MemberList':[{k:v for k,v in m.items() if k in ('Member_Account','Role','ShutUpUntil')} for m in r.get('MemberList',[])]}))
 else: print(cmd,json.dumps(r))
r=tx('group_msg_get_simple',{'GroupId':group,'ReqMsgNumber':20})
print('recent',json.dumps({'ErrorCode':r.get('ErrorCode'),'messages':[{'seq':m.get('MsgSeq'),'time':m.get('MsgTimeStamp'),'op':m.get('MsgBody',{}).get('OpType') if isinstance(m.get('MsgBody'),dict) else 'message','mute_all':m.get('MsgBody',{}).get('MsgGroupNewInfo',{}).get('ShutupAll') if isinstance(m.get('MsgBody'),dict) else None,'mute_seconds':[x.get('ShutupTime') for x in m.get('MsgBody',{}).get('MsgMemberInfo',[])] if isinstance(m.get('MsgBody'),dict) else []} for m in r.get('RspMsgList',[])]}))
q="SELECT command_name,COUNT(*),MIN(created_at),MAX(created_at) FROM bridge_inbox_event WHERE created_at > UNIX_TIMESTAMP()-600 GROUP BY command_name;"
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
print('inbox_recent_counts',p.stdout)
