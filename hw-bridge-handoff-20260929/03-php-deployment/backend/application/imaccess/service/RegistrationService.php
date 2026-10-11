<?php
namespace app\imaccess\service;

use think\Db;

class RegistrationException extends \RuntimeException {}

/** Durable registration transaction. No native passwords/tokens enter PHP. */
class RegistrationService
{
    public static function enabled() { return (bool) config('im_bridge.strict_registration'); }

    public static function install()
    {
        $prefix = (string) config('database.prefix');
        if (!preg_match('/^[A-Za-z0-9_]*$/', $prefix)) throw new \RuntimeException('Invalid database prefix');
        Db::execute('CREATE TABLE IF NOT EXISTS `' . $prefix . 'im_bridge_registration` (
            request_id VARCHAR(64) NOT NULL PRIMARY KEY, username VARCHAR(64) NOT NULL,
            reserve_id VARCHAR(64) NOT NULL DEFAULT \'\', uid BIGINT UNSIGNED NOT NULL DEFAULT 0,
            status VARCHAR(24) NOT NULL, attempts INT NOT NULL DEFAULT 0,
            next_retry_at BIGINT NOT NULL DEFAULT 0, created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
            KEY idx_registration_due(status,next_retry_at), KEY idx_registration_user(uid), KEY idx_registration_name(username)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4');
        Db::execute('CREATE TABLE IF NOT EXISTS `' . $prefix . 'im_bridge_account_status` (
            tim_uid VARCHAR(32) NOT NULL PRIMARY KEY, state VARCHAR(32) NOT NULL,
            reason VARCHAR(32) NOT NULL, version BIGINT NOT NULL, updated_at BIGINT NOT NULL
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4');
    }

    public function prepare($username, $nick, $invite, $requestKey)
    {
        if (!self::enabled()) return null;
        if ($requestKey === '' || (string)config('im_bridge.secret') === '') throw new RegistrationException('注册请求未就绪，请重新获取验证码');
        if (!preg_match('/^(?:[A-Za-z][A-Za-z0-9_]{7,21}|[0-9]{8,22})$/D', $username)) throw new RegistrationException('用户名须为8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线');
        $id = hash_hmac('sha256', 'registration:' . $requestKey, (string) config('im_bridge.secret'));
        $row = Db::name('im_bridge_registration')->where('request_id', $id)->find();
        if (!$row) {
            Db::name('im_bridge_registration')->insert(['request_id'=>$id,'username'=>$username,'status'=>'preparing','created_at'=>time(),'updated_at'=>time()]);
        } elseif ($row['username'] !== $username || !in_array($row['status'], ['preparing','prepared'], true)) {
            throw new RegistrationException('该注册请求已处理，请登录或重新发起');
        }
        // Trusted transport address only. Never forward an APK-provided IP.
        $ip = (string) request()->server('REMOTE_ADDR', '');
        // The optional product referral is validated locally; never treat it
        // as the native channel credential. Bridge reads that from its .env.
        $r = (new BridgeService())->registrationCall('prepare', ['request_id'=>$id,'username'=>$username,'nick'=>$nick,'client_ip'=>$ip]);
        if ((int) ($r['code'] ?? -1) !== 0 || empty($r['data']['reserve_id'])) throw new RegistrationException((string) ($r['msg'] ?? '注册结果待确认，请重试'));
        Db::name('im_bridge_registration')->where('request_id',$id)->update(['reserve_id'=>$r['data']['reserve_id'],'status'=>'prepared','updated_at'=>time()]);
        return $id;
    }

    /** Must run inside the SAME local user transaction. */
    public function queueCommit($id, $uid)
    {
        if ($id === null) return;
        $changed = Db::name('im_bridge_registration')->where('request_id',$id)->where('status','prepared')->update(['uid'=>(int)$uid,'status'=>'commit_pending','updated_at'=>time(),'next_retry_at'=>0]);
        if (!$changed) throw new \RuntimeException('注册确认状态不匹配');
    }

    public function abort($id)
    {
        if ($id === null) return;
        Db::name('im_bridge_registration')->where('request_id',$id)->where('status','prepared')->update(['status'=>'abort_pending','updated_at'=>time(),'next_retry_at'=>0]);
        $this->dispatch($id);
    }

    public function dispatch($id)
    {
        $r = Db::name('im_bridge_registration')->where('request_id',$id)->find();
        if (!$r) return false;
        if ($r['status']==='committed') return true;
        if (!in_array($r['status'], ['commit_pending','abort_pending'], true)) return false;
        try {
            $action = $r['status']==='commit_pending' ? 'commit' : 'abort';
            $payload=['reserve_id'=>$r['reserve_id']];
            if ($action==='commit') {
                $user=Db::name('user')->where('uid',(int)$r['uid'])->find();
                if (!$user || $user['user_name']!==$r['username']) throw new \RuntimeException('Local registration identity mismatch');
                $map=(new AccountMappingService())->resolve((int)$r['uid'],(string)config('im_access.sdk_app_id'));
                $payload['tim_uid']=(string)$map['im_user_id'];
            } else { $payload['reason']='local_registration_failed'; }
            $result=(new BridgeService())->registrationCall($action,$payload);
            if ((int)($result['code'] ?? -1)!==0) throw new \RuntimeException('Bridge confirmation pending');
            Db::name('im_bridge_registration')->where('request_id',$id)->where('status',$r['status'])->update(['status'=>$action==='commit'?'committed':'aborted','updated_at'=>time()]);
            return $action==='commit';
        } catch (\Throwable $e) {
            // Safe metadata only; the row remains retryable without a retry cap.
            Db::name('im_bridge_registration')->where('request_id',$id)->where('status',$r['status'])->update(['attempts'=>(int)$r['attempts']+1,'next_retry_at'=>time()+60,'updated_at'=>time()]);
            return false;
        }
    }

    public function retryPending($limit=20)
    {
        // The switch controls new registrations only. Previously committed local
        // transactions must still finish after an entry-point rollback.
        $rows=Db::name('im_bridge_registration')->where('status','in',['commit_pending','abort_pending'])->where('next_retry_at','<=',time())->limit($limit)->select();
        foreach ($rows as $row) $this->dispatch($row['request_id']);
    }

    public function accountReady($uid)
    {
        $r=Db::name('im_bridge_registration')->where('uid',(int)$uid)->where('status','in',['commit_pending','committed'])->find();
        return !$r || $r['status']==='committed' || $this->dispatch($r['request_id']);
    }

    public function lookup($username,$reserve)
    {
        $rows=Db::name('im_bridge_registration')->where('username',$username)->select();
        foreach ($rows as $r) {
            $expected=hash('sha256',"registration\0".$r['request_id']."\0");
            if (!hash_equals($expected,$reserve) || !$r['uid'] || !in_array($r['status'],['commit_pending','committed'],true)) continue;
            $u=Db::name('user')->where('uid',(int)$r['uid'])->find();
            if (!$u || $u['user_name']!==$username) continue;
            $mapping=(new AccountMappingService())->resolve((int)$r['uid'],(string)config('im_access.sdk_app_id'));
            return (string)$mapping['im_user_id'];
        }
        return '';
    }
}
