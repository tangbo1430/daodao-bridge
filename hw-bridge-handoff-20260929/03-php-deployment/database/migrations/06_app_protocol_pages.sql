SET NAMES utf8mb4;

SET @system_type = 3;

INSERT INTO `yy_page`
  (`title`,`url`,`content`,`is_delete`,`delete_time`,`add_time`,`update_time`,`system_type`)
SELECT '企才汇用户协议','app_user_agreement',
       '<h2>企才汇用户协议</h2><p>请在后台「网站管理 - 协议与隐私」中编辑用户协议内容。</p>',
       1,0,UNIX_TIMESTAMP(),UNIX_TIMESTAMP(),@system_type
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_page` WHERE `url`='app_user_agreement' AND `system_type`=@system_type
);

INSERT INTO `yy_page`
  (`title`,`url`,`content`,`is_delete`,`delete_time`,`add_time`,`update_time`,`system_type`)
SELECT '企才汇隐私政策','app_privacy_policy',
       '<h2>企才汇隐私政策</h2><p>请在后台「网站管理 - 协议与隐私」中编辑隐私政策内容。</p>',
       1,0,UNIX_TIMESTAMP(),UNIX_TIMESTAMP(),@system_type
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_page` WHERE `url`='app_privacy_policy' AND `system_type`=@system_type
);

UPDATE `yy_page`
SET `is_delete`=1, `delete_time`=0, `update_time`=UNIX_TIMESTAMP()
WHERE `url` IN ('app_user_agreement','app_privacy_policy')
  AND `system_type`=@system_type
  AND `is_delete`<>1;

SET @system_root = (
  SELECT `id` FROM `yy_system_menu`
  WHERE `pid`=0 AND `type`=1 AND `title`='系统管理'
  ORDER BY `id` ASC LIMIT 1
);

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'website_group','网站管理','layui-icon layui-icon-website','','_self',1,86
WHERE @system_root IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='website_group');

SET @website_group = (
  SELECT `id` FROM `yy_system_menu`
  WHERE `limit_sign`='website_group'
     OR (`pid`=@system_root AND `title`='网站管理')
  ORDER BY `id` ASC LIMIT 1
);

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @website_group,1,'website_protocol_manage','协议与隐私','','page/website/protocol.html','_self',1,5
WHERE @website_group IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='website_protocol_manage');

UPDATE `yy_system_menu`
SET `title`='协议与隐私', `href`='page/website/protocol.html', `status`=1, `sort`=5
WHERE `limit_sign`='website_protocol_manage';

SET @protocol_menu = (
  SELECT `id` FROM `yy_system_menu`
  WHERE `limit_sign`='website_protocol_manage'
  ORDER BY `id` ASC LIMIT 1
);

UPDATE `yy_admin`
SET `system_menu_id_list` = CONCAT_WS(',', NULLIF(`system_menu_id_list`, ''), @website_group)
WHERE `aid`=32
  AND @website_group IS NOT NULL
  AND FIND_IN_SET(@website_group, IFNULL(`system_menu_id_list`, '')) = 0;

UPDATE `yy_admin`
SET `system_menu_id_list` = CONCAT_WS(',', NULLIF(`system_menu_id_list`, ''), @protocol_menu)
WHERE `aid`=32
  AND @protocol_menu IS NOT NULL
  AND FIND_IN_SET(@protocol_menu, IFNULL(`system_menu_id_list`, '')) = 0;
