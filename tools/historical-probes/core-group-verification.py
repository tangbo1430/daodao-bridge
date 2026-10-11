import json,os,uuid,hmac,hashlib,urllib.request,urllib.error
cfg={}
for line in open('/opt/hw-bridge/.env'):
 k,sep,v=line.strip().partition('=')
 if sep: cfg[k]=v.strip().strip('"').strip("'")
owner='74629';peer='0cb644b2e1304dc59ffead8af5170ba9'
audit={'request_id':'core-acceptance-'+uuid.uuid4().hex}
path='/tmp/hwbridge-core-group-verification.json'
def save():
 fd=os.open(path,os.O_CREAT|os.O_TRUNC|os.O_WRONLY,0o600)
 with os.fdopen(fd,'w') as f:json.dump(audit,f)
def bridge(action,payload):
 body=json.dumps(payload,ensure_ascii=False,separators=(',',':')).encode()
 req=urllib.request.Request('http://127.0.0.1:8090/api/bridge/'+action,data=body,headers={'Content-Type':'application/json','X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()})
 try:
  with urllib.request.urlopen(req,timeout=90) as r: result=json.load(r);status=r.status
 except urllib.error.HTTPError as e: result=json.load(e);status=e.code
 print(action,'http',status,'code',result.get('code'),'step',result.get('step'))
 if status!=200 or result.get('code')!=200: raise RuntimeError(json.dumps({k:result.get(k) for k in ('code','msg','step')},ensure_ascii=False))
 return result
save()
created=bridge('group_create',{'tencent_user_id':owner,'members':[peer],'name':'桥接验收临时群-20260930','request_id':audit['request_id']})
audit['group_id']=created['data']['group_id'];save()
try:
 again=bridge('group_create',{'tencent_user_id':owner,'members':[peer],'name':'桥接验收临时群-20260930','request_id':audit['request_id']})
 if again['data']['group_id']!=audit['group_id']: raise RuntimeError('Duplicate create produced another group')
 bridge('group_update',{'tencent_user_id':owner,'group_id':audit['group_id'],'field':'notice','value':'自动化业务验收临时群，验收后解散'})
 bridge('group_sync',{'tencent_user_id':owner,'group_id':audit['group_id']})
 audit['create_retry_profile_sync']='passed';save()
finally:
 deleted=bridge('group_disband',{'tencent_user_id':owner,'group_id':audit['group_id']})
 audit['disband']=deleted['data'];save()
print('GROUP_CREATE_RETRY_UPDATE_DISBAND_OK',audit['group_id'])
