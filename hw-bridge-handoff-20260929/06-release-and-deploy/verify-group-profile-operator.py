"""Real group profile attribution, echo and cleanup checks. No credentials printed."""
from pathlib import Path
exec(Path(__file__).with_name('verify-group-settings.py').read_text().split('gid=None')[0])
gid=None

def notices():
 r=tx('group_msg_get_simple',{'GroupId':gid,'ReqMsgNumber':100})
 if r.get('ErrorCode')!=0:raise RuntimeError('Tencent history query failed')
 result=[]
 for m in r.get('RspMsgList',[]):
  b=m.get('MsgBody')
  if isinstance(b,str):
   try:b=json.loads(b)
   except ValueError:continue
  if isinstance(b,dict) and 'MsgGroupNewInfo' in b:
   result.append({'seq':m.get('MsgSeq'),'operator':b.get('Operator_Account'),'changes':b['MsgGroupNewInfo'],'extra':b.get('MsgOperatorMemberExtraInfo')})
 return result

def wait_change(before):
 for _ in range(25):
  found=[n for n in notices() if n['seq'] not in before]
  if found:return found
  time.sleep(1)
 raise RuntimeError('No Tencent profile notification')

try:
 g=bridge('group_create',members=[peer],name='操作人验收-'+run[:6],request_id=run)
 gid=g['group_id'];audit['group_id']=gid;save();print('TEMP_GROUP',gid,flush=True)
 time.sleep(3)
 assert profile().get('Introduction','')=='','Synthetic introduction should not be set'
 for mode in ['facade','native']:
  before={n['seq'] for n in notices()}
  value=('A名称-' if mode=='facade' else 'B名称-')+run[:6]
  if mode=='facade':update('name',value)
  else:native('PUT','/v1/groups/'+gid[3:],{'name':value})
  ns=wait_change(before)
  assert all(n['operator']==owner for n in ns),'Wrong real operator on rename'
  assert profile()['Name']==value
  print('PROFILE_OPERATOR_OK',mode,'name',len(ns),flush=True)
  audit['steps'].append({'check':'operator','mode':mode,'field':'name','operator':owner,'notices':len(ns)});save()
 before={n['seq'] for n in notices()}
 update('notice','操作人公告-'+run[:6])
 ns=wait_change(before);assert all(n['operator']==owner for n in ns)
 print('PROFILE_OPERATOR_OK facade notice',len(ns),flush=True)
 avatar=sql("SELECT avatar_url FROM bridge_identity_mapping WHERE source_system='tencent' AND source_id='%s' AND entity_type='user' LIMIT 1"%owner)
 if not avatar:raise RuntimeError('Authorized account has no avatar fixture')
 before={n['seq'] for n in notices()}
 update('avatar',avatar)
 ns=wait_change(before);assert all(n['operator']==owner for n in ns)
 print('PROFILE_OPERATOR_OK facade avatar',len(ns),flush=True)
 # The original event time and field must match the durable echo records.
 time.sleep(5)
 before={n['seq'] for n in notices()}
 for _ in range(3):bridge('group_sync',group_id=gid)
 time.sleep(5)
 assert {n['seq'] for n in notices()}==before,'Repeated sync emitted additional notices'
 n=native('POST','/v1/message/channel/sync',{'channel_id':gid[3:],'channel_type':2,'start_message_seq':0,'end_message_seq':0,'limit':100,'pull_mode':0,'device_uuid':'hwbridge-profile-operator-audit'})
 changes=[m['payload'] for m in n.get('messages',[]) if m.get('payload',{}).get('type')==1005]
 for field in ['name','notice']:
  expected=2 if field=='name' else 1
  assert sum(field in p.get('data',{}) for p in changes)==expected,'Native duplicate profile update'
 print('PROFILE_NO_DUPLICATE_OR_LOOP_OK',flush=True)
 audit['result']='passed';save()
finally:
 if not gid:
  for raw in sql("SELECT group_state_json FROM group_map WHERE source_system='tangseng'").splitlines():
   saved=json.loads(raw)
   if saved.get('request_owner')==owner and saved.get('request_id')==run:gid=saved['tencent_id'];break
 if gid:
  bridge('group_disband',group_id=gid)
  assert state()['status']==2
  audit['cleanup']='passed';save();print('TEMP_GROUP_CLEANED',gid,flush=True)
