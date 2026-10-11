from pathlib import Path
import hashlib,json,zipfile,tarfile,shutil,os,sys
sys.stdout.reconfigure(encoding='utf-8')

work=Path(__file__).resolve().parent.parent
root=work/'hw-bridge-handoff-20260929'
release=root/'06-release-and-deploy'
stamp=sys.argv[1] if len(sys.argv)>1 else '20261001'
baseline=zipfile.ZipFile(work/'hw-bridge-handoff-20260929.zip')
base_names={}
for entry in baseline.infolist():
 name=entry.filename
 if not entry.flag_bits & 0x800:
  try:name=name.encode('cp437').decode('utf-8')
  except (UnicodeEncodeError,UnicodeDecodeError):pass
 base_names[name]=entry.filename
files={}
def sha(b):return hashlib.sha256(b).hexdigest()
def read(p):return Path('\\\\?\\'+str(p)).read_bytes() if os.name=='nt' else p.read_bytes()
def add(path,area,only_changed=False):
 p=Path(path);rel=p.relative_to(root).as_posix();b=read(p)
 if p.name in ('.env','local.properties') or p.suffix.lower() in ('.pem','.key','.jks','.keystore'):raise RuntimeError('Unsafe source path')
 name=root.name+'/'+rel
 previous=sha(baseline.read(base_names[name])) if name in base_names else None
 if only_changed and previous==sha(b):return
 if b'-----BEGIN OPENSSH PRIVATE KEY-----' in b or b'-----BEGIN RSA PRIVATE KEY-----' in b:raise RuntimeError('Private key found')
 files[rel]={'path':rel,'sha256':sha(b),'bytes':len(b),'area':area,'baseline_sha256':previous,'change':'added' if previous is None else ('modified' if previous!=sha(b) else 'unchanged support')}

bridge=root/'01-bridge/tencent-tangseng-bridge'
for p in bridge.rglob('*'):
 if p.is_file() and (p.suffix in ('.go','.sql') or p.name in ('go.mod','go.sum','Dockerfile.runtime')):
  add(p,'Bridge',True)
# Existing cumulative overlay retains earlier changes and their build support.
with zipfile.ZipFile(release/'business-acceptance-20260930-source.zip') as z:
 for n in z.namelist():
  p=root/n
  if p.is_file() and n.startswith(('01-bridge/','02-authoritative-qch-source/','03-php-deployment/')):add(p,'previous cumulative overlay')
with tarfile.open(release/'php-bridge-provision.tar.gz') as t:
 for m in t.getmembers():
  if m.isfile() and m.name.endswith('.php'):add(root/'03-php-deployment/backend'/m.name,'PHP deployed')
for p in (root/'03-php-deployment/tests').glob('*.php'):add(p,'PHP tests')
android=root/'02-authoritative-qch-source/企才汇源码交付-20260810/Android'
for directory,dirs,names in os.walk(android):
 dirs[:]=[d for d in dirs if d not in ('build','.gradle','.idea','.git','.cxx','libs')]
 for n in names:
  p=Path(directory)/n
  if p.suffix in ('.java','.xml','.gradle','.pro') and n!='local.properties':add(p,'Android',True)
for n in ('deploy-bridge-private-vpc.redacted.ps1','deploy-php-bridge-provision.redacted.ps1','configure-closure-endpoints.py','enable-strict-registration.py','test-php-closure.ps1','test-native-registration-transaction.ps1','capture-group-system-contract.py','capture-registration-contract.py','delete-disposable-registration.php'):
 add(release/n,'deployment and verification')
for p in release.glob('verify-*'):
 if p.suffix in ('.py','.php'):add(p,'verification')
for p in (root/'08-context').iterdir():
 if p.is_file() and (p.suffix=='.md' or p.name.endswith('20261001.json')):add(p,'documentation')
manifest={'release':'business-closure-'+stamp,'baseline':'hw-bridge-handoff-20260929.zip','baseline_sha256':sha((work/'hw-bridge-handoff-20260929.zip').read_bytes()),'scope':'Cumulative overlay against original handoff; no secrets, generated builds, SDK paths or signing files. Existing baseline still required.','files':[files[k] for k in sorted(files)]}
manifest_path=release/('business-closure-'+stamp+'-SOURCE-MANIFEST.json')
manifest_path.write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
for src,dst in [('hw-bridge-private-vpc.tar.gz','hw-bridge-business-closure-'+stamp+'.tar.gz'),('php-bridge-provision.tar.gz','php-business-closure-'+stamp+'.tar.gz')]:shutil.copy2(release/src,release/dst)
archive=release/('business-closure-'+stamp+'-source.zip')
with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED) as z:
 for rel in sorted(files):z.writestr(rel,read(root/rel))
 z.write(manifest_path,manifest_path.relative_to(root).as_posix())
with zipfile.ZipFile(archive) as z:
 assert z.testzip() is None
 for item in manifest['files']:assert sha(z.read(item['path']))==item['sha256']
products=[archive,manifest_path,release/('hw-bridge-business-closure-'+stamp+'.tar.gz'),release/('php-business-closure-'+stamp+'.tar.gz'),release/('qch-business-closure-'+stamp+'-qa.apk')]
(release/('business-closure-'+stamp+'-SHA256SUMS.txt')).write_text(''.join(sha(p.read_bytes())+'  '+p.name+'\n' for p in products),encoding='utf-8')
print('SOURCE_FILES',len(files))
print('SOURCE_ARCHIVE_BYTES',archive.stat().st_size)
print('ANDROID_CHANGED_FILES')
for f in files.values():
 if f['area']=='Android':print(f['path'])
print('RELEASE_MANIFEST_AND_ARCHIVE_VERIFIED')
