<?php
// +----------------------------------------------------------------------
// | ThinkPHP [ WE CAN DO IT JUST THINK ]
// +----------------------------------------------------------------------
// | Copyright (c) 2006-2016 http://thinkphp.cn All rights reserved.
// +----------------------------------------------------------------------
// | Licensed ( http://www.apache.org/licenses/LICENSE-2.0 )
// +----------------------------------------------------------------------
// | Author: 流年 <liu21st@gmail.com>
// +----------------------------------------------------------------------
use think\Db;

// 应用公共文件
if (!function_exists('datetime')) {
    function datetime($time, $format = 'Y-m-d H:i:s')
    {
        $time = is_numeric($time) ? $time : strtotime($time);
        return date($format, $time);
    }
}
function ajaxReturn($code, $msg = "", $data = "")
{
    header("content:application/json;chartset=uft-8");
    $return = ['code' => $code, 'msg' => $msg, 'data' => $data];
    //return json($return);
    echo json_encode($return);
    exit();
}

function validate_param_sign($data, $login_token)
{
    $app_debug = config('app.app_debug');
    if (config('app.app_debug')) {
        return ['code' => 200, 'msg' => '参数合法', 'data' => null];
    }
    if (empty($data['sign'])) {
        if ($app_debug) {
            return ['code' => 201, 'msg' => 'sign参数不存在', 'data' => null];
        }
        return ['code' => 201, 'msg' => '参数非法', 'data' => null];
    }
    if (empty($data['timestamp'])) {
        if ($app_debug) {
            return ['code' => 201, 'msg' => 'timestamp参数不存在', 'data' => null];
        }
        return ['code' => 201, 'msg' => '参数非法', 'data' => null];
    }
    if (!$app_debug) {
        if (time() - $data['timestamp'] > 5) {
            return ['code' => 201, 'msg' => '参数非法', 'data' => null];
        }
    }
    $sign = $data['sign'];
    unset($data['sign']);
    ksort($data);
    $param_sign = md5(http_build_query($data) . $login_token);
    if ($param_sign != $sign) {
        if ($app_debug) {
            return ['code' => 201, 'msg' => "验签失败，正确sing值：$param_sign", 'data' => null];
        }
        return ['code' => 201, 'msg' => '参数非法', 'data' => null];
    }
    return ['code' => 200, 'msg' => '参数合法', 'data' => null];
}
//生成随机数

function generateRandom($num = 0)
{
    $code = strtolower('ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789');
    $str = '';
    for ($i = 0; $i < $num; $i++) {
        $str .= $code[mt_rand(0, strlen($code) - 1)];
    }
    return $str;
}



//判断文件夹是否存在不存在则创建

function mkdirs($dir, $mode = 0777)
{
    if (is_dir($dir) || @mkdir($dir, $mode)) {
        return true;
    }
    if (!mkdirs(dirname($dir), $mode)) {
        return false;
    }
    return @mkdir($dir, $mode);
}

//获取系统配置信息
function get_system_config($key = "")
{
    return model('admin/Config')->get_system_config($key);
}

function is_sms_test_code_allowed($code)
{
    return filter_var(getenv('SMS_ALLOW_TEST_CODE') ?: 'false', FILTER_VALIDATE_BOOLEAN)
        && (string) $code === '9999';
}

//获取系统配置信息
function get_uncache_system_config($key = "")
{
    return model('admin/Config')->get_uncache_system_config($key);
}

