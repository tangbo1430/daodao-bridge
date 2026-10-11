import json,subprocess,base64
q="SELECT i.command_name,i.payload_json FROM bridge_outbox_job o JOIN bridge_inbox_event i ON o.event_id=i.event_id WHERE o.id='5d64623288f0d416877c072e92110af0736527fd3a2788e8521d399d703ecb85' OR o.last_error LIKE '%频道%' ORDER BY o.updated_at DESC LIMIT 5;"
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=q,text=True,cwd='/opt/hw-bridge',capture_output=True)
if p.returncode: print('QUERY_FAILED',p.stderr[:400])
else:
 for row in p.stdout.splitlines():
  command,raw=row.split('\t',1); outer=json.loads(raw); payload=outer.get('payload',{})
  if isinstance(payload,str):
   try: payload=json.loads(base64.b64decode(payload))
   except Exception: payload=json.loads(payload)
  # Commands only: never emit tokens, text content or media links.
  print(json.dumps({'command':command,'outer_fields':list(outer),'owner':outer.get('to_uid'),'channel_id':outer.get('channel_id'),'from_uid':outer.get('from_uid'),'channel_type':outer.get('channel_type'),'payload_keys':list(payload),'payload':{k:v for k,v in payload.items() if k in ('type','cmd','param')}},ensure_ascii=False))
