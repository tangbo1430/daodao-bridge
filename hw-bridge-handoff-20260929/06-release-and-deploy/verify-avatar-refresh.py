"""Re-sync an authorized account's existing profile; never print credentials/URLs."""
import hashlib, hmac, json, re, subprocess, sys, time, urllib.request

owner = sys.argv[1]
if not re.fullmatch(r'[A-Za-z0-9_-]{1,64}', owner):
    raise SystemExit('Invalid account')
cfg = {}
for line in open('/opt/hw-bridge/.env'):
    key, sep, value = line.strip().partition('=')
    if sep:
        cfg[key] = value.strip().strip('"').strip("'")

def sql(query):
    result = subprocess.run(['docker','compose','exec','-T','mysql','sh','-c',
        'export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],
        cwd='/opt/hw-bridge',input=query,text=True,capture_output=True)
    if result.returncode:
        raise RuntimeError('Database query failed')
    return result.stdout.strip()

uid, name, avatar = sql("SELECT target_id,display_name,avatar_url FROM bridge_identity_mapping WHERE source_system='tencent' AND source_id='%s'" % owner).split('\t')
if not name or not avatar.startswith('https://'):
    raise RuntimeError('Existing complete public profile required; no fallback profile will be invented')
assert re.fullmatch(r'[A-Za-z0-9_-]{1,64}', uid)

def native_avatar_hash():
    try:
        with urllib.request.urlopen(cfg['TANGSENG_BASE_URL'].rstrip('/')+'/v1/users/'+uid+'/avatar',timeout=30) as r:
            data=r.read(20*1024*1024+1)
            assert len(data)<=20*1024*1024
            return hashlib.sha256(data).hexdigest()
    except Exception as e:
        raise RuntimeError('Avatar fetch failed: '+type(e).__name__) from None

before=native_avatar_hash()
values=[]
for attempt in range(2):
    body=json.dumps({'tencent_user_id':owner,'display_name':name,'avatar_url':avatar},separators=(',',':')).encode()
    req=urllib.request.Request('http://127.0.0.1:8090/api/bridge/profile_sync',data=body,
        headers={'Content-Type':'application/json','X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()})
    with urllib.request.urlopen(req,timeout=15) as response: result=json.load(response)
    assert result.get('code')==200 and result.get('status')=='queued'
    event=result['event_id']; assert re.fullmatch(r'[A-Za-z0-9_-]+',event)
    for _ in range(20):
        status=sql("SELECT status FROM bridge_outbox_job WHERE event_id='%s'" % event)
        if status=='done':break
        if status=='dead':raise RuntimeError('Profile job failed')
        time.sleep(1)
    else: raise RuntimeError('Profile job not completed within 20 seconds')
    cursor=json.loads(sql("SELECT cursor_json FROM bridge_business_cursor WHERE scope_key='avatar-profile-refresh:%s'" % uid))
    assert len(cursor.get('aux',''))==64
    values.append(cursor)
    print('PROFILE_REFRESH_JOB_DONE',event)
    time.sleep(1)
assert values[0]==values[1], 'duplicate request refreshed the profile again'
after=native_avatar_hash()
assert before==after, 'existing avatar bytes changed'
for route in ['healthz','readyz']:
    assert urllib.request.urlopen('http://127.0.0.1:8090/'+route,timeout=10).status==200
print('AVATAR_PROFILE_REFRESH_CHECKPOINT_OK')
print('DUPLICATE_REFRESH_SUPPRESSED_OK')
print('EXISTING_AVATAR_PRESERVED_SHA256',after)
