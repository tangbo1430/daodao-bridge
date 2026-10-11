<?php

namespace app\admin\controller;

use think\Controller;

class Ueditor extends Common
{
    public $web_site = "";
    public function initialize()
    {
        // $this->web_site = config('bosUrl') . "/";
    }

    public function index()
    {
        // header('Access-Control-Allow-Origin: *'); //设置http://www.baidu.com允许跨域访问
        // header('Access-Control-Allow-Headers: X-Requested-With,X_Requested_With'); //设置允许的跨域header
        // date_default_timezone_set("Asia/chongqing");
        // error_reporting(E_ERROR);
        // header("Content-Type: text/html; charset=utf-8");

        $path = str_replace('public', "", $_SERVER['DOCUMENT_ROOT']);
        $config_json_path = $path . "/config/ueditconfig.json";
        $CONFIG = json_decode(preg_replace("/\/\*[\s\S]+?\*\//", "", file_get_contents($config_json_path)), true);
        $action = input('action');

        switch ($action) {
            case 'config':
                // header("content:application/json;chartset=uft-8");
                $result = json_encode($CONFIG);
                break;
                /* 上传图片 */
            case 'uploadimage':
                $fieldName = $CONFIG['imageFieldName'];
                $result = $this->upFile($fieldName);
                break;
                /* 上传涂鸦 */
            case 'uploadscrawl':
                $config = array(
                    "pathFormat" => $CONFIG['scrawlPathFormat'],
                    "maxSize" => $CONFIG['scrawlMaxSize'],
                    "allowFiles" => $CONFIG['scrawlAllowFiles'],
                    "oriName" => "scrawl.png",
                );
                $fieldName = $CONFIG['scrawlFieldName'];
                $base64 = "base64";
                $result = $this->upBase64($config, $fieldName);
                break;
                /* 上传视频 */
            case 'uploadvideo':
                $fieldName = $CONFIG['videoFieldName'];
                $result = $this->upFile($fieldName);
                break;
                /* 上传文件 */
            case 'uploadfile':
                $fieldName = $CONFIG['fileFieldName'];
                $result = $this->upFile($fieldName);
                break;
                /* 列出图片 */
            case 'listimage':
                $allowFiles = $CONFIG['imageManagerAllowFiles'];
                $listSize = $CONFIG['imageManagerListSize'];
                $path = $CONFIG['imageManagerListPath'];
                $get = $_GET;
                $result = $this->fileList($allowFiles, $listSize, $get);
                break;
                /* 列出文件 */
            case 'listfile':
                $allowFiles = $CONFIG['fileManagerAllowFiles'];
                $listSize = $CONFIG['fileManagerListSize'];
                $path = $CONFIG['fileManagerListPath'];
                $get = $_GET;
                $result = $this->fileList($allowFiles, $listSize, $get);
                break;
                /* 抓取远程文件 */
            case 'catchimage':
                $config = array(
                    "pathFormat" => $CONFIG['catcherPathFormat'],
                    "maxSize" => $CONFIG['catcherMaxSize'],
                    "allowFiles" => $CONFIG['catcherAllowFiles'],
                    "oriName" => "remote.png",
                );
                $fieldName = $CONFIG['catcherFieldName'];
                /* 抓取远程图片 */
                $list = array();
                isset($_POST[$fieldName]) ? $source = $_POST[$fieldName] : $source = $_GET[$fieldName];

                foreach ($source as $imgUrl) {
                    $info = json_decode($this->saveRemote($config, $imgUrl), true);
                    array_push($list, array(
                        "state" => $info["state"],
                        "url" => $info["url"],
                        "size" => $info["size"],
                        "title" => htmlspecialchars($info["title"]),
                        "original" => htmlspecialchars($info["original"]),
                        "source" => htmlspecialchars($imgUrl),
                    ));
                }

                $result = json_encode(array(
                    'state' => count($list) ? 'SUCCESS' : 'ERROR',
                    'list' => $list,
                ));
                break;
            default:
                $result = json_encode(array(
                    'state' => '请求地址出错',
                ));
                break;
        }

        /* 输出结果 */
        if (isset($_GET["callback"])) {
            if (preg_match("/^[\w_]+$/", $_GET["callback"])) {
                echo htmlspecialchars($_GET["callback"]) . '(' . $result . ')';
            } else {
                echo json_encode(array(
                    'state' => 'callback参数不合法',
                ));
            }
        } else {
            echo $result;
            die();
        }
    }

