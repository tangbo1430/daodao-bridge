<?php
// CLI-only integration fixture; never deploy under public or register a route.
// Empty product referral; native channel credential stays on the Bridge host.
if (PHP_SAPI !== 'cli') exit(1);
$_SERVER['REMOTE_ADDR']='127.0.0.1';
$_SERVER['argv'][0]='/var/www/app/think';
chdir('/var/www/app/public');
require '/var/www/app/thinkphp/base.php';
\think\Loader::addAutoLoadDir('/var/www/app/extend/');
\think\Container::get('app')->path('/var/www/app/application/')->initialize();
request()->setModule('api');
config('im_bridge.strict_registration',true);
$file='/tmp/hw-bridge-product-registration.json';
if (is_file($file)) $state=json_decode(file_get_contents($file),true);
else {
    $state=['username'=>'hwqa'.date('mdHi').bin2hex(random_bytes(3)),'request_key'=>'closure-'.bin2hex(random_bytes(16))];
    file_put_contents($file,json_encode($state));chmod($file,0600);
}
try {
    $user=\think\Db::name('user')->where('user_name',$state['username'])->find();
    if (!$user) {
        $old=\think\Db::name('im_bridge_registration')->where('username',$state['username'])->where('status','aborted')->find();
        if ($old) { $state['request_key']='closure-'.bin2hex(random_bytes(16));file_put_contents($file,json_encode($state)); }
        $password=bin2hex(random_bytes(6));
        $result=model('api/User')->user_reg_account($state['username'],$password,'',$password,$state['request_key']);
        unset($password);
        echo 'PHP_REGISTRATION_RESULT code='.(int)($result['code']??0).' bridge_status='.(string)($result['data']['bridge_status']??'unknown')."\n";
        if ((int)($result['code']??0)!==200) {
            $safe=['注册失败','注册失败11','注册失败22','注册失败33','登录失败','注册服务暂不可用，请稍后重试','注册来源地址无效','注册处理未完成，请稍后重试','注册通道暂不可用','邀请码无效或已失效','用户名已被占用，请换一个','该注册请求已处理，请登录或重新发起'];
            echo 'SAFE_REGISTRATION_REASON='.(in_array($result['msg']??'',$safe,true)?$result['msg']:'unlisted')."\n";
            // RegistrationException text is curated; avoid arbitrary raw errors.
            echo "PHP_REGISTRATION_NOT_COMPLETED\n";exit(3);
        }
        $user=\think\Db::name('user')->where('user_name',$state['username'])->find();
    }
    if (!$user) throw new \RuntimeException('missing local user');
    $state['uid']=(int)$user['uid'];
    $row=\think\Db::name('im_bridge_registration')->where('uid',$state['uid'])->find();
    if (!$row) throw new \RuntimeException('missing durable registration');
    $service=new \app\imaccess\service\RegistrationService();
    $ready=$service->dispatch($row['request_id']);
    $state['reserve_id']=$row['reserve_id'];
    file_put_contents($file,json_encode($state));chmod($file,0600);
    echo 'DEDICATED_REGISTRATION_UID='.$state['uid']."\n";
    echo 'PHP_NATIVE_COMMIT_READY='.($ready?'true':'false')."\n";
    config('im_bridge.strict_registration',false);
    if (!$service->accountReady($state['uid'])) throw new \RuntimeException('rollback guard');
    echo "PHP_NATIVE_REGISTRATION_TRANSACTION_OK\n";
} catch (\Throwable $e) {
    echo 'REGISTRATION_FIXTURE_FAILED class='.get_class($e).' file='.basename($e->getFile()).' line='.$e->getLine()."\n";
    if (preg_match('/^Call to undefined function [A-Za-z0-9_\\\\]+\(\)/',$e->getMessage(),$safe)) echo $safe[0]."\n";
    exit(4);
}
