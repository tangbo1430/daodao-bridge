"""Read-only status evidence on Bridge; never output cursor JSON or logs."""
import subprocess,json,hashlib,pathlib,urllib.request
def sql(q):
 p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
 if p.returncode:raise RuntimeError('Bridge database check failed')
 return p.stdout.strip().splitlines()
print('BRIDGE_SHA256',hashlib.sha256(pathlib.Path('/opt/hw-bridge/bridge').read_bytes()).hexdigest())
for path in ('healthz','readyz'):
 with urllib.request.urlopen('http://127.0.0.1:8090/'+path,timeout=10) as r:print(path,r.status,json.load(r))
rows=sql("SELECT scope_key,cursor_json FROM bridge_business_cursor WHERE scope_key LIKE 'account-status-ack:%' OR scope_key LIKE 'group-unread-day:%' OR scope_key='registration-scan';")
counts={}
for line in rows:
 key,raw=line.split('\t',1);v=json.loads(raw);kind=key.split(':')[0];counts[kind]=counts.get(kind,0)+1
print('PERSISTED_CHECKPOINT_COUNTS',json.dumps(counts))
p=subprocess.run(['docker','compose','logs','--since','5m','--no-color','bridge'],cwd='/opt/hw-bridge',text=True,capture_output=True)
logs=p.stdout+p.stderr
for marker in ('account status notification pending','registration reconciliation pending','business sync pending','panic:','tangseng proxy connecting'):
 print('LOG_COUNT',marker,logs.count(marker))
for line in sql("SELECT status,COUNT(*) FROM bridge_outbox_job GROUP BY status;"):print('OUTBOX_STATUS',line)