//本地地址转换为网络地址
function localpath_to_netpath($path)
{
    $config = get_system_config();
    $path = str_replace('\\', '/', trim((string) $path));
    if ($path === '' || preg_match('#^https?://#i', $path)) {
        return $path;
    }

    $relativePath = ltrim($path, '/');
    if (strpos($relativePath, 'uploads/') === 0) {
        $relativePath = substr($relativePath, 8);
    }
    $legacyFile = rtrim((string) \think\facade\Env::get('root_path'), '/\\')
        . DIRECTORY_SEPARATOR . 'public'
        . DIRECTORY_SEPARATOR . 'uploads'
        . DIRECTORY_SEPARATOR . str_replace('/', DIRECTORY_SEPARATOR, $relativePath);
    if (is_file($legacyFile)) {
        $legacyBaseUrl = rtrim((string) ($config['loacl_resource_url'] ?? ''), '/');
        if (!preg_match('#/uploads$#i', $legacyBaseUrl)) {
            $legacyBaseUrl .= '/uploads';
        }
        return $legacyBaseUrl . '/' . $relativePath;
    }

    $storage = new \app\common\service\UploadStorageConfig($config);
    return $storage->publicUrl($path);
}

function request_client_ip()
{
    return \app\common\service\ClientIpResolver::resolve($_SERVER);
}
//根据IP地址转换为地理位置
function ip_to_position($ip)
{
    $Ip2Region = new \Ip2Region\Ip2Region();
    $reslut = $Ip2Region->memorySearch($ip);
    $data = explode('|', $reslut['region']);
    return ['country' => $data[0], 'province' => $data[2], 'city' => $data[3]];
}

function myCurl($url, $post_data = array(), $header = array(), $cookie = "")
{

    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_HEADER, false);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1); //如果把这行注释掉的话，就会直接输出
    $curl_header = array();
    if (!empty($header)) {
        foreach ($header as $k => $v) {
            $curl_header[] = "$k:$v";
        }
        curl_setopt($ch, CURLOPT_HTTPHEADER, $curl_header);
    }

    if (!empty($post_data)) {
        curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query($post_data));
    }
    if (!empty($cookie)) {
        curl_setopt($ch, CURLOPT_COOKIEFILE, $cookie);
    }
    curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
    curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
    $result = curl_exec($ch);
    curl_close($ch);
    return $result;
}

function get_content_description($content, $str_len = 10)
{
    $post_excerpt = strip_tags(htmlspecialchars_decode($content));
    $post_excerpt = trim($post_excerpt);
    $patternArr = array('/\s/', '/ /');
    $replaceArr = array('', '');
    $post_excerpt = preg_replace($patternArr, $replaceArr, $post_excerpt);
    return mb_strcut($post_excerpt, 0, $str_len, 'utf-8');
}

function add_operation($type, $uid)
{

    $data = [];
    $data['type'] = $type;
    $data['id'] = $uid;
    $data['url'] = request()->url(true);
    $data['param'] = json_encode(request()->param());
    $data['header'] = json_encode(request()->header());
    $data['user_agent'] = json_encode(request()->header('user-agent'));

    $data['add_ip'] = request_client_ip();
    $address = ip_to_position($data['add_ip']);
    $data['country'] = $address['country'];
    $data['province'] = $address['province'];
    $data['city'] = $address['city'];
    $data['add_time'] = time();
    db::name('operation')->insert($data);
}

function formatBytes($size)
{
    $units = array(' B', ' KB', ' MB', ' GB', ' TB');
    for ($i = 0; $size >= 1024 && $i < 4; $i++) {
        $size /= 1024;
    }
    return round($size, 2) . $units[$i];
}

function removeEmoji($clean_text)
{

    // Match Emoticons
    $regexEmoticons = '/[\x{1F600}-\x{1F64F}]/u';
    $clean_text = preg_replace($regexEmoticons, '', $clean_text);

    // Match Miscellaneous Symbols and Pictographs
    $regexSymbols = '/[\x{1F300}-\x{1F5FF}]/u';
    $clean_text = preg_replace($regexSymbols, '', $clean_text);

    // Match Transport And Map Symbols
    $regexTransport = '/[\x{1F680}-\x{1F6FF}]/u';
    $clean_text = preg_replace($regexTransport, '', $clean_text);

    // Match Miscellaneous Symbols
    $regexMisc = '/[\x{2600}-\x{26FF}]/u';
    $clean_text = preg_replace($regexMisc, '', $clean_text);

    // Match Dingbats
    $regexDingbats = '/[\x{2700}-\x{27BF}]/u';
    $clean_text = preg_replace($regexDingbats, '', $clean_text);

    return $clean_text;
}

