<?php
// CLI only. Existing login token stays on PHP host; stdout contains no token.
if (PHP_SAPI !== 'cli' || count($argv)<4 || !ctype_digit($argv[1])) exit(1);
$uid=(int)$argv[1];$keywords=array_slice($argv,2);
$_SERVER['argv'][0]='/var/www/app/think';chdir('/var/www/app/public');
require '/var/www/app/thinkphp/base.php';
\think\Container::get('app')->path('/var/www/app/application/')->initialize();
$user=\think\Db::name('user')->where('uid',$uid)->find();
if (!$user || (int)$user['login_status']!==1 || empty($user['login_token'])) {echo "SEARCH_ACCOUNT_UNAVAILABLE\n";exit(2);}
foreach($keywords as $index=>$keyword){
    $body=json_encode(['login_token'=>$user['login_token'],'keyword'=>$keyword]);
    $ch=curl_init('https://admin.awhaha.com/api/bridge/search_user');
    curl_setopt_array($ch,[CURLOPT_POST=>true,CURLOPT_POSTFIELDS=>$body,CURLOPT_HTTPHEADER=>['Content-Type: application/json'],CURLOPT_RETURNTRANSFER=>true,CURLOPT_CONNECTTIMEOUT=>8,CURLOPT_TIMEOUT=>25]);
    $raw=curl_exec($ch);$http=(int)curl_getinfo($ch,CURLINFO_HTTP_CODE);curl_close($ch);
    $r=json_decode((string)$raw,true);$rows=$r['users']??($r['data']['users']??[]);
    echo 'PHP_PUBLIC_SEARCH case='.($index+1).' http='.$http.' matches='.count($rows)."\n";
    if ($http!==200 || count($rows)!==1 || empty($rows[0]['user_id'])) exit(3);
    if ($index===0) $matched=$rows[0]['user_id'];
    elseif ($rows[0]['user_id']!==$matched) {echo "SEARCH_IDENTITIES_DIFFER\n";exit(4);}
}
echo "PHP_SEARCH_NAME_AND_CODE_SAME_USER_OK\n";
