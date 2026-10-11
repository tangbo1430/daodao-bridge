"""Read-only native diagnostics; credentials remain in server memory."""
import json, subprocess, hashlib, base64, urllib.request, sys, re
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
cfg = {}
for line in open('/opt/hw-bridge/.env'):
    k, sep, v = line.strip().partition('=')
    if sep: cfg[k] = v.strip().strip('"').strip("'")
owner, channel, ct = sys.argv[1:4]
assert re.fullmatch(r'[A-Za-z0-9_-]+', owner) and re.fullmatch(r'[A-Za-z0-9_-]+', channel)
def sql(q):
    p = subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],cwd='/opt/hw-bridge',input=q,text=True,capture_output=True)
    if p.returncode: raise RuntimeError('query failed')
    return p.stdout.strip()
def cursor(key): return json.loads(sql("SELECT cursor_json FROM bridge_business_cursor WHERE scope_key='%s'" % key))
reg = json.loads(cursor('registration:'+cursor('account-registration:'+owner)['aux'])['aux'])
raw = base64.urlsafe_b64decode(reg['password_cipher']+'='*(-len(reg['password_cipher'])%4))
password = AESGCM(hashlib.sha256(('hw-bridge-registration-v1\0'+cfg['BRIDGE_ENCRYPTION_KEY']).encode()).digest()).decrypt(raw[:12],raw[12:],None).decode()
def call(method,path,body=None,token=''):
    req = urllib.request.Request(cfg['TANGSENG_BASE_URL'].rstrip('/')+path,data=None if body is None else json.dumps(body).encode(),method=method,headers={'Content-Type':'application/json','token':token})
    try:
        with urllib.request.urlopen(req,timeout=30) as r: return json.load(r)
    except Exception as e: raise RuntimeError('native request failed: '+type(e).__name__) from None
login = call('POST','/v1/user/usernamelogin',{'username':reg['username'],'password':password,'flag':2,'device':{'device_id':'hwbridge'+reg['tsdd_uid'],'device_name':'华为互通','device_model':'hw-bridge'}})
login=login.get('data',login);assert login['uid']==reg['tsdd_uid'];token=login['token']
if ct=='2':
    g=call('GET','/v1/groups/'+channel,token=token)
    print('group',json.dumps({k:v for k,v in g.items() if k in ['group_no','status','revoke_remind','join_group_remind','member_count','quit','forbidden','kick_remind']},ensure_ascii=True))
messages=call('POST','/v1/message/channel/sync',{'channel_id':channel,'channel_type':int(ct),'start_message_seq':0,'end_message_seq':0,'limit':60,'pull_mode':0,'device_uuid':'hwbridge-feedback-audit'},token)
for m in messages.get('messages',[]):
    p=m.get('payload',{})
    if isinstance(p,str):
        try:p=json.loads(base64.b64decode(p))
        except: p={}
    print(json.dumps({'id':m.get('message_idstr',str(m.get('message_id'))),'from':m.get('from_uid'),'seq':m.get('message_seq'),'header':m.get('header'),'setting':m.get('setting'),'revoke':m.get('revoke'),'extra':{k:v for k,v in (m.get('message_extra') or {}).items() if k in ['revoke','readed','readed_count']},'payload_keys':list(p),'type':p.get('type'),'cmd':p.get('cmd'),'param':p.get('param') if p.get('cmd') in ['userAvatarUpdate','channelUpdate','memberUpdate','syncMessageExtra'] else None,'width':p.get('width'),'height':p.get('height'),'url_relative':bool(p.get('url')) and not p['url'].startswith('http')},ensure_ascii=True))
