"""Check the actual R8 output needed by the reflected TUIKit business client."""
from pathlib import Path
import re,sys,json,zipfile,struct
root=Path(__file__).resolve().parent.parent
mapping=root/'02-authoritative-qch-source/企才汇源码交付-20260810/Android/qch/app/build/outputs/mapping/qaRelease/mapping.txt'
text=mapping.read_text('utf-8')
block=re.search(r'^com\.qchm\.xyz\.DemoApplication -> ([^:]+):\n(.*?)(?=^\S|\Z)',text,re.M|re.S)
assert block,'Application absent from mapping'
field=re.search(r'^\s+java\.lang\.String NewAddress -> (\S+)',block[2],re.M)
result={'application_name_preserved':block[1]=='com.qchm.xyz.DemoApplication','mapped_field_name':field[1] if field else 'identity mapping omitted'}
print(json.dumps(result))
# R8 may omit unchanged field names. The actual DEX definition below is required.
assert result['application_name_preserved'] and (field is None or field[1]=='NewAddress'),'R8 breaks runtime endpoint reflection'

# Verify the packaged DEX field definition, not only a name in mapping.txt.
apk=root/'02-authoritative-qch-source/企才汇源码交付-20260810/Android/qch/app/build/outputs/apk/qaRelease/企才汇-qaRelease.apk'
def uleb(data,offset):
 value=0;shift=0
 while True:
  v=data[offset];offset+=1;value|=(v&127)<<shift
  if not v&128:return value,offset
  shift+=7
  assert shift<35,'Invalid DEX value'
found=False
with zipfile.ZipFile(apk) as archive:
 for name in archive.namelist():
  if not re.fullmatch(r'classes\d*\.dex',name):continue
  b=archive.read(name)
  string_count,string_off,type_count,type_off,_,_,field_count,field_off,_,_,class_count,class_off=struct.unpack_from('<12I',b,56)
  strings=[]
  for i in range(string_count):
   off=struct.unpack_from('<I',b,string_off+4*i)[0];_,off=uleb(b,off)
   strings.append(b[off:b.index(b'\0',off)].decode('utf-8','replace'))
  types=[strings[struct.unpack_from('<I',b,type_off+4*i)[0]] for i in range(type_count)]
  for i in range(class_count):
   row=struct.unpack_from('<8I',b,class_off+32*i)
   if types[row[0]]!='Lcom/qchm/xyz/DemoApplication;':continue
   offset=row[6];static_count,offset=uleb(b,offset)
   for _ in range(3):_,offset=uleb(b,offset)
   field_index=0
   for _ in range(static_count):
    delta,offset=uleb(b,offset);flags,offset=uleb(b,offset);field_index+=delta
    owner,kind,label=struct.unpack_from('<HHI',b,field_off+8*field_index)
    if strings[label]=='NewAddress':
     assert types[owner]=='Lcom/qchm/xyz/DemoApplication;' and types[kind]=='Ljava/lang/String;' and flags&9==9
     found=True
assert found,'APK lacks the public static field required by getField(NewAddress)'
print('APK_PUBLIC_STATIC_ENDPOINT_FIELD_OK')
print('ANDROID_RELEASE_ENDPOINT_REFLECTION_OK')
