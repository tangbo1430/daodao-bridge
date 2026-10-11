import json,subprocess,re
jobs=json.load(open('/tmp/hwbridge-state-command-verification.json'))
ids=[x['job_id'] for x in jobs]
assert all(re.fullmatch('[a-f0-9]{64}',x) for x in ids)
q="SELECT status,COUNT(*) FROM bridge_outbox_job WHERE id IN ("+','.join("'"+x+"'" for x in ids)+") GROUP BY status;"
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
print(p.stdout)
