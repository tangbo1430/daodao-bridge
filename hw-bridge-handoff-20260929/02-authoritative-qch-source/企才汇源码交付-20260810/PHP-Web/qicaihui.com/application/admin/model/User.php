<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class User extends Model
{
    protected $pk = 'uid';
    protected $auto = ['update_time'];
    protected $insert = [
        'add_time',
        'reg_code',
        'nick_name' => '默认昵称',
        'head_pic' => 'head_pic/head_pic1.png',
        'sex' => 0,
        'birthday' => '',
        'pid' => 0,
        'login_status' => 1,
        'last_login_time' => 0,
        'money' => 0,
        'integral' => 0,
        'frozen_money' => 0,
        'login_token' => '',
        'system' => 0,
        'reg_code_path' => '',
        'login_ip' => '',

    ];
    protected $update = ['update_time'];

    protected function setPasswordAttr($value)
    {
        return md5($value);
    }
    protected function setBirthdayAttr($value)
    {
        return date('Y-m-d');
    }

    protected function setAddTimeAttr()
    {
        return time();
    }
    protected function setNickNameAttr($value)
    {
        return removeEmoji($value);
    }

    protected function setUpdateTimeAttr()
    {
        return time();
    }
    protected function setRegCodeAttr()
    {
        $code = 'ABCDEFGHIJKLMNPQRSTUVWXYZ123456789';
        $reg_code = '';
        for ($i = 0; $i < 6; $i++) {
            $reg_code .= $code[mt_rand(0, strlen($code) - 1)];
        }
        $user_info = db::name('user')->where(['reg_code' => $reg_code])->find();
        if (!empty($user_info)) {
            $this->setRegCodeAttr();
        } else {
            return $reg_code;
        }
    }

    public function get_user_list($uid, $user_name, $nick_name, $order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        if (!empty($uid)) {
            $map[] = ['uid', '=', $uid];
        }

        if (!empty($user_name)) {
            $map[] = ['user_name', 'like', '%' . $user_name . '%'];
        }
        if (!empty($nick_name)) {
            $map[] = ['nick_name', 'like', '%' . $nick_name . '%'];
        }
        $map[] = ['is_show', '=', 1];
        $map[] = ['system_type', '=', 2];
        $user_list = db::name('user')->where($map)->order($order, $sort)->page($page, $limit)->select();
        foreach ($user_list as $k => &$v) {
            $v['head_pic'] = localpath_to_netpath($v['head_pic']);
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
        }
        $data = [];
        $data['count'] = db::name('user')->where($map)->count();
        $data['list'] = $user_list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    public function get_user_info($uid)
    {
        if (empty($uid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $user_info = db::name('user')->find($uid);
        if ($user_info['sex'] == 0) {
            $user_info['sex'] = 1;
        }
        $user_info['head_pic'] = localpath_to_netpath($user_info['head_pic']);
        $user_info['nick_name'] = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        if (empty($user_info['vip_end_time'])) {
            $user_info['vip_end_time'] = '';
        } else {
            $user_info['vip_end_time'] = date('Y-m-d', $user_info['vip_end_time']);
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $user_info];
    }
    public function edit_user_info($uid, $nick_name, $sex, $login_status)
    {

        if (empty($uid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $user_info = db::name('user')->find($uid);
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }


        $update_data = [];
        $update_data['uid'] = $user_info['uid'];

        $update_data['nick_name'] = $nick_name;
        $update_data['base64_nick_name'] = base64_encode($nick_name);
        $update_data['sex'] = $sex;
        $update_data['login_status'] = $login_status;

        //用户登录状态修改
        if ($login_status == 2) {
            $update_data['login_token'] = "";
        }
        $validate = validate('user');
        $reslut = $validate->scene('adminEdit')->check($update_data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }
        $reslut = db::name('user')->where('uid', $uid)->update($update_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        }
    }
    public function add_user_info($user_name, $password)
    {
        $password = trim($password);
        if(empty($user_name)) {
            return ['code' => 201, 'msg' => '请输入登录账号', 'data' => null];
        }
        if(empty($password)) {
            return ['code' => 201, 'msg' => '请输入登录密码', 'data' => null];
        }
        
        $map = ['user_name' => $user_name];
        $user_info = db::name('user')->where($map)->find();
        if ($user_info) {
            return ['code' => 201, 'msg' => '账号已存在', 'data' => null];
        }
        $nick_name = $this->get_rand_nick_name();
        $reg_ip = request()->ip();
        $position = ip_to_position($reg_ip);
        $birthday = date('Y-m-d');
        $constellation = model('api/User')->get_user_constellation($birthday);
        if($constellation['code'] == 200) {
            $constellation = $constellation['data'];
        } else {
            $constellation = '';
        }
        $reg_code = $this->setRegCodeAttr();
        // $uid = 1001;
        // dump($reg_code);die;
        $uid = model('api/User')->get_available_uid($uid = 0);
        $insert_data = [
            'user_name' => $user_name,
            'password' => md5($password),
            'head_pic' => 'head_pic/head_pic.png',
            'nick_name' => $nick_name,
            'add_time' => time(),
            'update_time' => time(),
            'base64_nick_name' => base64_encode($nick_name),
            'hobby' => '暂无',
            'reg_ip' => $reg_ip,
            'is_admin' => 1,
            'country' => $position['country'],
            'province' => $position['province'],
            'city' => $position['city'],
            'birthday' => $birthday,
            'constellation' => $constellation,
            'autograph' => '这个人很懒，什么都没写',
            'hobby' => '暂无',
            'reg_code' => $reg_code,
            'uid' => $uid
        ];
        // dump($insert_data);die;
        // $reg_ip = request()->ip();
        // $data['reg_ip'] = $reg_ip;
        // $position = ip_to_position($reg_ip);
        // $data['country'] = $position['country'];
        // $data['province'] = $position['province'];
        // $data['city'] = $position['city'];
        // $birthday = date('Y-m-d');
        // $data['birthday'] = $birthday;
        // $constellation = $this->get_user_constellation($birthday);
        // $data['constellation'] = $constellation['data'];
        // $data['head_pic'] = 'head_pic/head_pic.png';
        // $data['autograph'] = '这个人很懒，什么都没写';
        // $data['hobby'] = '暂无';
        
        $reslut = db::name('user')->insert($insert_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        }
    }
    
    public function add_user_info_new($user_name, $password)
    {
        $password = trim($password);
        if(empty($user_name)) {
            return ['code' => 201, 'msg' => '请输入登录账号', 'data' => null];
        }
        if(empty($password)) {
            return ['code' => 201, 'msg' => '请输入登录密码', 'data' => null];
        }
        
        $map = ['user_name' => $user_name];
        $user_info = db::name('user')->where($map)->find();
        if ($user_info) {
            return ['code' => 201, 'msg' => '账号已存在', 'data' => null];
        }
        $nick_name = $this->get_rand_nick_name();
        $reg_ip = request()->ip();
        $position = ip_to_position($reg_ip);
        $birthday = date('Y-m-d');
        $constellation = model('api/User')->get_user_constellation($birthday);
        if($constellation['code'] == 200) {
            $constellation = $constellation['data'];
        } else {
            $constellation = '';
        }
        $reg_code = $this->setRegCodeAttr();
        // $uid = 1001;
        // dump($reg_code);die;
        $uid = model('api/User')->get_available_uid($uid = 0);
        $insert_data = [
            'user_name' => $user_name,
            'password' => md5($password),
            'head_pic' => 'head_pic/head_pic1.png',
            'nick_name' => $nick_name,
            'add_time' => time(),
            'update_time' => time(),
            'base64_nick_name' => base64_encode($nick_name),
            'hobby' => '暂无',
            'reg_ip' => $reg_ip,
            'is_admin' => 1,
            'country' => $position['country'],
            'province' => $position['province'],
            'city' => $position['city'],
            'birthday' => $birthday,
            'constellation' => $constellation,
            'autograph' => '这个人很懒，什么都没写',
            'hobby' => '暂无',
            'reg_code' => $reg_code,
            'uid' => $uid,
            'is_show' => 2,
            'system_type' => 2,
        ];
        // dump($insert_data);die;
        // $reg_ip = request()->ip();
        // $data['reg_ip'] = $reg_ip;
        // $position = ip_to_position($reg_ip);
        // $data['country'] = $position['country'];
        // $data['province'] = $position['province'];
        // $data['city'] = $position['city'];
        // $birthday = date('Y-m-d');
        // $data['birthday'] = $birthday;
        // $constellation = $this->get_user_constellation($birthday);
        // $data['constellation'] = $constellation['data'];
        // $data['head_pic'] = 'head_pic/head_pic.png';
        // $data['autograph'] = '这个人很懒，什么都没写';
        // $data['hobby'] = '暂无';
        
        $reslut = db::name('user')->insert($insert_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        }
    }

    public function edit_user_password($data)
    {
        $validate = validate('user');
        $reslut = $validate->scene('adminEditPassword')->check($data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }
        $data['login_token'] = '';
        $reslut = model('user')->isUpdate(true)->save($data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '编辑失败', 'data' => ''];
        } else {
            return ['code' => 200, 'msg' => '编辑成功', 'data' => ''];
        }
    }

    public function edit_user_vip_time($data)
    {

        $user_info = db::name('user')->find($data['uid']);
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '用户信息不存在', 'data' => null];
        }
        $now_time = time();
        //当前用户是否有会员

        if ($data['day'] < 1) {
            return ['code' => 201, 'msg' => '增加会员时间不能小于1天', 'data' => null];
        }
        if (ceil($data['day']) != $data['day']) {
            return ['code' => 201, 'msg' => '会员天数必须为整数', 'data' => null];
        }

        if ($user_info['is_vip'] == 1) {
            if ($user_info['vip_end_time'] < $now_time) {
                $vip_end_time = $now_time + ($data['day'] * 60 * 60 * 24);
            } else {
                $vip_end_time = $user_info['vip_end_time'] + ($data['day'] * 60 * 60 * 24);
            }
        } else {
            $vip_end_time = $now_time + ($data['day'] * 60 * 60 * 24);
        }

        $update = [];
        $update['is_vip'] = 1;
        $update['vip_end_time'] = $vip_end_time;
        $reslut = db::name('user')->where('uid', $data['uid'])->update($update);
        if ($reslut) {
            return ['code' => 201, 'msg' => '成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        }
    }

    public function edit_users_vip_time($data)
    {

        $user_info = db::name('user')->find($data['uid']);
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '用户信息不存在', 'data' => null];
        }
        $now_time = time();
        $time = strtotime(date('Y-m-d'));
        //当前用户是否有会员

        $end_time = strtotime($data['vip_end_time']);
        if ($time == $end_time) {
            $vip_end_time = 0;
            $is_vip = 2;
        } else if ($time > $end_time) {
            $vip_end_time = 0;
            $is_vip = 2;
        } else if ($time < $end_time) {
            $vip_end_time = $end_time;
            $is_vip = 1;
        } else {
            return ['code' => 201, 'msg' => '参数错误', 'data' => null];
        }

        $update = [];
        $update['is_vip'] = $is_vip;
        $update['vip_end_time'] = $vip_end_time;
        $update['update_time'] = time();
        $reslut = db::name('user')->where('uid', $data['uid'])->update($update);
        if ($reslut) {
            return ['code' => 201, 'msg' => '成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        }
    }

    public function get_rand_nick_name()
    {
        $nicheng_tou = array('快乐的', '冷静的', '醉熏的', '潇洒的', '糊涂的', '积极的', '冷酷的', '深情的', '粗暴的', '温柔的', '可爱的', '愉快的', '义气的', '认真的', '威武的', '帅气的', '传统的', '潇洒的', '漂亮的', '自然的', '专一的', '听话的', '昏睡的', '狂野的', '等待的', '搞怪的', '幽默的', '魁梧的', '活泼的', '开心的', '高兴的', '超帅的', '留胡子的', '坦率的', '直率的', '轻松的', '痴情的', '完美的', '精明的', '无聊的', '有魅力的', '丰富的', '繁荣的', '饱满的', '炙热的', '暴躁的', '碧蓝的', '俊逸的', '英勇的', '健忘的', '故意的', '无心的', '土豪的', '朴实的', '兴奋的', '幸福的', '淡定的', '不安的', '阔达的', '孤独的', '独特的', '疯狂的', '时尚的', '落后的', '风趣的', '忧伤的', '大胆的', '爱笑的', '矮小的', '健康的', '合适的', '玩命的', '沉默的', '斯文的', '香蕉', '苹果', '鲤鱼', '鳗鱼', '任性的', '细心的', '粗心的', '大意的', '甜甜的', '酷酷的', '健壮的', '英俊的', '霸气的', '阳光的', '默默的', '大力的', '孝顺的', '忧虑的', '着急的', '紧张的', '善良的', '凶狠的', '害怕的', '重要的', '危机的', '欢喜的', '欣慰的', '满意的', '跳跃的', '诚心的', '称心的', '如意的', '怡然的', '娇气的', '无奈的', '无语的', '激动的', '愤怒的', '美好的', '感动的', '激情的', '激昂的', '震动的', '虚拟的', '超级的', '寒冷的', '精明的', '明理的', '犹豫的', '忧郁的', '寂寞的', '奋斗的', '勤奋的', '现代的', '过时的', '稳重的', '热情的', '含蓄的', '开放的', '无辜的', '多情的', '纯真的', '拉长的', '热心的', '从容的', '体贴的', '风中的', '曾经的', '追寻的', '儒雅的', '优雅的', '开朗的', '外向的', '内向的', '清爽的', '文艺的', '长情的', '平常的', '单身的', '伶俐的', '高大的', '懦弱的', '柔弱的', '爱笑的', '乐观的', '耍酷的', '酷炫的', '神勇的', '年轻的', '唠叨的', '瘦瘦的', '无情的', '包容的', '顺心的', '畅快的', '舒适的', '靓丽的', '负责的', '背后的', '简单的', '谦让的', '彩色的', '缥缈的', '欢呼的', '生动的', '复杂的', '慈祥的', '仁爱的', '魔幻的', '虚幻的', '淡然的', '受伤的', '雪白的', '高高的', '糟糕的', '顺利的', '闪闪的', '羞涩的', '缓慢的', '迅速的', '优秀的', '聪明的', '含糊的', '俏皮的', '淡淡的', '坚强的', '平淡的', '欣喜的', '能干的', '灵巧的', '友好的', '机智的', '机灵的', '正直的', '谨慎的', '俭朴的', '殷勤的', '虚心的', '辛勤的', '自觉的', '无私的', '无限的', '踏实的', '老实的', '现实的', '可靠的', '务实的', '拼搏的', '个性的', '粗犷的', '活力的', '成就的', '勤劳的', '单纯的', '落寞的', '朴素的', '悲凉的', '忧心的', '洁净的', '清秀的', '自由的', '小巧的', '单薄的', '贪玩的', '刻苦的', '干净的', '壮观的', '和谐的', '文静的', '调皮的', '害羞的', '安详的', '自信的', '端庄的', '坚定的', '美满的', '舒心的', '温暖的', '专注的', '勤恳的', '美丽的', '腼腆的', '优美的', '甜美的', '甜蜜的', '整齐的', '动人的', '典雅的', '尊敬的', '舒服的', '妩媚的', '秀丽的', '喜悦的', '甜美的', '彪壮的', '强健的', '大方的', '俊秀的', '聪慧的', '迷人的', '陶醉的', '悦耳的', '动听的', '明亮的', '结实的', '魁梧的', '标致的', '清脆的', '敏感的', '光亮的', '大气的', '老迟到的', '知性的', '冷傲的', '呆萌的', '野性的', '隐形的', '笑点低的', '微笑的', '笨笨的', '难过的', '沉静的', '火星上的', '失眠的', '安静的', '纯情的', '要减肥的', '迷路的', '烂漫的', '哭泣的', '贤惠的', '苗条的', '温婉的', '发嗲的', '会撒娇的', '贪玩的', '执着的', '眯眯眼的', '花痴的', '想人陪的', '眼睛大的', '高贵的', '傲娇的', '心灵美的', '爱撒娇的', '细腻的', '天真的', '怕黑的', '感性的', '飘逸的', '怕孤独的', '忐忑的', '高挑的', '傻傻的', '冷艳的', '爱听歌的', '还单身的', '怕孤单的', '懵懂的');

        $nicheng_wei = array('嚓茶', '凉面', '便当', '毛豆', '花生', '可乐', '灯泡', '哈密瓜', '野狼', '背包', '眼神', '缘分', '雪碧', '人生', '牛排', '蚂蚁', '飞鸟', '灰狼', '斑马', '汉堡', '悟空', '巨人', '绿茶', '自行车', '保温杯', '大碗', '墨镜', '魔镜', '煎饼', '月饼', '月亮', '星星', '芝麻', '啤酒', '玫瑰', '大叔', '小伙', '哈密瓜，数据线', '太阳', '树叶', '芹菜', '黄蜂', '蜜粉', '蜜蜂', '信封', '西装', '外套', '裙子', '大象', '猫咪', '母鸡', '路灯', '蓝天', '白云', '星月', '彩虹', '微笑', '摩托', '板栗', '高山', '大地', '大树', '电灯胆', '砖头', '楼房', '水池', '鸡翅', '蜻蜓', '红牛', '咖啡', '机器猫', '枕头', '大船', '诺言', '钢笔', '刺猬', '天空', '飞机', '大炮', '冬天', '洋葱', '春天', '夏天', '秋天', '冬日', '航空', '毛衣', '豌豆', '黑米', '玉米', '眼睛', '老鼠', '白羊', '帅哥', '美女', '季节', '鲜花', '服饰', '裙子', '白开水', '秀发', '大山', '火车', '汽车', '歌曲', '舞蹈', '老师', '导师', '方盒', '大米', '麦片', '水杯', '水壶', '手套', '鞋子', '自行车', '鼠标', '手机', '电脑', '书本', '奇迹', '身影', '香烟', '夕阳', '台灯', '宝贝', '未来', '皮带', '钥匙', '心锁', '故事', '花瓣', '滑板', '画笔', '画板', '学姐', '店员', '电源', '饼干', '宝马', '过客', '大白', '时光', '石头', '钻石', '河马', '犀牛', '西牛', '绿草', '抽屉', '柜子', '往事', '寒风', '路人', '橘子', '耳机', '鸵鸟', '朋友', '苗条', '铅笔', '钢笔', '硬币', '热狗', '大侠', '御姐', '萝莉', '毛巾', '期待', '盼望', '白昼', '黑夜', '大门', '黑裤', '钢铁侠', '哑铃', '板凳', '枫叶', '荷花', '乌龟', '仙人掌', '衬衫', '大神', '草丛', '早晨', '心情', '茉莉', '流沙', '蜗牛', '战斗机', '冥王星', '猎豹', '棒球', '篮球', '乐曲', '电话', '网络', '世界', '中心', '鱼', '鸡', '狗', '老虎', '鸭子', '雨', '羽毛', '翅膀', '外套', '火', '丝袜', '书包', '钢笔', '冷风', '八宝粥', '烤鸡', '大雁', '音响', '招牌', '胡萝卜', '冰棍', '帽子', '菠萝', '蛋挞', '香水', '泥猴桃', '吐司', '溪流', '黄豆', '樱桃', '小鸽子', '小蝴蝶', '爆米花', '花卷', '小鸭子', '小海豚', '日记本', '小熊猫', '小懒猪', '小懒虫', '荔枝', '镜子', '曲奇', '金针菇', '小松鼠', '小虾米', '酒窝', '紫菜', '金鱼', '柚子', '果汁', '百褶裙', '项链', '帆布鞋', '火龙果', '奇异果', '煎蛋', '唇彩', '小土豆', '高跟鞋', '戒指', '雪糕', '睫毛', '铃铛', '手链', '香氛', '红酒', '月光', '酸奶', '银耳汤', '咖啡豆', '小蜜蜂', '小蚂蚁', '蜡烛', '棉花糖', '向日葵', '水蜜桃', '小蝴蝶', '小刺猬', '小丸子', '指甲油', '康乃馨', '糖豆', '薯片', '口红', '超短裙', '乌冬面', '冰淇淋', '棒棒糖', '长颈鹿', '豆芽', '发箍', '发卡', '发夹', '发带', '铃铛', '小馒头', '小笼包', '小甜瓜', '冬瓜', '香菇', '小兔子', '含羞草', '短靴', '睫毛膏', '小蘑菇', '跳跳糖', '小白菜', '草莓', '柠檬', '月饼', '百合', '纸鹤', '小天鹅', '云朵', '芒果', '面包', '海燕', '小猫咪', '龙猫', '唇膏', '鞋垫', '羊', '黑猫', '白猫', '万宝路', '金毛', '山水', '音响');

        $nicheng = $nicheng_tou[array_rand($nicheng_tou, 1)] . $nicheng_wei[array_rand($nicheng_wei, 1)];

        return $nicheng; //输出生成的昵称

    }

    public function edit_user_money($uid, $change_value, $secondary_password, $remarks = "", $from_uid = 0)
    {
        if (empty($remarks)) {
            return ['code' => 201, 'msg' => '备注必填', 'data' => null];
        }
        // if(empty($secondary_password)){
        //     return ['code' => 201, 'msg' => '二级密码必填', 'data' => null];
        // }else{
        //     if($secondary_password != '147369'){
        //         return ['code' => 201, 'msg' => '二级密码错误', 'data' => null];
        //     }
        // }

        $change_values = abs($change_value);
        if (strlen($change_values) > 8) {
            return ['code' => 201, 'msg' => '变动的数值过长', 'data' => null];
        }

        return $this->change_user_money_by_uid($uid, $change_value, 1, $remarks, $from_uid);
    }

    //修改用户资金
    public function change_user_money_by_uid($uid, $change_value, $change_type, $remarks = "", $from_uid = 0)
    {

        $user_info = db::name('user')->find($uid);
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => "非法参数", 'data' => null];
        }
        return $this->change_user_money_by_user_info($user_info, $change_value, $change_type,  $remarks, $from_uid,  true);
    }

    //修改用户资金
    public function change_user_money_by_user_info($user_info, $change_value, $change_type, $remarks = "", $from_uid = 0, $is_uid_search = false)
    {
        if (empty($user_info['uid'])) {
            return ['code' => 201, 'msg' => "用户信息错误", 'data' => null];
        }

        if (!$is_uid_search) {
            $user_info = db::name('user')->find($user_info['uid']);
        }

        if (!isset($user_info['uid']) && !isset($user_info['money'])) {
            return ['code' => 201, 'msg' => "用户信息错误", 'data' => null];
        }


        $after_money = $user_info['money'];

        $change_field = "money";
        $after_money += $change_value;
        if ($after_money > 99999999) {
            return ['code' => 201, 'msg' => "当前用户余额已达上限", 'data' => null];
        }

        $ChangeTypeLable = model('admin/UserMoneyLog')->ChangeTypeLable();

        if (empty($ChangeTypeLable[$change_type])) {
            return ['code' => 201, 'msg' => "非法资金变动类型", 'data' => null];
        }
        if (!is_numeric($change_value)) {
            return ['code' => 201, 'msg' => "变动的数值必须为数字", 'data' => null];
        }
        /*
        if (abs($change_value) == 0) {
        return ['code' => 201, 'msg' => "变动的数值不能为0", 'data' => null];
        }
         */


        $change_name = '余额';

        $data = [];
        $data['uid'] = $user_info['uid'];
        $data['change_type'] = $change_type;
        $data['change_value'] = $change_value;
        $data['after_money'] = $after_money;
        $data['remarks'] = $remarks;
        $data['from_uid'] = $from_uid;
        $data['add_time'] = time();
        $data['update_time'] = time();

        Db::startTrans();
        try {
            $map = [];
            $map[] = ['uid', '=', $user_info['uid']];
            if ($change_value < 0) {
                $map[] = [$change_field, '>=', abs($change_value)];
            }
            if (abs($change_value) != 0) {
                $reslut = Db::name('user')->where($map)->setInc($change_field, $change_value);
                if (!$reslut) {
                    Db::rollback();
                    return ['code' => 201, 'msg' => $change_name . "不足", 'data' => null];
                }
            }
            $reslut = Db::name('user_money_log')->insert($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => "请重试", 'data' => null];
            }

            // 提交事务
            Db::commit();
            return ['code' => 200, 'msg' => "修改成功", 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            dump($e);
            return ['code' => 201, 'msg' => "请重试", 'data' => null];
        }
    }
    
    //获取用户实名审核列表
    public function user_real_name_list($uid, $user_name, $nick_name, $order, $sort, $page, $limit)
    {
        $map = [];
        if (!empty($uid)) {
            $map[] = ['a.uid', '=', $uid];
        }

        if (!empty($user_name)) {
            $map[] = ['b.user_name', 'like', '%' . $user_name . '%'];
        }
        if (!empty($nick_name)) {
            $map[] = ['b.nick_name', 'like', '%' . $nick_name . '%'];
        }

        $list = db::name('user_real_name')->alias('a')->join('yy_user b', 'a.uid = b.uid')->where($map)->field('a.*,b.nick_name')->order($order, $sort)->page($page, $limit)->select();
        foreach ($list as $k => &$v) {
            $list[$k]['user_nick_name'] = $v['uid'] . '-' . $v['nick_name'];
        }
        $data = [];
        $data['count'] = db::name('user_real_name')->alias('a')->join('yy_user b', 'a.uid = b.uid')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
        
    }
    
    //获取实名信息
    public function get_user_real_name($nid){
        if (empty($nid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $real_info = db::name('user_real_name')->find($nid);
        $real_info['identity1'] = localpath_to_netpath($real_info['identity1']);
        $real_info['identity2'] = localpath_to_netpath($real_info['identity2']);
        return ['code' => 200, 'msg' => '获取成功', 'data' => $real_info];
    }
    
    //编辑用户实名信息
    public function edit_user_real_name($nid,$status,$remarke){
        if (empty($nid)||empty($status)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        if($status == 2 && empty($remarke)){
            return ['code' => 201, 'msg' => '驳回必须说明原因', 'data' => null];
        }
        
        $info = db::name('user_real_name')->find($nid);
        
        if($status == $info['status']){
            return ['code' => 201, 'msg' => '状态未改变', 'data' => null];
        }
        
        $data = [
            'status'=>$status,
            'remarke' =>$remarke,
            'update_time'=>time()
            ];
        Db::startTrans();
        try{
            $res = db::name('user_real_name')->where('nid',$nid)->update($data);
            if(!$res){
                Db::rollback();
                return ['code' => 201, 'msg' => "修改失败1", 'data' => null];
            }
            $s = 0;
            if($status == 1){
                $s = 3;
            }else if($status == 3){
                $s = 1;
            }else{
                $s = $status;
            }
            $res = db::name('user')->where('uid',$info['uid'])->update(['is_real'=>$s]);
            if(!$res){
                Db::rollback();
                return ['code' => 201, 'msg' => "修改失败2", 'data' => null];
            }
            // 提交事务
            Db::commit();
            return ['code' => 200, 'msg' => "修改成功", 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            
            return ['code' => 201, 'msg' => "请重试", 'data' => null];
        }
    }
    //封建解封
    public function add_banned_user_file($filepath, $type)
    {
        if(empty($type)) {
            return ['code' => 201, 'msg' => "请选择封禁/解封", 'data' => null];
        }
        if(empty($filepath)) {
            return ['code' => 201, 'msg' => "请上传封禁/解封文件", 'data' => null];
        }
        if($type == 1) {
            $login_status = 2;
        } else if($type == 2) {
            $login_status = 1;
        }
        $content = file_get_contents($filepath);
        if($content) {
            // dump($content);
            $array = explode("\n", $content);
            // dump($array);die;
            $uid_arr = [];
            foreach($array as $v) {
                if($v > 0) {
                    $uid_arr[] = intval($v);
                }
            }
            // dump($uid_arr);die;
            Db::startTrans();
            try{
                Db::name('user')->whereIn('uid', $uid_arr)
                ->where('system_type',  config('app.system_type'))
                ->update(['login_status' => $login_status, 'update_time' => time(), 'banned_time' => time()]);
                Db::name('user_banned_file')->insert([
                    'type' => $type,
                    'filepath' => $filepath,
                    'add_time' => time(),
                    'update_time' => time(),
                    'banned_type' => 1,
                    'system_type' => config('app.system_type')
                ]);
                // 提交事务
                Db::commit();
                return ['code' => 200, 'msg' => "操作成功", 'data' => null];
            } catch (\Exception $e) {
                // 回滚事务
                Db::rollback();
                
                return ['code' => 201, 'msg' => "请重试", 'data' => null];
            }
        }
        
    }
    
    //封建解封
    public function add_banned_user_file_ip($filepath, $type)
    {
        if(empty($type)) {
            return ['code' => 201, 'msg' => "请选择封禁/解封", 'data' => null];
        }
        if(empty($filepath)) {
            return ['code' => 201, 'msg' => "请上传封禁/解封文件", 'data' => null];
        }
        if($type == 1) {
            $login_status = 2;
        } else if($type == 2) {
            $login_status = 1;
        }
        $content = file_get_contents($filepath);
        if($content) {
            // dump($content);
            $array = explode("\n", $content);
            // dump($array);die;
            $ip_arr = [];
            foreach($array as $v) {
                if($v > 0) {
                    $ip_arr[] = trim($v);
                }
            }
            // dump($ip_arr);die;
            $uid_arr = Db::name('user')->whereIn('login_ip', $ip_arr)->column('uid');
            // dump($uid_arr);die;
            Db::startTrans();
            try{
                Db::name('user')->whereIn('uid', $uid_arr)
                ->where('system_type',  config('app.system_type'))
                ->update(['login_status' => $login_status, 'update_time' => time(), 'banned_time' => time()]);
                Db::name('user_banned_file')->insert([
                    'type' => $type,
                    'filepath' => $filepath,
                    'add_time' => time(),
                    'update_time' => time(),
                    'banned_type' => 2,
                ]);
                // 提交事务
                Db::commit();
                return ['code' => 200, 'msg' => "操作成功", 'data' => null];
            } catch (\Exception $e) {
                // 回滚事务
                Db::rollback();
                
                return ['code' => 201, 'msg' => "请重试", 'data' => null];
            }
        }
        
    }
    
}