    //上传文件
    private function upFile($fieldName)
    {
        $file = request()->file($fieldName);
        $reslut = model('Upload')->single_file_upload($file, "onepage");
        // dump($reslut);exit;
        $data = [];
        if ($reslut['code'] == 200) {
            $data = array(
                'state' => 'SUCCESS',
                'url' => $reslut['data']['image_path'],
                'title' => "image",
                'original' => $reslut['data']['image_path'],
                'type' => '',
                'size' =>  $reslut['data']['size'],
            );
        } else {
            $data = array(
                'state' => "上传错误",
            );
        }
        return json_encode($data);
        /*
        $reslut = model('api/BaiduBce')->admin_uedit_upload('content', $file);
        if ($reslut['code'] == 1) {

            $data = array(
                'state' => 'SUCCESS',
                'url' => $this->web_site . $reslut['data']['bosPath'],
                'title' => $reslut['data']['title'],
                'original' => $this->web_site . $reslut['data']['original'],
                'type' => $reslut['data']['type'],
                'size' => $reslut['data']['size'],
            );

        } else {
            $data = array(
                'state' => $reslut['data'],
            );

        }

        return json_encode($data);
        */
        /*
        $file = request()->file($fieldName);
        $info = $file->validate(['size' => 1567800000, 'ext' => 'jpg,gif,png,jpeg,mp4,mp3'])->move('uploads/uedit/');
        if ($info) { //上传成功
            $fname = '/uploads/uedit/' . str_replace('\\', '/', $info->getSaveName());
            $imgArr = explode(',', 'jpg,gif,png,jpeg,bmp,ttf,tif');
            $imgExt = strtolower($info->getExtension());
            $isImg = in_array($imgExt, $imgArr);

            // if($isImg){//如果是图片，开始处理
            //     $image = Image::open($file);
            //     $thumbnail = 1;
            //     $water = 1;

            //     //在这里你可以根据你需要，调用ThinkPHP5的图片处理方法了
            //     if($water == 1){//文字水印
            //         $image->text('df81.com','./public/font/4.ttf',180,'#ff0000')->save('.'.$fname);
            //     }
            //     if($water ==2 ){//图片水印
            //         $image->water('./public/img/df81.png',9,100)->save('.'.$fname);
            //     }
            //     if($thumbnail == 1){//生成缩略图
            //         $image->thumb(500,500,1)->save('.'.$fname);
            //     }
            // }

            $data = array(
                'state' => 'SUCCESS',
                'url' => $this->web_site . $fname,
                'title' => $info->getFilename(),
                'original' => $this->web_site . $info->getFilename(),
                'type' => '.' . $info->getExtension(),
                'size' => $info->getSize(),
            );
        } else {
            $data = array(
                'state' => $info->getError(),
            );
        }
        return json_encode($data);
        */
    }

    //列出图片
    private function fileList($allowFiles, $listSize, $get)
    {
        $dirname = './uploads/uedit/';
        $allowFiles = substr(str_replace(".", "|", join("", $allowFiles)), 1);

        /* 获取参数 */
        $size = isset($get['size']) ? htmlspecialchars($get['size']) : $listSize;
        $start = isset($get['start']) ? htmlspecialchars($get['start']) : 0;
        $end = $start + $size;

        /* 获取文件列表 */
        $path = $dirname;
        $files = $this->getFiles($path, $allowFiles);
        if (!count($files)) {
            return json_encode(array(
                "state" => "no match file",
                "list" => array(),
                "start" => $start,
                "total" => count($files),
            ));
        }

        /* 获取指定范围的列表 */
        $len = count($files);
        for ($i = min($end, $len) - 1, $list = array(); $i < $len && $i >= 0 && $i >= $start; $i--) {
            $list[] = $files[$i];
        }

        /* 返回数据 */
        $result = json_encode(array(
            "state" => "SUCCESS",
            "list" => $list,
            "start" => $start,
            "total" => count($files),
        ));

        return $result;
    }

    /*
     * 遍历获取目录下的指定类型的文件
     * @param $path
     * @param array $files
     * @return array
     */
    private function getFiles($path, $allowFiles, &$files = array())
    {
        if (!is_dir($path)) {
            return null;
        }

        if (substr($path, strlen($path) - 1) != '/') {
            $path .= '/';
        }

        $handle = opendir($path);

        while (false !== ($file = readdir($handle))) {
            if ($file != '.' && $file != '..') {
                $path2 = $path . $file;
                if (is_dir($path2)) {
                    $this->getFiles($path2, $allowFiles, $files);
                } else {
                    if (preg_match("/\.(" . $allowFiles . ")$/i", $file)) {
                        $files[] = array(
                            'url' => substr($path2, 1),
                            'mtime' => filemtime($path2),
                        );
                    }
                }
            }
        }

        return $files;
    }

