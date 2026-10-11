import json,subprocess
q="SELECT scope_key,JSON_UNQUOTE(JSON_EXTRACT(cursor_json,'$.value')),JSON_UNQUOTE(JSON_EXTRACT(cursor_json,'$.started_at')) FROM bridge_business_cursor WHERE scope_key IN ('account:74629','commands:74629','conversations:74629','profile-refresh:74629');"
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
print(p.stdout)
if p.returncode: print(p.stderr[:300])
