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
 n=native('POST','/v1/message/channel/sync',{'channel_id':gid[3:],'channel_type':2,'start_message_seq':0,'end_message_seq':0,'limit':100,'pull_mode':0,'device_uuid':'hwbridge-role-audit'})
 t=tx('group_msg_get_simple',{'GroupId':gid,'ReqMsgNumber':100})
 def summary(m):
  b=m.get('MsgBody')
  return {'seq':m.get('MsgSeq'),'op':b.get('OpType') if isinstance(b,dict) else 'custom','roles':[p.get('Role') for p in b.get('MsgMemberExtraInfo',[])] if isinstance(b,dict) else []}
 print(label,json.dumps({'tencent_role':(member() or {}).get('Role'),'native_response_keys':list(n),'native_self':login.get('data',login).get('uid'),'creator':state().get('creator'),'native':[{'seq':m.get('message_seq'),'type':m.get('payload',{}).get('type'),'content':m.get('payload',{}).get('content'),'marker':m.get('payload',{}).get('bridge_event_id')} for m in n.get('messages',[])],'tencent':[summary(m) for m in t.get('RspMsgList',[])]},ensure_ascii=True),flush=True)

try:
 g=bridge('group_create',members=[peer],name='管理员通知验收-'+run[:6],request_id=run);gid=g['group_id'];print('temporary_group',gid,flush=True)
 update('manager',members=[peer]);time.sleep(5);snapshot('facade_promote')
 native('DELETE','/v1/groups/'+gid[3:]+'/managers',[peer]);time.sleep(6);bridge('group_sync',group_id=gid);snapshot('native_demote')
finally:
 if gid:bridge('group_disband',group_id=gid);print('TEMP_GROUP_CLEANED',gid)