function connectionRedis()
{
    $redis = new Redis();
    $redis->connect(
        getenv('REDIS_HOST') ?: '127.0.0.1',
        (int) (getenv('REDIS_PORT') ?: 6379)
    );
    return $redis;
}
//redis 锁 默认锁定60s
function redis_lock($key, $value = 1, $time = 10)
{
    // return true;

    $redis = connectionRedis();
    $is_lock = $redis->setnx($key, $value);
    if ($is_lock) {
        $redis->setex($key, $time, $value);
        return true;
    } else {
        return false;
        return ajaxReturn(302, '访问频繁，请稍后重试', []);
    }
}
function redis_lock_exit($key, $value = 1, $time = 10)
{
    // return true;
    $redis = connectionRedis();
    $is_lock = $redis->setnx($key, $value);
    if (!$is_lock) {
        return ajaxReturn(302, '访问频繁，请稍后重试', []);
    } else {
        $redis->setex($key, $time, $value);
        return true;
    }
}
function redis_lock_exits($key, $value = 1, $time = 10)
{
    // return true;
    $redis = connectionRedis();
    $is_lock = $redis->setnx($key, $value);
    if (!$is_lock) {
        return false;
    } else {
        $redis->setex($key, $time, $value);
        return true;
    }
}
function redis_unlock($key)
{
    // return true;

    $redis = connectionRedis();
    $redis->del($key);
    return true;
}

function set_ip(){
    $redis = connectionRedis();
    $IP = get_real_ip();
    echo '【当前IP:'.$IP."】<br>其它IP:<br>";
    
    $IP = 'ip:'.$IP;
    $t = 60*60*24;
    $ts = date('Y-m-d H:i:s');
    // $redis->del($IP);exit;
    $redis->set($IP, $ts, $t);
    //$redis->pexpire($IP, $t);
    $list = $redis->Keys('ip:*'); 
    foreach($list as $val){
        echo $val."<br>";
    }
}

function set_ips($ip){
    $redis = connectionRedis();
    $IP = $ip;
    echo '【当前IP:'.$IP."】<br>其它IP:<br>";
    
    $IP = 'ip:'.$IP;
    $t = 60*60*24;
    $ts = date('Y-m-d H:i:s');
    // $redis->del($IP);exit;
    $redis->set($IP, $ts, $t);
    //$redis->pexpire($IP, $t);
    $list = $redis->Keys('ip:*'); 
    foreach($list as $val){
        echo $val."<br>";
    }
}

function del_ip($ip){
    $redis = connectionRedis();
    $ip = 'ip:'.$ip;
    $redis->del($ip);
    $list = $redis->Keys('ip:*'); 
    foreach($list as $val){
        echo $val."<br>";
    }
}

function has_ip(){
    $redis = connectionRedis();
    $IP = get_real_ip();
    $IP = 'ip:'.$IP;
    if($redis->exists($IP)){
	    return true;
	}
	return false;
}

function get_real_ip()
{
    static $realip;
    if (isset($_SERVER)) {
        if (isset($_SERVER['HTTP_X_FORWARDED_FOR'])) {
            $realip = $_SERVER['HTTP_X_FORWARDED_FOR'];
        } else {
            if (isset($_SERVER['HTTP_CLIENT_IP'])) {
                $realip = $_SERVER['HTTP_CLIENT_IP'];
            } else {
                $realip = $_SERVER['REMOTE_ADDR'];
            }
        }
    } else {
        if (getenv('HTTP_X_FORWARDED_FOR')) {
            $realip = getenv('HTTP_X_FORWARDED_FOR');
        } else {
            if (getenv('HTTP_CLIENT_IP')) {
                $realip = getenv('HTTP_CLIENT_IP');
            } else {
                $realip = getenv('REMOTE_ADDR');
            }
        }
    }
    return $realip;
}
/**
 * @desc arraySort php二维数组排序 按照指定的key 对数组进行自然排序
 * @param array $arr 将要排序的数组
 * @param string $keys 指定排序的key
 * @param string $type 排序类型 asc | desc
 * @return array
 */

