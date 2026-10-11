"""Run on Bridge CVM. Verify real MySQL MIME capacity without retaining test rows."""
import subprocess
import uuid

mimes = [
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'application/vnd.openxmlformats-officedocument.presentationml.presentation',
]
rows = []
for mime in mimes:
    key = uuid.uuid4().hex + uuid.uuid4().hex
    rows.append((key, mime))
query = "SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='media_cache' AND column_name='media_type'; START TRANSACTION;\n"
for key, mime in rows:
    query += "INSERT INTO media_cache(sha256,media_type,created_at,updated_at) VALUES('%s','%s',UNIX_TIMESTAMP(),UNIX_TIMESTAMP());\n" % (key, mime)
    query += "SELECT media_type FROM media_cache WHERE sha256='%s';\n" % key
query += 'ROLLBACK;\n'
p = subprocess.run(['docker','compose','exec','-T','mysql','sh','-c',
    'export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" -N -B --raw'],
    cwd='/opt/hw-bridge', input=query, text=True, capture_output=True)
assert p.returncode == 0, 'MySQL Office MIME round trip failed'
output = p.stdout.splitlines()
assert int(output[0]) >= 255 and output[1:] == mimes, 'MIME truncated or incompatible schema'
print('OFFICE_MIME_MYSQL_ROUNDTRIP_OK width=' + output[0] + ' types=3 rolled_back=true')
