<?php


namespace app\common\libs;

class ExportLib
{
    public static function zipData($data = [], $tileArr = [], $filename='caigou.csv',$mark='')
    {
        if($mark == ''){
            $mark = 'product-' . date('YmdHis') ;
        }
        $sqlCount = count($data);
        // 每次只从数据库取 50000 条以防变量缓存太大
        // 每隔 50000 行，刷新一下输出buffer，不要太大，也不要太小
        $sqlLimit = 50000;
        // buffer计数器
        $cnt = 0;
        //总数组个数
        $total = ceil($sqlCount/$sqlLimit);
        $dataArr = array_chunk($data,$sqlLimit);
        $path = './static/temp/down/';

        set_time_limit(0);
        // 输出Excel文件头，可把user.csv换成你要的文件名
        header('Expires: 0');
        header('Cache-control: private');
        header('Cache-Control: must-revalidate, post-check=0, pre-check=0');
        header('Content-Description: File Transfer');
        header('Content-Type: text/csv;charset=utf-8');
        header('Content-Encoding: UTF-8');
        header('Content-Disposition: attachment;filename="' . $filename . '"');
        header('Cache-Control: max-age=0');

        //（默认30秒）set_time_limit(0)不对PHP执行时间做限制。
        $fileNameArr = array();
        // 逐行取出数据，不浪费内存
        for ($i = 0; $i < $total; $i++) {
            $file = $path . $mark . '_' . $i . '.csv';
            $fp = fopen($file , 'w'); //生成临时文件
            fwrite($fp, chr(0xEF).chr(0xBB).chr(0xBF)); // 添加 BOM
            $fileNameArr[] = $file;
            // 将数据通过fputcsv写到文件句柄
            fputcsv($fp, $tileArr);
            foreach ($dataArr[$i] as $a) {
                $cnt++;
                if ($sqlLimit == $cnt) {
                    //刷新一下输出buffer，防止由于数据过多造成问题
                    ob_flush();
                    flush();
                    $cnt = 0;
                }
                fputcsv($fp, $a);
            }
            fclose($fp);  //每生成一个文件关闭
        }
        //进行多个文件压缩
        $zip = new \ZipArchive();
        $filename = $path . $mark . ".zip";
        $zip->open($filename, \ZipArchive::CREATE);   //打开压缩包
        foreach ($fileNameArr as $file) {
            $zip->addFile($file, basename($file));   //向压缩包中添加文件
        }
        $zip->close();  //关闭压缩包
        foreach ($fileNameArr as $file) {
            unlink($file); //删除csv临时文件
        }
        //输出压缩文件提供下载
        header("Cache-Control: max-age=0");
        header("Content-Description: File Transfer");
        header('Content-disposition: attachment; filename=' . basename($filename)); // 文件名
        header("Content-Type: application/zip"); // zip格式的
        header("Content-Transfer-Encoding: binary"); //
        header('Content-Length: ' . filesize($filename)); //
        @readfile($filename);//输出文件;
        unlink($filename); //删除压缩包临时文件
    }

}