<?php


namespace app\admin\model;

use think\Model;
use Qiniu\Storage\UploadManager;
use Qiniu\Auth;
use think\facade\Env;
use think\facade\Log;
use Qcloud\Cos\Client;
use app\common\service\UploadStorageConfig;

class Upload extends Model
{
    //单文件上传
    public function single_file_upload($file, $file_category_name = 'all')
    {
        $config = get_system_config();
        $storage = new UploadStorageConfig($config);
        $file_category_name = $this->normalizeCategory($file_category_name);
        $reslut =  ['code' => 201, 'msg' => '非法上传存储类型', 'data' => null];
        if ($storage->driver() === 'local') {
            $reslut  = $this->local_upload($file, $file_category_name);
        } elseif ($storage->driver() === 'qiniu') {
            $reslut  = $this->qiniu_upload($file, $file_category_name);
        } elseif ($storage->driver() === 'cos') {
            $reslut = $this->tencent_upload($file, $file_category_name, $storage);
        }
        return $reslut;
    }

    public function tencent_upload($file, $file_category_name = 'all', UploadStorageConfig $storage = null)
    {
        $config = get_system_config();
        $storage = $storage ?: new UploadStorageConfig($config);
        $cos = $storage->cos();
        if ($storage->missingCosSettings()) {
            return ['code' => 503, 'msg' => 'COS存储配置不完整', 'data' => null];
        }
        if (empty($file)) {
            return  ['code' => 201, 'msg' => '请上传文件', 'data' => null];
        }

        $validate_reslut = $file->validate(['ext' => 'jpeg,jpg,png,gif,webp,mp4,mp3,wgt,svga,txt'])->check();
        if (!$validate_reslut) {
            return  ['code' => 201, 'msg' => '非法上传存储类型', 'data' => null];
        }
        
        $extension = strtolower(pathinfo($file->getInfo('name'), PATHINFO_EXTENSION));
        $file_info = $file->getInfo();


        $filePath = $file_info['tmp_name'];
        $key = $file_category_name . '/' . date('Ymd') . '/' . bin2hex(random_bytes(16)) . '.' . $extension;

        try {
            $cosClient = new Client([
                'region' => $cos['region'],
                'schema' => 'https',
                'allow_accelerate' => $cos['accelerate'],
                'credentials' => [
                    'secretId' => $cos['secret_id'],
                    'secretKey' => $cos['secret_key'],
                ],
            ]);
            $stream = fopen($filePath, 'rb');
            if (!$stream) {
                return ['code' => 201, 'msg' => '无法读取上传文件', 'data' => null];
            }
            try {
                $cosClient->putObject([
                    'Bucket' => $cos['bucket'],
                    'Key' => $key,
                    'Body' => $stream,
                ]);
            } finally {
                if (is_resource($stream)) {
                    fclose($stream);
                }
            }

            $data = [];
            $data['image_path'] = $key;
            $data['size'] = $file_info['size'];
            $data['http_image_path'] = $storage->publicUrl($key);
            $data['mp4_task_id'] = '';
            $data['extension'] = $extension;
            return  ['code' => 200, 'msg' =>  "上传成功", 'data' => $data];
        } catch (\Throwable $exception) {
            Log::error('[COS upload] ' . $exception->getMessage());
            return ['code' => 503, 'msg' => 'COS上传失败', 'data' => null];
        }
    }

    private function normalizeCategory($category)
    {
        $category = preg_replace('/[^a-zA-Z0-9_-]/', '', (string) $category);
        return $category === '' ? 'all' : $category;
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
        $root_path = rtrim((string) Env::get('root_path'), '/\\')
            . DIRECTORY_SEPARATOR . 'public'
            . DIRECTORY_SEPARATOR . 'uploads'
            . DIRECTORY_SEPARATOR . $file_category_name;
        mkdirs($root_path); //创建文件
        $info = $file->validate(['ext' => 'jpeg,jpg,png,gif,webp,mp4,mp3,wgt,svga'])->move($root_path);
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

        $validate_reslut = $file->validate(['ext' => 'jpeg,jpg,png,gif,webp,mp4,mp3,wgt,svga'])->check();
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
