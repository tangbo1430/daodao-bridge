<?php
if (PHP_SAPI!=='cli') exit(1);
$_SERVER['argv'][0]='/var/www/app/think';chdir('/var/www/app/public');
require '/var/www/app/thinkphp/base.php';
\think\Container::get('app')->path('/var/www/app/application/')->initialize();request()->setModule('api');
$file=getenv('REGISTRATION_FIXTURE_STATE')?:'/tmp/hw-bridge-closure-registration.json';
if (!in_array($file,['/tmp/hw-bridge-closure-registration.json','/tmp/hw-bridge-product-registration.json'],true)) exit(2);
$s=json_decode(file_get_contents($file),true);
if (empty($s['uid']) || strpos($s['username'],'hwqa')!==0) exit(2);
$user=\think\Db::name('user')->where('uid',(int)$s['uid'])->find();
if (!$user) exit(3);
if ((int)$user['login_status']===3) {echo "DISPOSABLE_PHP_ALREADY_DELETED\n";exit(0);}
if ($user['user_name']!==$s['username']) exit(4);
$r=model('api/User')->cancel_account((int)$s['uid']);
echo 'DISPOSABLE_PHP_DELETE_CODE='.(int)($r['code']??0)."\n";
if ((int)($r['code']??0)!==200) exit(5);