    //抓取远程图片
    private function saveRemote($config, $fieldName)
    {
        $imgUrl = htmlspecialchars($fieldName);
        $imgUrl = str_replace("&amp;", "&", $imgUrl);

        //http开头验证
        if (strpos($imgUrl, "http") !== 0) {
            $data = array(
                'state' => '链接不是http链接',
            );
            return json_encode($data);
        }
        //获取请求头并检测死链
        $heads = get_headers($imgUrl);
        if (!(stristr($heads[0], "200") && stristr($heads[0], "OK"))) {
            $data = array(
                'state' => '链接不可用',
            );
            return json_encode($data);
        }
        //格式验证(扩展名验证和Content-Type验证)
        $fileType = strtolower(strrchr($imgUrl, '.'));
        if (!in_array($fileType, $config['allowFiles']) || stristr($heads['Content-Type'], "image")) {
            $data = array(
                'state' => '链接contentType不正确',
            );
            return json_encode($data);
        }

        //打开输出缓冲区并获取远程图片
        ob_start();
        $context = stream_context_create(
            array('http' => array(
                'follow_location' => false, // don't follow redirects
            ))
        );
        readfile($imgUrl, false, $context);
        $img = ob_get_contents();
        ob_end_clean();
        preg_match("/[\/]([^\/]*)[\.]?[^\.\/]*$/", $imgUrl, $m);

        $dirname = './public/uploads/remote/';
        $file['oriName'] = $m ? $m[1] : "";
        $file['filesize'] = strlen($img);
        $file['ext'] = strtolower(strrchr($config['oriName'], '.'));
        $file['name'] = uniqid() . $file['ext'];
        $file['fullName'] = $dirname . $file['name'];
        $fullName = $file['fullName'];

        //检查文件大小是否超出限制
        if ($file['filesize'] >= ($config["maxSize"])) {
            $data = array(
                'state' => '文件大小超出网站限制',
            );
            return json_encode($data);
        }

        //创建目录失败
        if (!file_exists($dirname) && !mkdir($dirname, 0777, true)) {
            $data = array(
                'state' => '目录创建失败',
            );
            return json_encode($data);
        } else if (!is_writeable($dirname)) {
            $data = array(
                'state' => '目录没有写权限',
            );
            return json_encode($data);
        }

        //移动文件
        if (!(file_put_contents($fullName, $img) && file_exists($fullName))) { //移动失败
            $data = array(
                'state' => '写入文件内容错误',
            );
            return json_encode($data);
        } else { //移动成功
            $data = array(
                'state' => 'SUCCESS',
                'url' => $this->web_site . substr($file['fullName'], 1),
                'title' => $file['name'],
                'original' => $this->web_site . $file['oriName'],
                'type' => $file['ext'],
                'size' => $file['filesize'],
            );
        }

        return json_encode($data);
    }

    /*
     * 处理base64编码的图片上传
     * 例如：涂鸦图片上传
     */
    private function upBase64($config, $fieldName)
    {
        $base64Data = $_POST[$fieldName];
        $img = base64_decode($base64Data);

        $dirname = './uploads/uedit/scrawl/';
        $file['filesize'] = strlen($img);
        $file['oriName'] = $config['oriName'];
        $file['ext'] = strtolower(strrchr($config['oriName'], '.'));
        $file['name'] = uniqid() . $file['ext'];
        $file['fullName'] = $dirname . $file['name'];
        $fullName = $file['fullName'];

        //检查文件大小是否超出限制
        if ($file['filesize'] >= ($config["maxSize"])) {
            $data = array(
                'state' => '文件大小超出网站限制',
            );
            return json_encode($data);
        }

        //创建目录失败
        if (!file_exists($dirname) && !mkdir($dirname, 0777, true)) {
            $data = array(
                'state' => '目录创建失败',
            );
            return json_encode($data);
        } else if (!is_writeable($dirname)) {
            $data = array(
                'state' => '目录没有写权限',
            );
            return json_encode($data);
        }

        //移动文件
        if (!(file_put_contents($fullName, $img) && file_exists($fullName))) { //移动失败
            $data = array(
                'state' => '写入文件内容错误',
            );
        } else { //移动成功
            $data = array(
                'state' => 'SUCCESS',
                'url' => $this->web_site . substr($file['fullName'], 1),
                'title' => $file['name'],
                'original' => $this->web_site . $file['oriName'],
                'type' => $file['ext'],
                'size' => $file['filesize'],
            );
        }

        return json_encode($data);
    }
}
