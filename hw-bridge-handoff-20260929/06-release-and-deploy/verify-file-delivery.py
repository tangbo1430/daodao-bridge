"""Read-only CVM verification: Tencent roaming -> public media download -> SHA256.
Arguments are delivered C2C MsgKeys; credentials/URLs are never printed.
"""
import json, time, hmac, hashlib, base64, zlib, urllib.request, urllib.parse
import subprocess, sys, re

cfg = {}
for line in open('/opt/hw-bridge/.env'):
    k, sep, v = line.strip().partition('=')
    if sep: cfg[k] = v.strip().strip('"').strip("'")
now = int(time.time())
app, admin, ttl = int(cfg['TENCENT_SDK_APP_ID']), cfg['TENCENT_ADMIN_ACCOUNT'], 600
plain = f'TLS.identifier:{admin}\nTLS.sdkappid:{app}\nTLS.time:{now}\nTLS.expire:{ttl}\n'
signature = base64.b64encode(hmac.new(cfg['TENCENT_SECRET_KEY'].encode(), plain.encode(), hashlib.sha256).digest()).decode()
token = {'TLS.ver':'2.0','TLS.identifier':admin,'TLS.sdkappid':app,'TLS.expire':ttl,'TLS.time':now,'TLS.sig':signature}
usersig = base64.b64encode(zlib.compress(json.dumps(token,separators=(',',':')).encode())).decode().replace('+','*').replace('/','-').replace('=','_')
query = urllib.parse.urlencode({'sdkappid':app,'identifier':admin,'usersig':usersig,'random':now,'contenttype':'json'})

def sql(q):
    p = subprocess.run(['docker','compose','exec','-T','mysql','sh','-c',
        'export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],
        cwd='/opt/hw-bridge', input=q, text=True, capture_output=True)
    assert p.returncode == 0, 'Database lookup failed'
    return p.stdout.splitlines()

assert len(sys.argv) > 1, 'C2C message keys required'
for key in sys.argv[1:]:
    assert re.fullmatch(r'[0-9_]{1,100}', key), 'Invalid message key'
    records = sql("SELECT record_json FROM bridge_message_record WHERE tencent_key='%s';" % key)
    assert len(records) == 1, 'Expected one message record'
    v = json.loads(records[0])
    payload = {'Operator_Account':v['tencent_from'],'Peer_Account':v['tencent_to'],
        'MinTime':v['occurred_at']-5,'MaxTime':int(time.time()),'MaxCnt':100}
    req = urllib.request.Request(cfg.get('TENCENT_REST_BASE_URL','https://adminapisgp.im.qcloud.com')+'/v4/openim/admin_getroammsg?'+query,
        data=json.dumps(payload).encode(), headers={'Content-Type':'application/json'})
    try:
        with urllib.request.urlopen(req, timeout=30) as r: result = json.load(r)
    except Exception: raise RuntimeError('Tencent lookup transport failed') from None
    assert result.get('ErrorCode') == 0, 'Tencent lookup rejected'
    messages = [m for m in result.get('MsgList',[]) if m.get('MsgKey') == key]
    assert len(messages) == 1, 'Message absent from roaming result'
    files = [p['MsgContent'] for p in messages[0].get('MsgBody',[]) if p.get('MsgType') == 'TIMFileElem']
    assert len(files) == 1, 'Expected file message'
    file = files[0]
    url = urllib.parse.urlsplit(file['Url'])
    base = urllib.parse.urlsplit(cfg['BRIDGE_MEDIA_PUBLIC_URL'])
    assert (url.scheme,url.netloc,url.path) == (base.scheme,base.netloc,base.path) and url.scheme == 'https', 'File does not use HTTPS media gateway'
    object_id = urllib.parse.parse_qs(url.query).get('id',[''])[0]
    assert re.fullmatch('[a-f0-9]{64}',object_id), 'Invalid object identity'
    objects = sql("SELECT object_json FROM media_cache WHERE sha256='%s';" % object_id)
    assert len(objects) == 1
    metadata = json.loads(objects[0])
    digest, count = hashlib.sha256(), 0
    try:
        with urllib.request.urlopen(file['Url'], timeout=120) as response:
            assert response.status == 200
            while True:
                chunk = response.read(65536)
                if not chunk: break
                count += len(chunk)
                digest.update(chunk)
    except Exception: raise RuntimeError('Public file download failed') from None
    assert count == file['FileSize'] == metadata['size'], 'Downloaded file size differs'
    assert digest.hexdigest() == object_id, 'Downloaded content checksum differs'
    print(json.dumps({'result':'TENCENT_FILE_PUBLIC_SHA256_OK','message_key':key,'bytes':count,'sha256':object_id}))
