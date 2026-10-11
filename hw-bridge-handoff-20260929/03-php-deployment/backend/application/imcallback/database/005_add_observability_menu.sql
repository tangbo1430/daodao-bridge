SET @im_callback_root = (SELECT id FROM `yy_system_menu` WHERE `limit_sign`='im_callback_root' LIMIT 1);

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @im_callback_root,2,'im_callback_overview',CONVERT(0xE8BF90E8A18CE6A682E8A788 USING utf8mb4),'','page/imcallback/overview.html','_self',1,12
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='im_callback_overview');

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @im_callback_root,2,'im_callback_anomaly',CONVERT(0xE5BC82E5B8B8E4B8ADE5BF83 USING utf8mb4),'','page/imcallback/anomalies.html','_self',1,11
WHERE NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='im_callback_anomaly');

SET @im_callback_overview = (SELECT id FROM `yy_system_menu` WHERE `limit_sign`='im_callback_overview' LIMIT 1);
SET @im_callback_anomaly = (SELECT id FROM `yy_system_menu` WHERE `limit_sign`='im_callback_anomaly' LIMIT 1);

UPDATE `yy_admin`
SET `system_menu_id_list` = CONCAT_WS(',', NULLIF(`system_menu_id_list`, ''), @im_callback_overview, @im_callback_anomaly)
WHERE `aid` = 32 AND FIND_IN_SET(@im_callback_overview, `system_menu_id_list`) = 0;
