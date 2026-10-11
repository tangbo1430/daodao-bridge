"""Append only non-secret internal endpoint settings; preserve every existing byte."""
from pathlib import Path
import os,shutil,time
p=Path('/opt/hw-bridge/.env');original=p.read_text();present={}
for line in original.splitlines():
 k,sep,v=line.partition('=')
 if sep:present[k]=v.strip().strip('"').strip("'")
settings={'BRIDGE_PHP_REGISTRATION_LOOKUP_URL':'https://admin.awhaha.com/internal/bridge/registration_lookup','BRIDGE_PHP_ACCOUNT_STATUS_URL':'https://admin.awhaha.com/internal/bridge/account_status'}
extra=[]
for k,v in settings.items():
 if k in present:
  if present[k]!=v:raise RuntimeError('Existing internal endpoint differs; no changes applied')
 else:extra.append(k+'='+v)
if extra:
 backup=p.with_name('.env.before-business-closure-'+time.strftime('%Y%m%d%H%M%S'))
 shutil.copy2(p,backup);os.chmod(backup,0o600)
 with p.open('a') as f:f.write(('' if original.endswith('\n') else '\n')+'\n'.join(extra)+'\n')
 os.chmod(p,0o600)
print('CLOSURE_ENDPOINTS_CONFIGURED',len(extra),'existing_settings_preserved',True)
