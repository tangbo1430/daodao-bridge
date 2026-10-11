INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT 0,1,'im_callback_root',CONVERT(0x494DE59B9EE8B083 USING utf8mb4),'layui-icon layui-icon-log','','_self',1,95
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='im_callback_root');

SET @im_callback_root = (SELECT id FROM `yy_system_menu` WHERE `limit_sign`='im_callback_root' LIMIT 1);

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @im_callback_root,2,'im_callback_log',CONVERT(0xE59B9EE8B083E697A5E5BF97 USING utf8mb4),'','page/imcallback/callback_log.html','_self',1,10
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='im_callback_log');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @im_callback_root,2,'im_callback_state',CONVERT(0x494DE78AB6E68081 USING utf8mb4),'','page/imcallback/im_state.html','_self',1,9
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='im_callback_state');

SET @im_callback_log = (SELECT id FROM `yy_system_menu` WHERE `limit_sign`='im_callback_log' LIMIT 1);
SET @im_callback_state = (SELECT id FROM `yy_system_menu` WHERE `limit_sign`='im_callback_state' LIMIT 1);

UPDATE `yy_admin`
SET `system_menu_id_list` = CONCAT_WS(',', NULLIF(`system_menu_id_list`, ''), @im_callback_log, @im_callback_state)
WHERE `aid` = 32
  AND FIND_IN_SET(@im_callback_log, `system_menu_id_list`) = 0;
