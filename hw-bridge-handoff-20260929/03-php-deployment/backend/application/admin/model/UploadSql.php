<?php


namespace app\admin\model;

use think\Model;
use Qiniu\Storage\UploadManager;
use Qiniu\Auth;
use think\facade\Env;
use Qcloud\Cos\Client;
use think\Db;
class UploadSql extends Model
{
    /**
     * 递归上传MySQL备份文件夹中的所有文件
     */
    public  function uploadMysqlBackupFolderRecursive()
    {
        
        set_time_limit(0);
        
        // ini_set('open_basedir', '/www/:/tmp/:/usr/:/www/wwwroot/msql_back/database/mysql/crontab_backup/jishitongxun_com/');
        $local_path = '/www/wwwroot/jishitongxun.com/mysql_back/database/mysql/crontab_backup/jishitongxun_com/';
        // $iterator = new \RecursiveIteratorIterator(
        //     new \RecursiveDirectoryIterator($local_path, \RecursiveDirectoryIterator::SKIP_DOTS),
        //     \RecursiveIteratorIterator::SELF_FIRST
        // );
        $dir_list = scandir($local_path);
        $config = get_system_config();
        $secretId = getenv('TENCENT_SECRET_ID'); 
        // $secretId = 111111; 
        $secretKey = getenv('TENCENT_SECRET_KEY');
        if (!$secretId || !$secretKey) {
            throw new \RuntimeException('Tencent cloud credentials are not configured');
        } 
        $region = 'ap-guangzhou'; 
        $bucket = 'yincaihui-1365057050';
        foreach ($dir_list as $key => $file_name_v) {
            
            if($key <= 2) {
                continue;
            }
            try {
                    $is_back = Db::name('mysql_back')->where('file_name', $file_name_v)->find();
                    if(empty($is_back)) {
                        $local_file_path = $local_path . $file_name_v;
                        if (file_exists($local_file_path)) {
                            $file_info = pathinfo($local_file_path);
                            $extension = $file_info['extension'] ?? '';
                            $file_name = $file_info['basename'];
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
                            $filePath = $local_file_path;
                            // 上传到存储后保存的文件名
                            $savename =   date('Ymd') . '/' . $file_name . ".$extension";
                            $key = 'mysql_back' . '/' . $savename;
                            // 初始化 UploadManager 对象并进行文件的上传。
                            $file = fopen($filePath, "rb");
                            
                            if ($file) {
                                $result = $cosClient->putObject(array(
                                    'Bucket' => $bucket,
                                    'Key' => $key,
                                    'Body' => $file)
                                    );
                                    
                                // $data = [];
                                // $data['image_path'] = $result['Key'];
                                // $data['size'] = $file_info['size'];
                                // $data['http_image_path'] = localpath_to_netpath($result['Key']);
                                // $data['mp4_task_id'] = '';
                                // $data['extension'] = $extension;
                    
                                Db::name('mysql_back')->insert(['file_name' => $file_name_v, 'add_time' => time()]);
                            }
                        }
                        
                    
                    }
                    
                    echo '备份成功';
                    
                } catch (\Exception $e) {
                    echo '备份失败';
                }
        }

        
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

        $validate_reslut = $file->validate(['ext' => 'jpeg,jpg,png,gif,mp4,mp3,wgt,svga'])->check();
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
    
    
    

    
}
