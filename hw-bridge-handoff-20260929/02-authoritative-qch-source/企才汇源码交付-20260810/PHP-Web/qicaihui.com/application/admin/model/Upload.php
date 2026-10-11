<?php


namespace app\admin\model;

use think\Model;
use Qiniu\Storage\UploadManager;
use Qiniu\Auth;
use think\facade\Env;
use Qcloud\Cos\Client;

class Upload extends Model
{
    //单文件上传
    public function single_file_upload($file, $file_category_name = 'all')
    {
        //判断上传方式
        $config = get_system_config();
        $reslut =  ['code' => 201, 'msg' => '非法上传存储类型', 'data' => null];
        if ($config['file_upload_type'] == 1) {
            $reslut  = $this->local_upload($file, $file_category_name);
        } elseif ($config['file_upload_type'] == 2) {
            $reslut  = $this->qiniu_upload($file, $file_category_name);
        }elseif ($config['file_upload_type'] == 3) {
            $reslut = $this->tencent_upload($file, $file_category_name);
        }
        return $reslut;
    }
    public function tencent_upload($file, $file_category_name = 'all')
    {
        $config = get_system_config();
        $secretId = $config['tencent_secret_id']; 
        // $secretId = 111111; 
        $secretKey = $config['tencent_secret_key']; 
        $region = $config['tencent_territory']; 
        $bucket = $config['tencent_bucket_name'];
        // dump($secretId);
        // dump($secretKey);
        // dump($region);
        // dump($bucket);die;
        if (empty($file)) {
            return  ['code' => 201, 'msg' => '请上传文件', 'data' => null];
        }
        // dump($file);exit;

        $validate_reslut = $file->validate(['ext' => 'jpeg,jpg,png,gif,mp4,mp3,wgt,svga,txt'])->check();
        // dump($file);exit;
        if (!$validate_reslut) {
            return  ['code' => 201, 'msg' => '非法上传存储类型', 'data' => null];
        }
        
        $extension = strtolower(pathinfo($file->getInfo('name'), PATHINFO_EXTENSION));
        $file_info = $file->getInfo();
        // 构建鉴权对象
        $cosClient = new Client(
            array(
                'region' => $region,
                'schema' => 'https', //协议头部，默认为http
                'credentials'=> array(
                    'secretId'  => $secretId ,
                    'secretKey' => $secretKey
                    )
                )
            );
            
            
        // 要上传文件的本地路径
        $filePath = $file_info['tmp_name'];
        // 上传到存储后保存的文件名
        $savename =   date('Ymd') . '/' . md5(microtime(true)) . ".$extension";
        $key = $file_category_name . '/' . $savename;
        // 初始化 UploadManager 对象并进行文件的上传。
        $file = fopen($filePath, "rb");
        
        if ($file) {
            $result = $cosClient->putObject(array(
                'Bucket' => $bucket,
                'Key' => $key,
                'Body' => $file)
                );
                
            $data = [];
            $data['image_path'] = $result['Key'];
            $data['size'] = $file_info['size'];
            $data['http_image_path'] = localpath_to_netpath($result['Key']);
            $data['mp4_task_id'] = '';
            $data['extension'] = $extension;
            

            return  ['code' => 200, 'msg' =>  "上传成功", 'data' => $data];
        }else{
            return  ['code' => 201, 'msg' =>  "上传失败", 'data' => null];
        }
    }
    public function server_local_upload($filePath){
     
        $config = get_system_config();
        $accessKey = $config['qiniu_access_key'];
        $secretKey = $config['qiniu_secret_key'];
        $bucket = $config['qiniu_bucket_name'];
        // 构建鉴权对象
        $auth = new Auth($accessKey, $secretKey);
        // 生成上传 Token
        $token = $auth->uploadToken($bucket);
        // 初始化 UploadManager 对象并进行文件的上传。
        $uploadMgr = new UploadManager();
        // 调用 UploadManager 的 putFile 方法进行文件的上传。
        $local_filePath= Env::get('root_path')."/public/".$filePath;
        list($ret, $err) = $uploadMgr->putFile($token, $filePath,  $local_filePath);
        if ($err !== null) {
            return  ['code' => 201, 'msg' =>  "上传失败", 'data' => null];
        } else {
            $data = [];
            $data['image_path'] = $ret['key'];
            $data['size'] = 0;
            $data['http_image_path'] = localpath_to_netpath($ret['key']);

            return  ['code' => 200, 'msg' =>  "上传成功", 'data' => $data];
        }
    }
    
    

