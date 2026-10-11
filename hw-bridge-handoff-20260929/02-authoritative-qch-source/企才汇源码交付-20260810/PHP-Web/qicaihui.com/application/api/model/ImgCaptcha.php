<?php


namespace app\api\model;




use app\common\component\captcha\Captcha;

class ImgCaptcha
{
    protected $config = [
        'codeSet' => '0123456789',
        // 验证码字体大小
        'fontSize' => 30,
        // 验证码位数
        'length' => 4,
        // 关闭验证码杂点
        'useNoise' => false,
        //过期时间
        'expire' => 600,
        'useCurve' => false,
    ];
    //生成验证码
    public function create_captcha()
    {
        $captcha = new Captcha($this->config);
        $id = $this->generateRandom(8). date('YmdHis');
        $res = $captcha->entry($id);
        $base64_image = "data:image/png;base64," . base64_encode($res->getData());
        return ['code' => 200, 'msg' => '获取数据成功', 'data' => ['image' => $base64_image, 'key' => $id]];
    }
    //获取随机数
    private function generateRandom($num = 0)
    {
        $code = strtolower('ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789');
        $str = '';
        for ($i = 0; $i < $num; $i++) {
            $str .= $code[mt_rand(0, strlen($code) - 1)];
        }
        return $str;
    }
    //验证验证码
    public function check_captcha($code, $key)
    {
        $captcha = new Captcha($this->config);
        $res = $captcha->check($code, $key);
        if ($res === true) {
            return ['code' => 200, 'msg' => '验证成功', 'data' => null];
        }
        return ['code' => 201, 'msg' => '验证失败', 'data' => null];
    }
}