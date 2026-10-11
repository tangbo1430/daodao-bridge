"""Check the existing native self-member refresh contract in a disposable group."""
from pathlib import Path
exec(Path(__file__).with_name('verify-group-settings.py').read_text().split('gid=None')[0])
gid=None
try:
 g=bridge('group_create',members=[peer],name='成员刷新验收-'+run[:6],request_id=run)
 gid=g['group_id'];audit['group_id']=gid;save();print('TEMP_GROUP',gid,flush=True)
 path='/v1/groups/'+gid[3:]+('/members?limit=1000&page=1' if state().get('group_type')==1 else '/membersync?limit=1000&version=0')
 before=next(m for m in native('GET',path) if m['uid']==reg['tsdd_uid'])
 result=native('PUT','/v1/groups/'+gid[3:]+'/members/'+reg['tsdd_uid'],{})
 after=next(m for m in native('GET',path) if m['uid']==reg['tsdd_uid'])
 assert int(after['version'])>int(before['version'])
 for field in ['uid','name','remark','role','status','is_deleted','avatar']:
  assert before.get(field)==after.get(field),'Member business field changed: '+field
 evidence={'evidence':'Observed native self-member PUT on an authorized temporary group; identifiers and versions sanitized','request':{'method':'PUT','path':'/v1/groups/fixture-group/members/fixture-owner','body':{}},'response':result,'before':{'uid':'fixture-owner','version':1,'avatar':before.get('avatar','')},'after':{'uid':'fixture-owner','version':2,'avatar':after.get('avatar','')},'business_fields_unchanged':True}
 Path('/tmp/member-avatar-refresh-observed-20261002.json').write_text(json.dumps(evidence,ensure_ascii=False,indent=2))
 print('SELF_MEMBER_REFRESH_VERSION_ADVANCED_WITHOUT_PROFILE_CHANGE',flush=True)
 audit['result']='passed';save()
finally:
 if not gid:
  for raw in sql("SELECT group_state_json FROM group_map WHERE source_system='tangseng'").splitlines():
   saved=json.loads(raw)
   if saved.get('request_owner')==owner and saved.get('request_id')==run:gid=saved['tencent_id'];break
 if gid:
  bridge('group_disband',group_id=gid);audit['cleanup']='passed';save();print('TEMP_GROUP_CLEANED',gid,flush=True)
