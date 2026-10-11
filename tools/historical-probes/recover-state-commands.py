import json,subprocess,base64,urllib.request,hmac,hashlib,os
cfg={}
for line in open('/opt/hw-bridge/.env'):
 k,sep,v=line.strip().partition('=')
 if sep: cfg[k]=v.strip().strip('"').strip("'")
def sql(q):
 p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
 if p.returncode: raise RuntimeError('Database query failed')
 return p.stdout
q="SELECT o.id,o.status,i.payload_json FROM bridge_outbox_job o JOIN bridge_inbox_event i ON o.event_id=i.event_id WHERE o.last_error LIKE '%频道%' AND JSON_UNQUOTE(JSON_EXTRACT(i.payload_json,'$.to_uid'))='f278d20e4ef44eefb00ad403720c9d63' ORDER BY o.updated_at DESC LIMIT 10;"
jobs=[]
for row in sql(q).splitlines():
 job,status,raw=row.split('\t',2);outer=json.loads(raw);payload=outer.get('payload')
 if isinstance(payload,str): payload=json.loads(base64.b64decode(payload))
 if payload.get('cmd')!='syncMessageExtra' or outer.get('channel_id')!='0cb644b2e1304dc59ffead8af5170ba9' or outer.get('channel_type')!=1: continue
 if status!='dead': print('skip_status',job,status);continue
 jobs.append({'job_id':job,'before_status':status})
audit='/tmp/hwbridge-state-command-verification.json'
fd=os.open(audit,os.O_WRONLY|os.O_CREAT|os.O_TRUNC,0o600)
with os.fdopen(fd,'w') as out: json.dump(jobs,out)
for job in jobs:
 body=b'{}'
 req=urllib.request.Request('http://127.0.0.1:8090/admin/dead-letter/'+job['job_id'],data=body,headers={'Content-Type':'application/json','X-Bridge-Signature':hmac.new(cfg['BRIDGE_SHARED_SECRET'].encode(),body,hashlib.sha256).hexdigest()})
 with urllib.request.urlopen(req,timeout=30) as r:
  print('retry_state_command',job['job_id'],'http',r.status)
print('affected_jobs',len(jobs))