    public function local_upload($file, $file_category_name = 'all')
    {
        // // 获取表单上传文件 例如上传了001.jpg

        // 移动到框架应用根目录/uploads/ 目录下
        $root_path = 'public/uploads/' . $file_category_name;
        mkdirs($root_path); //创建文件
        $info = $file->validate(['ext' => 'jpeg,jpg,png,gif,mp4,mp3,wgt,svga'])->move('./uploads/' . $file_category_name);
        if ($info) {
            $file_info = $info->getInfo();
            // 成功上传后 获取上传信息
            // 输出 jpg
            // echo $info->getExtension();
            // // 输出 20160820/42a79759f284b767dfcb2a0197904287.jpg
            // echo $info->getSaveName();
            // // 输出 42a79759f284b767dfcb2a0197904287.jpg
            // echo $info->getFilename();
            $image_path = '/uploads/' . $file_category_name . '/' . $info->getSaveName(); //返回的地址
            $image_path = str_replace("\\", "/", $image_path);
            $img['image_path'] = $image_path;
            $img['size'] = $file_info['size'];
            $img['http_image_path'] = localpath_to_netpath($image_path);
            return  ['code' => 200, 'msg' => '上传成功', 'data' => $img];
        } else {

            return  ['code' => 201, 'msg' => '上传失败', 'data' => null];
        }
    }
    public function qiniu_upload($file, $file_category_name = 'all')
    {
        $config = get_system_config();
        $accessKey = $config['qiniu_access_key'];
        $secretKey = $config['qiniu_secret_key'];
        $bucket = $config['qiniu_bucket_name'];

        if (empty($file)) {
            return  ['code' => 201, 'msg' => '请上传文件', 'data' => null];
        }

        $validate_reslut = $file->validate(['ext' => 'jpeg,jpg,png,gif,mp4,mp3,wgt,svga'])->check();
        if (!$validate_reslut) {
            return  ['code' => 201, 'msg' => '非法上传存储类型', 'data' => null];
        }
        $extension = strtolower(pathinfo($file->getInfo('name'), PATHINFO_EXTENSION));
        $file_info = $file->getInfo();


        // 构建鉴权对象
        $auth = new Auth($accessKey, $secretKey);
        // 生成上传 Token
        $token = $auth->uploadToken($bucket);
        // 要上传文件的本地路径
        $filePath = $file_info['tmp_name'];
        // 上传到存储后保存的文件名
        $savename =   date('Ymd') . '/' . md5(microtime(true)) . ".$extension";
        $key = $file_category_name . '/' . $savename;
        // 初始化 UploadManager 对象并进行文件的上传。
        $uploadMgr = new UploadManager();
        // 调用 UploadManager 的 putFile 方法进行文件的上传。
        list($ret, $err) = $uploadMgr->putFile($token, $key, $filePath);
        if ($err !== null) {
            return  ['code' => 201, 'msg' =>  "上传失败", 'data' => null];
        } else {
            $data = [];
            $data['image_path'] = $ret['key'];
            $data['size'] = $file_info['size'];
            $data['http_image_path'] = localpath_to_netpath($ret['key']);

            return  ['code' => 200, 'msg' =>  "上传成功", 'data' => $data];
        }
    }



    public function get_qiniu_upload_token()
    {
        $config = get_system_config();
        $accessKey = $config['qiniu_access_key'];
        $secretKey = $config['qiniu_secret_key'];
        $bucket = $config['qiniu_bucket_name'];
        $auth = new Auth($accessKey, $secretKey);
        $token = $auth->uploadToken($bucket);
        return  ['code' => 200, 'msg' => '获取成功', 'data' => ['token' => $token]];
    }
}
