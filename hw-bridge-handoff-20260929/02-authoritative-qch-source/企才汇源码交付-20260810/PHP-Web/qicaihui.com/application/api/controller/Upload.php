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
                return ajaxReturn(0, '请上传文件');
            }
            $file_category = input('file_category', 'user_upload');
            $image_data = [];
            foreach ($files as $file) {
                $reslut = model('admin/Upload')->single_file_upload($file, $file_category);
                if ($reslut['code'] != 200) {
                    return ajaxReturn(200, $reslut['msg'], null);
                } else {
                    $image_data[] = $reslut['data']['http_image_path'];
                }
            }
            return ajaxReturn(200, '上传成功', $image_data);
        }
    }

    public function file_upload()
    {
        $file = request()->file('file');
        $file_category_name = input('file_category', 'user_upload');
        $reslut = model('admin/Upload')->single_file_upload($file, $file_category_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
}
