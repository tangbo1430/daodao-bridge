<?php
// Synthetic contract: in-memory database and Bridge transport, no live users.
namespace think {
class Db {
    public static $rows=[]; private $table;private $where=[];
    public static function name($t){$q=new self();$q->table=$t;return $q;}
    public function where($k,$op,$v=null){$this->where[]=[$k,$v===null?'=':$op,$v===null?$op:$v];return $this;}
    private function matches($r){foreach($this->where as $w){$v=$r[$w[0]]??null;if($w[1]==='='&&$v!=$w[2])return false;if($w[1]==='in'&&!in_array($v,$w[2],true))return false;if($w[1]==='<='&&$v>$w[2])return false;}return true;}
    public function insert($r){$r+=['uid'=>0,'reserve_id'=>'','attempts'=>0,'next_retry_at'=>0];self::$rows[$this->table][]=$r;return 1;}
    public function select(){return array_values(array_filter(self::$rows[$this->table]??[],function($r){return $this->matches($r);}));}
    public function find(){$r=$this->select();return $r[0]??null;}
    public function update($new){$n=0;foreach(self::$rows[$this->table] as &$r){if($this->matches($r)){$r=array_merge($r,$new);$n++;}}return $n;}
    public function limit($n){return $this;}
}}
namespace app\imaccess\service {
function config($key){return ['im_bridge.strict_registration'=>true,'im_bridge.secret'=>'fixture-secret','im_access.sdk_app_id'=>'fixture-app'][$key]??null;}
function request(){return new class{function server($key,$fallback){return '192.0.2.1';}};}
class AccountMappingService {function resolve($uid,$app){return ['im_user_id'=>'fixture-tim'];}}
class BridgeService {
 public static $calls=[];public static $failCommit=true;
 function registrationCall($action,$body){self::$calls[]=[$action,$body];if($action==='prepare')return ['code'=>0,'data'=>['reserve_id'=>hash('sha256',"registration\0".$body['request_id']."\0")]];if($action==='commit'&&self::$failCommit)throw new \RuntimeException('fixture network failure');return ['code'=>0];}
}}
namespace {
require (getenv('REGISTRATION_SERVICE_FILE')?:'/var/www/app/application/imaccess/service/RegistrationService.php');
function check($ok,$msg){if(!$ok)throw new \RuntimeException($msg);}
$s=new \app\imaccess\service\RegistrationService();
$id=$s->prepare('fixture_user','Fixture','','fixture-request');
$same=$s->prepare('fixture_user','Fixture','','fixture-request');
check($same===$id,'retry changed request identity');
$calls=\app\imaccess\service\BridgeService::$calls;
check($calls[0][1]['client_ip']==='192.0.2.1'&&!isset($calls[0][1]['tim_uid']),'prepare trusted client actor');
check(!isset($calls[0][1]['invite_code']),'product referral leaked to native registration');
$s->prepare('fixture_other','Other','product-referral','fixture-other-request');
$calls=\app\imaccess\service\BridgeService::$calls;
check(!isset($calls[count($calls)-1][1]['invite_code']),'optional product referral used as native credential');
\think\Db::$rows['user']=[['uid'=>700001,'user_name'=>'fixture_user']];
$s->queueCommit($id,700001);
check($s->dispatch($id)===false,'failed commit reported ready');
$r=\think\Db::$rows['im_bridge_registration'][0];
check($r['status']==='commit_pending'&&$r['attempts']===1,'commit retry not durable');
check($s->lookup('fixture_user',$r['reserve_id'])==='fixture-tim','reconciliation lookup failed');
check($s->lookup('other_user',$r['reserve_id'])==='','lookup accepted other identity');
\app\imaccess\service\BridgeService::$failCommit=false;
check($s->dispatch($id)===true,'commit retry failed');
$s->abort($id);
check(\think\Db::$rows['im_bridge_registration'][0]['status']==='committed','late abort reverted committed registration');
echo "REGISTRATION_TRANSACTION_CONTRACT_OK\n";
}
