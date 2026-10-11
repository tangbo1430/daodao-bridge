<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class UserZone extends Model
{

    //获取社区信息
    public function user_zone_data($uid,$nick_name,$content,$show_status,$page,$limit)
    {
        $map = [];
        if(!empty($uid)){
            $map[] = ['a.uid','=',$uid];
        }
        if(!empty($nick_name)){
            $map[] = ['b.nick_name','like','%'.$nick_name.'%'];
        }
        if(!empty($content)){
            $map[] = ['a.content','like','%'.$content.'%'];
        }
        if(!empty($show_status)){
            $map[] = ['a.show_status','=',$show_status];
        }
        $list = db::name('user_zone')->alias('a')->join('yy_user b','a.uid = b.uid')->where($map)->field('a.*,b.nick_name')->page($page,$limit)->order('zid desc')->select();
        foreach($list as $k=>&$v){
            $list[$k]['user_nick_name'] = $v['uid'].'-'.$v['nick_name'];
            $image_data = explode(',',$v['images']);
            $v['image_list'] = [];
            foreach($image_data as $m=>$n){
                $v['image_list'][] = localpath_to_netpath($n);
            }
        }
        $data = [];
        $data['count'] = db::name('user_zone')->alias('a')->join('yy_user b','a.uid = b.uid')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200 ,'msg' => '获取成功' ,'data'=>$data];
    }
    
    public function get_zone_info($zid)
    {
        $map = [];
        $map[] = ['zid', '=', $zid];
        $zone_info = db::name('user_zone')->where($map)->find();
        if (empty($zone_info)) {
            return ['code' => 201, 'msg' => '说说不存在', 'data' => null];
        }
        $user_info = db::name('user')->find($zone_info['uid']);
        $zone_info['user_name'] = $user_info['user_name'];
        $zone_info['nick_name'] = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');

      
        $zone_info['image_list'] = '';
        if (!empty($zone_info['images'])) {
            $image_data = explode(',', $zone_info['images']);
            foreach ($image_data as $m => &$n) {
               $n = localpath_to_netpath($n);
            }
            $zone_info['image_list']=join(',',$image_data);

        }
       
        return ['code' => 200, 'msg' => 'success', 'data' => $zone_info];
    }

    
    //审核发布说说
    public function edit_user_zone($data)
    {
        $map = [];
        $map[] = ['zid','=',$data['zid']];
        $zone_info = db::name('user_zone')->where($map)->find();
        if(empty($zone_info)){
            return ['code'=>201,'msg'=>'说说不存在','data'=>null];
        }
        if(!in_array($data['show_status'],[1,2,3])){
            return ['code'=>201, 'msg'=> '审核状态参数错误', 'data' => null];
        }
        $update_data =[];
        $update_data['zid'] = $data['zid'];
        $update_data['show_status'] = $data['show_status'];
        $update_data['update_time'] = time();
        $result  = db::name('user_zone')->update($update_data);
        if($result){
            return ['code' =>200,'msg'=> '操作成功', 'data' => null];
        }else {
            return ['code' => 201, 'msg' => '操作失败', 'data' => null];
        }
        
        
    }
    
    
}
