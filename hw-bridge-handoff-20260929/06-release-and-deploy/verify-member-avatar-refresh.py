"""Live self-member avatar refresh checkpoint/CMD/loop verification."""
from pathlib import Path
exec(Path(__file__).with_name('verify-group-settings.py').read_text().split('gid=None')[0])
gid=None
def stable(*values):return hashlib.sha256(b''.join(v.encode()+b'\0' for v in values)).hexdigest()
def checkpoint(group):return cursor('member-avatar-refresh:'+stable(group,reg['tsdd_uid']))
def own_member():
 path='/v1/groups/'+gid[3:]+('/members?limit=1000&page=1' if state().get('group_type')==1 else '/membersync?limit=1000&version=0')
 return next(m for m in native('GET',path) if m['uid']==reg['tsdd_uid'])
def observed_commands():
 rows=sql("SELECT payload_json FROM bridge_inbox_event WHERE source_system='tangseng' AND payload_json LIKE '%"+gid[3:]+"%' ORDER BY created_at DESC LIMIT 100")
 found=set()
 for raw in rows.splitlines():
  p=json.loads(raw);payload=p.get('payload',{})
  if isinstance(payload,str):
   try:payload=json.loads(base64.b64decode(payload))
   except (ValueError,TypeError):continue
  if not isinstance(payload,dict):continue
  param=payload.get('param',{})
  if payload.get('cmd')=='memberUpdate' and param.get('group_no')==gid[3:] and param.get('uid')==reg['tsdd_uid']:
   found.add(p.get('message_id'))
 return found
try:
 g=bridge('group_create',members=[peer],name='头像成员通知-'+run[:6],request_id=run)
 gid=g['group_id'];audit['group_id']=gid;save();print('TEMP_GROUP',gid,flush=True)
 upload=cursor('avatar-upload:'+stable(reg['tsdd_uid'],''))
 assert checkpoint(gid[3:])['aux']==upload['aux']
 for _ in range(20):
  commands=observed_commands()
  if commands:break
  time.sleep(1)
 assert len(commands)==1,'Expected one self-member CMD'
 print('NATIVE_MEMBER_UPDATE_CMD_OBSERVED',len(commands),flush=True)
 member_before=own_member();seq=profile()['NextMsgSeq']
 for _ in range(3):bridge('group_sync',group_id=gid)
 time.sleep(3)
 member_after=own_member()
 assert member_before['version']==member_after['version'],'Repeated sync changed member version'
 assert len(observed_commands())==1,'Notification loop'
 assert profile()['NextMsgSeq']==seq,'Unexpected visible Tencent group notification'
 print('MEMBER_REFRESH_NO_REPEAT_OR_CHAT_LOOP',flush=True)
 audit['result']='passed';save()
finally:
 if not gid:
  for raw in sql("SELECT group_state_json FROM group_map WHERE source_system='tangseng'").splitlines():
   saved=json.loads(raw)
   if saved.get('request_owner')==owner and saved.get('request_id')==run:gid=saved['tencent_id'];break
 if gid:
  bridge('group_disband',group_id=gid);audit['cleanup']='passed';save();print('TEMP_GROUP_CLEANED',gid,flush=True)
