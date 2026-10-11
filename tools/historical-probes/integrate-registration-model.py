from pathlib import Path
root=Path('C:/Users/Administrator/Desktop/tx2daodao/hw-bridge-handoff-20260929/03-php-deployment/backend')
p=root/'application/api/model/User.php'
s=p.read_text(encoding='utf-8-sig')
for name,next_name in [('user_reg','user_reg_account'),('user_reg_account','validate_user_reg_account')]:
 start=s.index('    public function '+name+'(');end=s.index('    public function '+next_name+'(',start+1)
 part=s[start:end]
 signature=part.index(')\n')
 part=part[:signature]+", $bridgeRequestKey = '')"+part[signature+1:]
 old='        Db::startTrans();\n        try {'
 new='''        $registration = new \\app\\imaccess\\service\\RegistrationService();
        $reservation = null;
        $localCommitted = false;
        try {
            $reservation = $registration->prepare($user_name, $nick_name, $reg_code, $bridgeRequestKey);
        } catch (\\Throwable $e) {
            $msg = $e instanceof \\app\\imaccess\\service\\RegistrationException ? $e->getMessage() : '注册服务暂不可用，请稍后重试';
            return ['code'=>201,'msg'=>$msg,'data'=>null];
        }
        Db::startTrans();
        try {'''
 assert part.count(old)==1
 part=part.replace(old,new)
 old='''            Db::commit();
            try {
                (new \\app\\imaccess\\service\\BridgeService())->provisionUser($uid);
            } catch (\\Throwable $e) {
                Log::error('[IM bridge account] provision failed uid=' . (string) $uid . ' error=' . $e->getMessage());
            }
            return ['code' => 200, 'msg' => '注册成功', 'data' => ['nick_name' => $nick_name]];'''
 new='''            $registration->queueCommit($reservation, $uid);
            Db::commit();
            $localCommitted = true;
            $bridgeReady = false;
            if ($reservation !== null) {
                $bridgeReady = $registration->dispatch($reservation);
            } else {
                try { (new \\app\\imaccess\\service\\BridgeService())->provisionUser($uid); $bridgeReady = true; }
                catch (\\Throwable $e) { Log::error('[IM bridge account] legacy provision pending uid=' . (string)$uid); }
            }
            return ['code'=>200,'msg'=>$bridgeReady?'注册成功':'本地注册成功，叨叨桥接确认中','data'=>['nick_name'=>$nick_name,'bridge_status'=>$bridgeReady?'confirmed':'pending']];'''
 assert part.count(old)==1
 part=part.replace(old,new)
 # Add cleanup to the existing transaction try/catch, covering every early return.
 suffix='        }\n    }'
 at=part.rfind(suffix);assert at!=-1
 part=part[:at]+'''        } finally {
            if (!$localCommitted && $reservation !== null) {
                try { $registration->abort($reservation); }
                catch (\\Throwable $e) { Log::error('[IM bridge registration] abort remains pending'); }
            }
        }
    }'''+part[at+len(suffix):]
 s=s[:start]+part+s[end:]
begin=s.index('        if (empty(validate_user_name($user_name)))',s.index('public function validate_user_reg_account'))
end=s.index("        if (Db::name('user')",begin)
old=s[begin:end]
s=s[:begin]+'''        if (\\app\\imaccess\\service\\RegistrationService::enabled()) {
            if (!preg_match('/^(?:[A-Za-z][A-Za-z0-9_]{7,21}|[0-9]{8,22})$/D', $user_name)) return ['code'=>201,'msg'=>'用户名须为8至22位字母、数字或下划线，纯数字账号需邀请码允许','data'=>null];
            if ($reg_code === '') return ['code'=>201,'msg'=>'请输入叨叨邀请码','data'=>null];
        } else {
'''+old+'        }\n'+s[end:]
# Strict registrations use only native invitation validation.
at=s.index('public function validate_user_reg_account')
a,b=s[:at],s[at:]
b=b.replace("if ($reg_code !== '') {", "if ($reg_code !== '' && !\\app\\imaccess\\service\\RegistrationService::enabled()) {",1)
p.write_text(a+b,encoding='utf-8')
p=root/'application/api/controller/Login.php';s=p.read_text(encoding='utf-8-sig')
s=s.replace("model('user')->user_reg($user_name, $password, $reg_code)","model('user')->user_reg($user_name, $password, $reg_code, $result_key)")
s=s.replace("model('user')->user_reg_account($user_name, $password, $reg_code, $repassword)","model('user')->user_reg_account($user_name, $password, $reg_code, $repassword, $result_key)")
p.write_text(s,encoding='utf-8')
print('Registration model/controller integrated')
