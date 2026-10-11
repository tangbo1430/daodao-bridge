import json,os,hmac,hashlib,urllib.request,urllib.error
cfg={}
for line in open('/opt/hw-bridge/.env'):
 k,sep,v=line.strip().partition('=')
 if sep: cfg[k]=v.strip().strip('"').strip("'")
audit=json.load(open('/tmp/hwbridge-core-group-verification.json'))
body=json.dumps({'tencent_user_id':'74629','group_id':audit['group_id']},separators=(',',':')).encode()
req=urllib.request.Request('http://127.0.0.1:8090/api/bridge/group_disband',data=body,headers={'Content-Type':'application/json','X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()})
try:
 with urllib.request.urlopen(req,timeout=30) as r: print('duplicate_disband',r.status,json.dumps(json.load(r),ensure_ascii=False))
except urllib.error.HTTPError as e: print('duplicate_disband',e.code,json.dumps(json.load(e),ensure_ascii=False))
