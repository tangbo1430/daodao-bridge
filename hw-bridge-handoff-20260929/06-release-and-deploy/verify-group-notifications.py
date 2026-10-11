"""Real native settings -> Tencent notice and name verification; cleans its own group."""
from pathlib import Path
exec(Path(__file__).with_name('verify-group-settings.py').read_text().split('gid=None')[0])
gid=None

def invitation_tips():
 r=tx('group_msg_get_simple',{'GroupId':gid,'ReqMsgNumber':100})
 if r.get('ErrorCode')!=0:raise RuntimeError('Tencent history query failed')
 tips=[]
 for m in r.get('RspMsgList',[]):
  if not isinstance(m.get('MsgBody'),list):continue
  for elem in m['MsgBody']:
   if elem.get('MsgType')!='TIMCustomElem':continue
   try:p=json.loads(elem['MsgContent']['Data'])
   except (ValueError,KeyError):continue
   text=p.get('text','')
   if p.get('businessID')=='bridge_tip' and ('群聊邀请确认' in text or '默认进群方式' in text):tips.append(text)
 return tips

try:
 g=bridge('group_create',members=[peer],name='群通知验收-'+run[:6],request_id=run)
 gid=g['group_id'];audit['group_id']=gid;save();print('TEMP_GROUP',gid,flush=True)
 expected=0
 for mode in ['native','facade']:
  for value in [1,0]:
   if mode=='native':native('PUT','/v1/groups/'+gid[3:],{'invite':str(value)})
   else:update('invite',str(value))
   expected+=1
   for _ in range(20):
    tips=invitation_tips()
    if len(tips)==expected and state().get('invite')==value:break
    time.sleep(1)
   else:raise RuntimeError('Invitation tip did not converge')
   assert all('{0}' not in text for text in tips)
   print('INVITATION_TIP_OK',mode,value,expected,flush=True)
 new_name='群名同步验证-'+run[:4]
 native('PUT','/v1/groups/'+gid[3:],{'name':new_name})
 for _ in range(20):
  if profile().get('Name')==new_name and state().get('name')==new_name:break
  time.sleep(1)
 else:raise RuntimeError('Native rename callback failed')
 print('NATIVE_NAME_EVENT_SYNC_OK',flush=True)
 g=bridge('group_sync',group_id=gid);assert g['name']==new_name
 for _ in range(3):bridge('group_sync',group_id=gid)
 time.sleep(3)
 assert len(invitation_tips())==4
 n=native('POST','/v1/message/channel/sync',{'channel_id':gid[3:],'channel_type':2,'start_message_seq':0,'end_message_seq':0,'limit':100,'pull_mode':0,'device_uuid':'hwbridge-settings-notice-audit'})
 updates=[m for m in n.get('messages',[]) if m.get('payload',{}).get('type')==1005 and 'invite' in m['payload'].get('data',{})]
 assert len(updates)==4
 assert not any('群聊邀请确认' in m.get('payload',{}).get('content','') and m.get('payload',{}).get('type')==1 for m in n.get('messages',[]))
 print('INVITE_TIPS_NO_DUPLICATE_OR_LOOP_OK',flush=True)
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
