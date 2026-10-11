<?php
namespace app\imaccess\controller;

use think\Controller;
use think\Db;
use app\imaccess\service\RegistrationService;

class BridgeInternal extends Controller
{
    private function authenticatedPayload()
    {
        if (!request()->isPost()) throw new \RuntimeException('method',405);
        $body=(string)file_get_contents('php://input');
        if (strlen($body)>8192) throw new \RuntimeException('size',400);
        $ts=(string)request()->header('X-HWB-Timestamp','');
        $nonce=(string)request()->header('X-HWB-Nonce','');
        $sign=(string)request()->header('X-HWB-Sign','');
        $secret=(string)config('im_bridge.secret');
        if ($secret==='' || !ctype_digit($ts) || abs(time()-(int)$ts)>300 || !preg_match('/^[a-f0-9]{32}$/D',$nonce)) throw new \RuntimeException('auth',401);
        if (!hash_equals(hash_hmac('sha256',$ts."\n".$nonce."\n".$body,$secret),$sign)) throw new \RuntimeException('auth',401);
            $redis=new \Redis();$redis->connect(getenv('REDIS_HOST')?:'127.0.0.1',(int)(getenv('REDIS_PORT')?:6379),1.0);
            if (getenv('REDIS_PASSWORD')) $redis->auth(getenv('REDIS_PASSWORD'));
            if (!$redis->set('bridge:internal:nonce:'.$nonce,'1',['nx','ex'=>600])) throw new \RuntimeException('replay',409);
            $p=json_decode($body,true);
            if (!is_array($p)) throw new \RuntimeException('payload',400);
            return $p;
    }

    private function rejected(\Throwable $e)
    {
        $code=in_array($e->getCode(),[400,401,405,409],true)?$e->getCode():503;
        return json(['code'=>$code,'msg'=>'内部状态请求未完成'],$code);
    }

    public function registrationLookup()
    {
        try {
            $p=$this->authenticatedPayload();
            if (!is_string($p['username']??null) || !preg_match('/^[a-f0-9]{64}$/D',(string)($p['reserve_id']??''))) throw new \RuntimeException('payload',400);
            $uid=(new RegistrationService())->lookup($p['username'],$p['reserve_id']);
            return json(['code'=>0,'tim_uid'=>$uid]);
        } catch (\Throwable $e) {
            return $this->rejected($e);
        }
    }

    public function accountStatus()
    {
        try {
            $p=$this->authenticatedPayload();
            if (!preg_match('/^[A-Za-z0-9_-]{1,32}$/D',(string)($p['tim_uid']??'')) || !in_array($p['state']??'', ['confirmed','banned','tenant_disabled','deleted'],true) || !is_int($p['version']??null) || $p['version']<=0) throw new \RuntimeException('payload',400);
            // Projection only: never mutate user.login_status or Tencent bans.
            $prefix=(string)config('database.prefix');
            if (!preg_match('/^[A-Za-z0-9_]*$/D',$prefix)) throw new \RuntimeException('configuration');
            Db::execute('INSERT INTO `'.$prefix.'im_bridge_account_status` (tim_uid,state,reason,version,updated_at) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE state=IF(VALUES(version)>version,VALUES(state),state),reason=IF(VALUES(version)>version,VALUES(reason),reason),updated_at=IF(VALUES(version)>version,VALUES(updated_at),updated_at),version=GREATEST(version,VALUES(version))',[$p['tim_uid'],$p['state'],$p['state'],$p['version'],time()]);
            return json(['code'=>0]);
        } catch (\Throwable $e) { return $this->rejected($e); }
    }
}
