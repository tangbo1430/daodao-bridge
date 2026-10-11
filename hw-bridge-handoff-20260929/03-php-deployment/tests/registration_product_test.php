<?php
// Synthetic contract of the actual user model's optional product referral.
namespace think {
class Model {}
class Db {
    public static $invite=null;
    public static function name($table){return new self();}
    public function where(...$args){return $this;}
    public function count(){return 0;}
    public function find(){return self::$invite;}
}}
namespace app\imaccess\service {
class RegistrationService {public static function enabled(){return true;}}
}
namespace app\api\model {
function config($key){return 2;}
}
namespace {
require(getenv('REGISTRATION_USER_FILE')?:'/var/www/app/application/api/model/User.php');
$s=new \app\api\model\User();
function check($value,$message){if(!$value)throw new \RuntimeException($message);}
$r=$s->validate_user_reg_account('fixture_user','fixture-pass','','fixture-pass');
check($r['code']===200,'empty optional referral rejected');
$r=$s->validate_user_reg_account('fixture_user','fixture-pass','unknown-referral','fixture-pass');
check($r['code']===201,'invalid product referral bypassed local validation');
\think\Db::$invite=['id'=>1,'user_id'=>'','invite_num'=>2];
$r=$s->validate_user_reg_account('fixture_user','fixture-pass','valid-referral','fixture-pass');
check($r['code']===200&&$r['data']['invitecode']['id']===1,'valid local referral lost');
\think\Db::$invite=['id'=>1,'user_id'=>'1,2','invite_num'=>2];
$r=$s->validate_user_reg_account('fixture_user','fixture-pass','used-referral','fixture-pass');
check($r['code']===201,'exhausted local referral accepted');
echo "REGISTRATION_PRODUCT_OPTIONAL_INVITE_OK\n";
}
