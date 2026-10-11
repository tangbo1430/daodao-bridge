<?php

namespace app\admin\controller;

use think\Controller;
use think\Request;

class Upload extends Common
{
    public function file_upload()
    {
        $file = request()->file('file');
        $file_category_name = input('file_category', 'all');
        $reslut = model('Upload')->single_file_upload($file, $file_category_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
}
