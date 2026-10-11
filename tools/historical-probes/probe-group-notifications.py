import pathlib,sys,json,urllib.request,time
exec(pathlib.Path('/tmp/verify-group-business.py').read_text().split('gid=None')[0])
def native(method,path,body=None):
 req=urllib.request.Request(cfg['TANGSENG_BASE_URL'].rstrip('/')+path,method=method,data=None if body is None else json.dumps(body).encode(),headers={'Content-Type':'application/json','token':token})
 try:
  with urllib.request.urlopen(req,timeout=30) as r:return json.load(r)
 except Exception as e:raise RuntimeError('native '+type(e).__name__) from None
token=''
for key in ['TANGSENG_SHADOW_PASSWORD','TANGSENG_LEGACY_SHADOW_PASSWORD']:
 if not cfg.get(key):continue
 try:
  login=native('POST','/v1/user/usernamelogin',{'username':'tx_'+owner,'password':cfg[key],'flag':2,'device':{'device_id':'hwbridge-tx_'+owner,'device_name':'Huawei Bridge','device_model':'bridge'}})
  token=login.get('data',login).get('token','')
  if token:break
 except RuntimeError:pass
assert token
gid=None
def snapshot(label):
 g=native('GET','/v1/groups/'+gid[3:])
 msgs=native('POST','/v1/message/channel/sync',{'channel_id':gid[3:],'channel_type':2,'start_message_seq':0,'end_message_seq':0,'limit':100,'pull_mode':0,'device_uuid':'hwbridge-feedback-audit'})
 print(label,json.dumps({'group_fields':list(g),'settings':{k:v for k,v in g.items() if 'remind' in k},'events':[{'seq':m.get('message_seq'),'type':m.get('payload',{}).get('type'),'red_dot':m.get('header',{}).get('red_dot')} for m in msgs.get('messages',[])]},ensure_ascii=True),flush=True)
try:
 g=bridge('group_create',members=[peer],name='群通知验收-'+run[:6],request_id=run);gid=g['group_id'];print('temporary_group',gid,flush=True)
 snapshot('created')
 bridge('group_remove',group_id=gid,members=[peer]);time.sleep(2);snapshot('removed_default')
 native('PUT','/v1/groups/'+gid[3:]+'/setting',{'join_group_remind':1})
 bridge('group_invite',group_id=gid,members=[peer]);time.sleep(2)
 bridge('group_remove',group_id=gid,members=[peer]);time.sleep(2);snapshot('removed_remind_on')
finally:
 if gid:bridge('group_disband',group_id=gid);print('TEMP_GROUP_CLEANED',gid)

