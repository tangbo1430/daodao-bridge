package main
import("encoding/json";"bytes";"image";_ "image/jpeg";"image/png";"os";"fmt";"crypto/sha256")
func main(){b,e:=os.ReadFile("toolchain/avatar-pixel-audit.json");if e!=nil{panic(e)};var data map[string][]byte;if e=json.Unmarshal(b,&data);e!=nil{panic(e)};src,_,e:=image.Decode(bytes.NewReader(data["source"]));if e!=nil{panic(e)};var out bytes.Buffer;if e=png.Encode(&out,src);e!=nil{panic(e)};fmt.Printf("BRIDGE_AVATAR_REENCODE_MATCH=%v\n",sha256.Sum256(out.Bytes())==sha256.Sum256(data["native"]))}
