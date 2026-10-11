<?php

namespace app\api\controller;


use think\Controller;


class Upload extends Common
{
    public function img_upload()
    {
        if (request()->isPost()) {
            $files = request()->file('file');
            if (!$files) {
                return json(['code' => 201, 'msg' => '请上传文件', 'data' => null]);
            }
            $file_category = input('file_category', 'user_upload');
            $image_data = [];
            try {
                foreach ($files as $file) {
                    $reslut = model('admin/Upload')->single_file_upload($file, $file_category);
                    if ($reslut['code'] != 200) {
                        return json(['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => null]);
                    } else {
                        $image_data[] = $reslut['data']['http_image_path'];
                    }
                }
            } catch (\Throwable $e) {
                return json(['code' => 500, 'msg' => '上传失败，请检查存储目录权限', 'data' => null]);
            }
            return json(['code' => 200, 'msg' => '上传成功', 'data' => $image_data]);
        }
    }

    public function file_upload()
    {
        $file = request()->file('file');
        $file_category_name = input('file_category', 'user_upload');
        try {
            $reslut = model('admin/Upload')->single_file_upload($file, $file_category_name);
        } catch (\Throwable $e) {
            return json(['code' => 500, 'msg' => '上传失败，请检查存储目录权限', 'data' => null]);
        }
        return json(['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => $reslut['data']]);
    }
}
