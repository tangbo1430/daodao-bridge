"""Re-publish an authorized user's CURRENT native avatar once, without changing bytes."""
from pathlib import Path
exec(Path(__file__).with_name('verify-group-settings.py').read_text().split('gid=None')[0])
url=cfg['TANGSENG_BASE_URL'].rstrip('/')+'/v1/users/'+reg['tsdd_uid']+'/avatar'
with urllib.request.urlopen(url,timeout=30) as r:
 if r.status!=200 or r.headers.get_content_type()!='image/png':raise RuntimeError('Existing PNG avatar required')
 image=r.read(20*1024*1024+1)
 if len(image)>20*1024*1024:raise RuntimeError('Avatar too large')
before=hashlib.sha256(image).hexdigest()
boundary='hwbridge-avatar-refresh-'+uuid.uuid4().hex
body=('--'+boundary+'\r\nContent-Disposition: form-data; name="file"; filename="avatar.png"\r\nContent-Type: image/png\r\n\r\n').encode()+image+('\r\n--'+boundary+'--\r\n').encode()
req=urllib.request.Request(url,data=body,headers={'token':token,'Content-Type':'multipart/form-data; boundary='+boundary})
try:
 with urllib.request.urlopen(req,timeout=45) as r:result=json.load(r)
except Exception as e:raise RuntimeError('Avatar refresh failed: '+type(e).__name__) from None
assert result.get('status',200)==200 and result.get('code',200) in [0,200]
with urllib.request.urlopen(url,timeout=30) as r:after=hashlib.sha256(r.read(20*1024*1024+1)).hexdigest()
assert after==before,'Avatar content unexpectedly changed'
print('OWN_AVATAR_REFRESH_SENT_BYTES_UNCHANGED',before,flush=True)
