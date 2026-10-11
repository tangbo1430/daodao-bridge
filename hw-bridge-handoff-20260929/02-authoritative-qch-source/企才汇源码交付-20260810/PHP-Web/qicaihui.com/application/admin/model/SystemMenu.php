<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class SystemMenu extends Model
{

    // 获取初始化数据
    public function getSystemInit($aid = 0)
    {
        $homeInfo = [
            'title' => '首页',
            'href' => 'page/welcome.html',
        ];
        $logoInfo = [
            'title' => '后台 管理',
            'image' => 'images/logo.png',
            'href' => 'javascript:;',
        ];
        $menuInfo = $this->getMenuList($aid);
        $systemInit = [
            'homeInfo' => $homeInfo,
            'logoInfo' => $logoInfo,
            'menuInfo' => $menuInfo,
        ];
        return $systemInit;
    }

    // 获取菜单列表
    private function getMenuList($aid)
    {
        $menuList = Db::name('system_menu')
            ->field('id,pid,title,icon,href,target')
            ->where('status', 1)
            ->where('type', 'in', [1, 2])
            ->order('sort', 'desc')
            ->select();
        //超级管理员默认有所有权限
        if ($aid != 10) {
            $admin_info  = db::name('admin')->find($aid);
            if (empty($admin_info)) {
                return ['code' => 201, 'msg' => '管理员信息不存在', 'data' => null];
            }
            $system_menu_id_list_data = explode(',', $admin_info['system_menu_id_list']);
            foreach ($menuList as $k => &$v) {
                if ($v['pid'] != 0) {
                    if (!in_array($v['id'], $system_menu_id_list_data)) {
                        unset($menuList[$k]);
                    }
                }
            }
        }


        $menuList = $this->buildMenuChild(0, $menuList);
        return $menuList;
    }

    //递归获取子菜单
    private function buildMenuChild($pid, $menuList)
    {
        $treeList = [];
        foreach ($menuList as &$v) {

            if ($pid == $v['pid']) {
                $node = $v;
                $child = $this->buildMenuChild($v['id'], $menuList);
                if (!empty($child)) {
                    $node['child'] = $child;
                }
                // todo 后续此处加上用户的权限判断
                $treeList[] = $node;
            }
        }
        return $treeList;
    }
    public function get_all_system_menu_list($aid)
    {
        $menuList = Db::name('system_menu')
            ->field('id,pid,title,icon,href,target')
            ->where('status', 1)
            ->where('type', 'in', [1, 2])
            ->order('sort', 'desc')
            ->select();
        $admin_info  = db::name('admin')->find($aid);
        if (empty($admin_info)) {
            return ['code' => 201, 'msg' => '管理员信息不存在', 'data' => null];
        }

        $system_menu_id_list_data = explode(',', $admin_info['system_menu_id_list']);
        $menu_data = [];
        foreach ($menuList as $k => &$v) {
            // $v['is_show'] = 1; //有权限
            // if (!in_array($v['id'], $system_menu_id_list_data) && $aid != 1) {
            //     $v['is_show'] = 2; //无权限
            // }
            $checked = false; //有权限
            if (in_array($v['id'], $system_menu_id_list_data) ) {
            
                $checked = true; //无权限
            }
            
            $v['checked'] = $checked;
            $v['spread'] = true;
            $menu_data[$v['id']] = $v;
        }
        foreach ($menu_data as $key => $val) {
            if ($val['checked'] === true && $val['pid'] != 0) {
                $menu_data[$val['pid']]['checked'] = false;
            }
        }
        // dump($menu_data);die;
        // $menuList = $this->buildMenuChild(0, $menuList);
        $menuList = $this->_getManySonData($menu_data); 
        return ['code' => 200, 'msg' => '获取成功', 'data' => $menuList];
    }
    
    //树形创建
    private function _getManySonData($data, $sonName = 'children')
    {
        $tree = [];
        //第一步，将分类id作为数组key,并创建children单元
        foreach ($data as $key => $val) {
            $tree[$val['id']] = $val;
            $tree[$val['id']][$sonName] = [];
        }
        //第二步，利用引用，将每个分类添加到父类children数组中，这样一次遍历即可形成树形结构。
        foreach ($tree as $key => $val) {
            if ($val['pid'] != 0) {
                $tree[$val['pid']][$sonName][] = &$tree[$key];
                if ($tree[$key][$sonName] == null) {
                    unset($tree[$key][$sonName]); //如果children为空，则删除该children元素（可选）
                }
            }
        }
        //第三步，删除无用的非根节点数据
        foreach ($tree as $key => $val) {
            if (isset($val['pid']) && $val['pid'] != 0) {
                unset($tree[$key]);
            }
        }
        $result = [];
        foreach ($tree as $key => $val) {
            $result[] = $val;
        }
        return $result;
    }
}
