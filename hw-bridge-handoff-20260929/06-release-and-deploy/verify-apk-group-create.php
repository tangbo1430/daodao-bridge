<?php
// CLI only, real APK form endpoint. Tokens remain on the PHP host.
if (PHP_SAPI !== 'cli' || count($argv) < 4 || !ctype_digit($argv[1]) || !preg_match('/^[0-9a-f]{32}$/i',$argv[2])) exit(1);
$uid=(int)$argv[1]; $peer=$argv[2]; $mode=$argv[3];
$_SERVER['argv'][0]='/var/www/app/think'; chdir('/var/www/app/public');
require '/var/www/app/thinkphp/base.php';
\think\Container::get('app')->path('/var/www/app/application/')->initialize();
$user=\think\Db::name('user')->where('uid',$uid)->find();
if (!$user || (int)$user['login_status']!==1 || empty($user['login_token'])) throw new \RuntimeException('Test account unavailable');
$stateFile='/tmp/apk-group-acceptance-'.$uid.'.json';
function api($path,$payload,$json=false) {
    $ch=curl_init('https://admin.awhaha.com'.$path);
    curl_setopt_array($ch,[CURLOPT_POST=>true,CURLOPT_POSTFIELDS=>$json?json_encode($payload):http_build_query($payload),
        CURLOPT_HTTPHEADER=>['Content-Type: '.($json?'application/json':'application/x-www-form-urlencoded')],
        CURLOPT_RETURNTRANSFER=>true,CURLOPT_CONNECTTIMEOUT=>8,CURLOPT_TIMEOUT=>90]);
    $raw=curl_exec($ch);$http=(int)curl_getinfo($ch,CURLINFO_HTTP_CODE);curl_close($ch);
    $result=json_decode((string)$raw,true);
    echo 'API '. $path.' http='.$http.' code='.($result['code']??0)."\n";
    if ($http!==200 || (int)($result['code']??0)!==200) throw new \RuntimeException('Test operation rejected');
    return $result['data']??[];
}
if ($mode==='create') {
    if (file_exists($stateFile)) $state=json_decode(file_get_contents($stateFile),true);
    else {
        $state=['uid'=>$uid,'peer'=>$peer,'request_id'=>bin2hex(random_bytes(16)),'created_at'=>time()];
        file_put_contents($stateFile,json_encode($state));chmod($stateFile,0600);
    }
    if (!empty($state['cleaned']) || $state['peer']!==$peer) throw new \RuntimeException('Use a new acceptance state after reviewing prior run');
    $payload=['login_token'=>$user['login_token'],'group_name'=>'建群接口验收-'.substr($state['request_id'],0,6),
        'member_ids'=>json_encode([$peer]),'request_id'=>$state['request_id'], 'uid'=>999999999];
    // Deliberately send a bogus uid: Common must derive the real owner from token.
    $first=api('/api/group_v/create_group',$payload);
    if (strpos($first['group_id']??'','dd_')!==0) throw new \RuntimeException('Tencent-only group returned');
    $state['group_id']=$first['group_id'];file_put_contents($stateFile,json_encode($state));
    $second=api('/api/group_v/create_group',$payload);
    if ($second['group_id']!==$first['group_id']) throw new \RuntimeException('Duplicate group created on retry');
    echo 'APK_FORM_NATIVE_GROUP_AND_RETRY_OK '.$first['group_id']."\n";
} elseif ($mode==='cleanup') {
    $state=json_decode(file_get_contents($stateFile),true);
    if ($state['uid']!==$uid || $state['peer']!==$peer || strpos($state['group_id']??'','dd_')!==0) throw new \RuntimeException('Invalid cleanup state');
    api('/api/bridge/group_disband',['login_token'=>$user['login_token'],'group_id'=>$state['group_id']],true);
    $state['cleaned']=true;file_put_contents($stateFile,json_encode($state));
    echo 'TEMP_APK_GROUP_DISBANDED '.$state['group_id']."\n";
} else throw new \RuntimeException('Unknown mode');
