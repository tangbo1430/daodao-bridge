"""PHP host only. Update one non-secret flag after compatibility deployment."""
from pathlib import Path
import subprocess,shutil,time,os,sys
enabled='--disable' not in sys.argv
flag='true' if enabled else 'false'
root=Path('/root/IM/YCH-IM-deploy');env=root/'.env';old=env.read_text()
backup=root/('.env.before-strict-registration-'+time.strftime('%Y%m%d%H%M%S'))
shutil.copy2(env,backup);os.chmod(backup,0o600)
lines=old.splitlines(keepends=True);found=False;result=[]
for line in lines:
 if line.startswith('IM_BRIDGE_STRICT_REGISTRATION='):
  if found:raise RuntimeError('Duplicate registration flag')
  result.append('IM_BRIDGE_STRICT_REGISTRATION='+flag+'\n');found=True
 else:result.append(line)
if not found:result.append(('' if old.endswith('\n') else '\n')+'IM_BRIDGE_STRICT_REGISTRATION='+flag+'\n')
env.write_text(''.join(result));os.chmod(env,0o600)
p=subprocess.run(['docker','compose','up','-d','--no-deps','--force-recreate','php','worker'],cwd=root,stdin=subprocess.DEVNULL,capture_output=True,text=True)
if p.returncode:
 shutil.copy2(backup,env);raise RuntimeError('Service recreation failed; original environment restored')
print('STRICT_REGISTRATION_ENABLED',enabled,'backup='+str(backup),'existing_secrets_preserved',True)
