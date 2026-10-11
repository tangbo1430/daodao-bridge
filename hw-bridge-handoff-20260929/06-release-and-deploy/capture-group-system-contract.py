"""Read-only: emit redacted native system packets already in Bridge Inbox."""
import base64,json,re,subprocess
query="SELECT payload_json FROM bridge_inbox_event WHERE source_system='tangseng' ORDER BY id DESC LIMIT 3000;"
p=subprocess.run(['docker','compose','exec','-T','mysql','sh','-c','export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],input=query,text=True,cwd='/opt/hw-bridge',capture_output=True)
if p.returncode:raise RuntimeError('Inbox query failed')
packets={};names=set()

def collect_names(v):
 if isinstance(v,dict):
  for k,x in v.items():
   if isinstance(x,str) and (k=='name' or k.endswith('_name')) and x:names.add(x)
   collect_names(x)
 elif isinstance(v,list):
  for x in v:collect_names(x)
def clean(v,key=''):
 if isinstance(v,dict):return {k:clean(x,str(k)) for k,x in v.items() if str(k).lower() not in ('token','vercode','password','authorization')}
 if isinstance(v,list):return [clean(x,key) for x in v]
 if isinstance(v,str):
  if key=='name' or key.endswith('_name'):return 'fixture-name'
  for name in sorted(names,key=len,reverse=True):v=v.replace(name,'fixture-name')
  return re.sub(r'[a-f0-9]{32}', 'fixture-native-id',v)
 return v
for row in p.stdout.splitlines():
 try:
  outer=json.loads(row);v=outer.get('payload',{})
  if isinstance(v,str):
   try:v=json.loads(v)
   except Exception:v=json.loads(base64.b64decode(v))
  typ=v.get('type')
  if typ in (1001,1002,1003,1004,1008,1009,1020,1021,2000) and typ not in packets:
   collect_names(v);packets[typ]=v
 except Exception:continue
print(json.dumps({'provenance':'real Bridge Inbox; identifiers and names redacted','packets':clean(packets)},ensure_ascii=False,indent=2))
