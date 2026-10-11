SET NAMES utf8mb4;

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT 0,1,'system_root','系统管理','','','_self',1,0
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_system_menu` WHERE `pid`=0 AND `title`='系统管理'
);
SET @system_root = (
  SELECT `id` FROM `yy_system_menu` WHERE `pid`=0 AND `title`='系统管理' ORDER BY `id` LIMIT 1
);

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,2,'member_group','会员管理','layui-icon layui-icon-username','','_self',1,99
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='会员管理'
);
SET @member_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='会员管理' ORDER BY `id` LIMIT 1
);
INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @member_group,2,'user_list','会员列表','','page/user/user_list.html','_self',1,0
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `href`='page/user/user_list.html');
INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @member_group,2,'user_real_name_list','实名认证列表','','page/user/user_real_name_list.html','',1,0
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `href`='page/user/user_real_name_list.html');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'config_group','系统配置','layui-icon layui-icon-set-fill','','_self',1,0
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='系统配置'
);
SET @config_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='系统配置' ORDER BY `id` LIMIT 1
);
INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @config_group,1,'config_list','基础配置',NULL,'page/config/config_list.html','_self',1,0
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `href`='page/config/config_list.html');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'admin_group','管理员管理','layui-icon layui-icon-friends','','_self',1,0
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='管理员管理'
);
SET @admin_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='管理员管理' ORDER BY `id` LIMIT 1
);
INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @admin_group,1,'admin_list','管理员列表','','page/admin/admin_list.html','_self',1,0
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `href`='page/admin/admin_list.html');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'website_group','网站管理','layui-icon layui-icon-website','','_self',1,86
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='网站管理'
);
SET @website_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='网站管理' ORDER BY `id` LIMIT 1
);
INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @website_group,1,'website_page_list','单页管理',NULL,'page/website/page_list.html','_self',1,0
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `href`='page/website/page_list.html');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'group_group','群组管理','','','',1,0
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='群组管理'
);
SET @group_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='群组管理' ORDER BY `id` LIMIT 1
);
INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @group_group,1,'group_list','群组列表','','page/group/group_list.html','',1,0
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `href`='page/group/group_list.html');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'invite_group','邀请码管理','','','',1,0
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='邀请码管理'
);
SET @invite_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `pid`=@system_root AND `title`='邀请码管理' ORDER BY `id` LIMIT 1
);
INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @invite_group,1,'invite_list','邀请码列表','','page/invitecode/list.html','',1,0
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `href`='page/invitecode/list.html');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'im_operations_group','IM运维','layui-icon layui-icon-console','','_self',1,85
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='im_operations_group');
SET @im_operations_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `limit_sign`='im_operations_group' ORDER BY `id` LIMIT 1
);
UPDATE `yy_system_menu`
SET `pid`=@im_operations_group
WHERE `limit_sign` IN ('im_callback_overview','im_callback_anomaly','im_callback_log','im_callback_state');
DELETE FROM `yy_system_menu`
WHERE `limit_sign`='im_callback_root' AND `id`<>@im_operations_group;

UPDATE `yy_admin`
SET `system_menu_id_list` = (
  SELECT GROUP_CONCAT(`id` ORDER BY `id` SEPARATOR ',')
  FROM `yy_system_menu`
  WHERE `status`=1 AND `type` IN (1,2)
)
WHERE `aid`=32;
