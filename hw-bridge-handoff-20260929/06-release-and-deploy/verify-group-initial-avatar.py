"""Fresh-group avatar notices, repeated sync and later real operator checks."""
from pathlib import Path
exec(Path(__file__).with_name('verify-group-settings.py').read_text().split('gid=None')[0])
gid=None
def notices():
 r=tx('group_msg_get_simple',{'GroupId':gid,'ReqMsgNumber':100})
 if r.get('ErrorCode')!=0:raise RuntimeError('History unavailable')
 out=[]
 for m in r.get('RspMsgList',[]):
  body=m.get('MsgBody')
  if isinstance(body,str):
   try:body=json.loads(body)
   except ValueError:continue
  if isinstance(body,dict) and body.get('MsgGroupNewInfo'):
   out.append({'seq':m.get('MsgSeq'),'operator':body.get('Operator_Account'),'fields':sorted(body['MsgGroupNewInfo'])})
 return out
try:
 for mode in ['facade','native']:
  if mode=='facade':
   g=bridge('group_create',members=[peer],name='首次头像验收-'+run[:5],request_id=run);gid=g['group_id']
  else:
   g=native('POST','/v1/group/create',{'name':'原生首次头像-'+run[:5],'members':[peer]});gid='dd_'+g['group_no']
  audit['group_id']=gid;save();print('TEMP_GROUP',mode,gid,flush=True)
  for _ in range(25):
   p=tx('get_group_info',{'GroupIdList':[gid]})
   gs=p.get('GroupInfo',[])
   if gs and gs[0].get('ErrorCode',0)==0 and gs[0].get('FaceUrl'):break
   time.sleep(1)
  else:raise RuntimeError('Initial avatar not mirrored')
  time.sleep(4)
  initial=notices()
  assert not any(n['operator']==admin and 'GroupFaceUrl' in n['fields'] for n in initial),'Initial administrator avatar notice'
  print('INITIAL_AVATAR_NO_ADMIN_NOTICE',mode,len(initial),flush=True)
  before={n['seq'] for n in initial}
  for _ in range(3):bridge('group_sync',group_id=gid)
  time.sleep(3)
  assert {n['seq'] for n in notices()}==before,'Initialization notice repeated'
  update('name','新群改名-'+run[:5])
  for _ in range(15):
   changes=[n for n in notices() if n['seq'] not in before]
   if changes:break
   time.sleep(1)
  assert len(changes)==1 and changes[0]['operator']==owner,'Later operator attribution regressed'
  print('REPEAT_SYNC_AND_LATER_OPERATOR_OK',mode,flush=True)
  audit['steps'].append({'mode':mode,'initial_avatar':True,'initial_notice_count':len(initial),'initial_admin_avatar_notice':False,'repeat_sync_unchanged':True,'later_operator_correct':True});save()
  bridge('group_disband',group_id=gid);print('TEMP_GROUP_CLEANED',gid,flush=True);gid=None
 audit['result']='passed';audit['cleanup']='passed';save()
finally:
 if gid:
  try:bridge('group_disband',group_id=gid)
  except RuntimeError:
   native('DELETE','/v1/groups/'+gid[3:]+'/disband')
  print('TEMP_GROUP_CLEANED',gid,flush=True)
