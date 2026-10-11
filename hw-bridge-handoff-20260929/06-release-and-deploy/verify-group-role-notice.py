import pathlib,sys,json,urllib.request,time
exec(pathlib.Path(__file__).with_name('verify-group-business.py').read_text().split('gid=None')[0])
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

def history():
 n=native('POST','/v1/message/channel/sync',{'channel_id':gid[3:],'channel_type':2,'start_message_seq':0,'end_message_seq':0,'limit':100,'pull_mode':0,'device_uuid':'hwbridge-role-audit'})
 assert 'messages' in n, 'Missing history response'
 return [m for m in n['messages'] if m.get('payload',{}).get('bridge_event_id','').startswith('bridge-group-role:')]

def verify(label,role,count):
 for attempt in range(15):
  notices=history()
  if (member() or {}).get('Role')==role and len(notices)==count:break
  time.sleep(2)
 else:raise RuntimeError('role/notice convergence failed: '+label)
 state_member=next(m for m in state()['members'] if m['uid']==peer)
 assert state_member['role']==(2 if role=='Admin' else 0)
 p=notices[-1]['payload']
 assert p['type']==2000 and ('已被设置为群管理员' if role=='Admin' else '的群管理员身份已取消') in p['content']
 t=tx('group_msg_get_simple',{'GroupId':gid,'ReqMsgNumber':100})
 assert t.get('ErrorCode')==0
 ops=[m['MsgBody']['OpType'] for m in t.get('RspMsgList',[]) if isinstance(m.get('MsgBody'),dict) and m['MsgBody'].get('OpType') in [4,5]]
 assert len(ops)==count and ops[0]==(4 if role=='Admin' else 5)
 for m in t.get('RspMsgList',[]):
  if isinstance(m.get('MsgBody'),list):
   assert 'bridge-group-role:' not in json.dumps(m), 'native notice looped back to Tencent'
 print(label,json.dumps({'tencent_role':role,'native_role':state_member['role'],'native_notice_count':len(notices),'tencent_role_tip_count':len(ops),'native_type':p['type']},ensure_ascii=True),flush=True)
 return notices

try:
 g=bridge('group_create',members=[peer],name='管理员通知验收-'+run[:6],request_id=run);gid=g['group_id'];audit['group_id']=gid;save();print('temporary_group',gid,flush=True)
 me=next(m for m in state()['members'] if m.get('tencent_id')==owner)
 assert me['uid']==login.get('data',login).get('uid'), 'native actor mismatch'
 update('manager',members=[peer]);verify('FACADE_PROMOTE_OK','Admin',1)
 update('member',members=[peer]);verify('FACADE_DEMOTE_OK','Member',2)
 native('POST','/v1/groups/'+gid[3:]+'/managers',[peer]);verify('NATIVE_PROMOTE_OK','Admin',3)
 native('DELETE','/v1/groups/'+gid[3:]+'/managers',[peer]);notices=verify('NATIVE_DEMOTE_OK','Member',4)
 before=profile()['NextMsgSeq']
 for _ in range(3):bridge('group_sync',group_id=gid)
 time.sleep(3)
 assert profile()['NextMsgSeq']==before and len(history())==4
 assert len({m['payload']['bridge_event_id'] for m in notices})==4
 print('REPEATED_SYNC_NO_DUPLICATE_OR_LOOP_OK',flush=True)
 audit['result']='passed';save()
finally:
 if gid:
  bridge('group_disband',group_id=gid)
  assert state()['status']==2
  audit['cleanup']='passed';save();print('TEMP_GROUP_CLEANED',gid,flush=True)