function arraySort($arr, $keys, $type = 'asc')
{
    $keysvalue = $new_array = array();
    foreach ($arr as $k => $v) {
        $keysvalue[$k] = $v[$keys];
    }

    $type == 'asc' ? asort($keysvalue) : arsort($keysvalue);
    foreach ($keysvalue as $k => $v) {
        $new_array[$k] = $arr[$k];
    }
    return $new_array;
}

function secondary_password(){
    $password = getenv('QCH_SECONDARY_PASSWORD_MD5');
    if (!is_string($password) || !preg_match('/^[a-f0-9]{32}$/i', $password)) {
        throw new \RuntimeException('QCH_SECONDARY_PASSWORD_MD5 is not configured');
    }
    return $password;
}

function check_password_format($password)
{
    // $password=132;
    //正则验证   密码必须包含数字、字母以及特殊字符，长度8-20位
    $rule = "/^(?![a-zA-Z]+$)(?![A-Z0-9]+$)(?![A-Z\W_]+$)(?![a-z0-9]+$)(?![a-z\W_]+$)(?![0-9\W_]+$)[a-zA-Z0-9\W_]{8,12}$/";
    if (!preg_match($rule,$password)) {
        return ['code' => 201, 'msg' => '密码必须包含数字、字母以及特殊字符，长度8-12位', 'data' => null];
    }
 
    //键盘连续字符活数字
    $str_continuities = array(
        "1234567890 0987654321", //数字倒序
        "qwertyuiop asdfghjkl zxcvbnm QWERTYUIOP ASDFGHJKL ZXCVBNM", //主键盘顺序
        "poiuytrewq lkjhgfdsa mnbvcxz POIUYTREWQ LKJHGFDSA MNBVCXZ", //主键盘逆序
        "qaz wsx edc rfv tgb yhn ujm QAZ WSX EDC RFV TGB YHN UJM",//主键盘正向斜
        "zaq xsw cde vfr bgt nhy mju ZAQ XSW CDE VFR BGT NHY MJU",//主键盘正向斜逆序
        "esz rdx tfc ygv uhb ijn okm OKM IJN UHB YGV TFC RDX ESZ",//主键盘反向斜
        "zse xdr cft vgy bhu nji mko MKO NJI BHU VGY CFT XDR ZSE",//主键盘反向斜逆序
        "147 369 258 852 963 741" //小键盘
        //特殊字符不计算在内 否则无休止
    );

    //$last_char = "";
    $list_char_3 = "";//连续三个字符
    $chars =preg_split('/(?<!^)(?!$)/u', $password); //也行是中文标点
    foreach ($chars as $char) {
        $list_char_3 .= $char;
        //  $last_char = $char;
        //判断三连

        if (strlen($list_char_3) >= 3) {
            $list_char_3 = substr($list_char_3, strlen($list_char_3) - 3, 3);
            foreach ($str_continuities as $str_continuity) {
                if (strpos($str_continuity, $list_char_3) !== false) {
                // return  '密码不能包括连续的3个字符键盘键位';
                    return ['code' => 201, 'msg' => '密码不能包含连续的3个字符键位', 'data' => null];
                }
            }
        }
    }
    return ['code' => 200, 'msg' => '通过', 'data' => null];
}

function sha256($code){
    return hash('sha256', $code);
}

function validate_user_name($account) {
    return preg_match('/^[a-zA-Z0-9]+$/', $account);
}
