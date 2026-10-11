<?php
// Standalone fixtures: execute the actual legacy GroupV model, without network.
namespace think {
    class Model {}
    class Db {
        private $table;
        public static function name($name) { $q=new self(); $q->table=$name; return $q; }
        public function where(...$args) { return $this; }
        public function whereRaw(...$args) { return $this; }
        public function order(...$args) { return $this; }
        public function field(...$args) { return $this; }
        public function value(...$args) { return null; }
        public function find() { return null; }
        public function count() { return 0; }
        public function insert($row) { $GLOBALS['local_groups'][]=$row; return 1; }
        public static function startTrans() {}
        public static function commit() {}
        public static function rollback() {}
    }
}
namespace app\imaccess\service {
    class BridgeService {
        public function createGroupForUser($uid,$name,$members,$request) {
            $GLOBALS['bridge_calls'][]=[$uid,$name,$members,$request];
            return $GLOBALS['bridge_result'];
        }
    }
}
namespace {
    function config($key) { return 3; }
    function model($name) { return new class {
        public function create_family(...$args) { $GLOBALS['tencent_calls']++; return ['code'=>200]; }
        public function send_group_created_message(...$args) { return ['code'=>200]; }
    }; }
    require (getenv('GROUP_MODEL_FILE') ?: '/var/www/app/application/api/model/GroupV.php');
    function check($value,$message) { if (!$value) throw new \RuntimeException($message); }
    $GLOBALS['local_groups']=[]; $GLOBALS['tencent_calls']=0; $GLOBALS['bridge_calls']=[];
    $GLOBALS['bridge_result']=['http_status'=>200,'body'=>['code'=>200,'data'=>['group_id'=>'dd_fixture']]];
    $m=new \app\api\model\GroupV(); $peer=str_repeat('a',32);
    $result=$m->create_group(700001,'Mixed',[$peer,$peer,'700001'],'stable-request');
    check($result['code']===200 && $result['data']['group_id']==='dd_fixture' && $result['data']['group_type']==='Public','legacy APK did not receive native mirror group');
    check($GLOBALS['bridge_calls'][0]===[700001,'Mixed',[$peer],'stable-request'],'actor/member/request changed');
    check(!$GLOBALS['local_groups'] && $GLOBALS['tencent_calls']===0,'mixed group created Tencent-only side effects');
    $GLOBALS['bridge_result']=['http_status'=>200,'body'=>['code'=>409,'msg'=>'请至少选择一位好友建群']];
    $result=$m->create_group(700001,'Rejected',[$peer],'stable-request');
    check($result['code']===409 && $GLOBALS['tencent_calls']===0,'native rejection fell back to Tencent success');
    $GLOBALS['bridge_result']=['http_status'=>200,'body'=>['code'=>200,'data'=>['group_id'=>'']]];
    check($m->create_group(700001,'Missing ID',[$peer],'stable-request')['code']===502,'missing mirror reported success');
    $result=$m->create_group(700001,'Normal',['700002'],'normal-request');
    check($result['code']===200 && strpos($result['data']['group_id'],'dd_')!==0 && $GLOBALS['tencent_calls']===1,'ordinary Tencent group changed');
    echo "GROUP_CREATION_ROUTE_CONTRACT_OK\n";
}
