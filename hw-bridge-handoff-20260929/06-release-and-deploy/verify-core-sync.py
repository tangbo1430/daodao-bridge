import json,sys,hmac,hashlib,urllib.request,urllib.error
cfg={}
for line in open('/opt/hw-bridge/.env'):
 k,sep,v=line.strip().partition('=')
 if sep: cfg[k]=v.strip().strip('"').strip("'")
owner,peer=sys.argv[1:3]
for value in [owner,peer]:
 if not value or len(value)>64 or not all(c.isalnum() or c in '_-' for c in value):raise RuntimeError('Invalid account ID')
for round_no in range(2):
 for action,payload in [('session_sync',{'tencent_user_id':owner}),('tencent_roam_sync',{'tencent_user_id':owner}),('message_extra_sync',{'tencent_user_id':owner,'channel_id':peer,'channel_type':1})]:
  body=json.dumps(payload,separators=(',',':')).encode()
  req=urllib.request.Request('http://127.0.0.1:8090/api/bridge/'+action,data=body,headers={'Content-Type':'application/json','X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()})
  try:
   with urllib.request.urlopen(req,timeout=90) as r: status=r.status; result=json.load(r)
  except urllib.error.HTTPError as e: status=e.code;result=json.load(e)
  print('round',round_no+1,action,'http',status,'result',json.dumps(result,ensure_ascii=False))
  if status!=200 or result.get('code')!=200: raise RuntimeError('Core sync failed')
print('CORE_SESSION_ROAM_EXTRA_OK')
