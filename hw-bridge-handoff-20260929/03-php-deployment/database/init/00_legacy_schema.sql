-- MySQL dump 10.13  Distrib 5.7.44, for Linux (x86_64)
--
-- Host: localhost    Database: jishitongxun_com
-- ------------------------------------------------------
-- Server version	5.7.44-log

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `yy_admin`
--

DROP TABLE IF EXISTS `yy_admin`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_admin` (
  `aid` int(11) NOT NULL AUTO_INCREMENT,
  `user_name` varchar(255) DEFAULT '',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `password` varchar(255) DEFAULT '',
  `login_token` varchar(255) DEFAULT '',
  `system_menu_id_list` text COMMENT '有权限菜单  0所有权限',
  `is_delete` tinyint(1) DEFAULT '1' COMMENT '1正常2删除',
  `delete_time` int(11) DEFAULT '0' COMMENT '删除时间',
  `add_time` int(11) DEFAULT '0',
  `ip` varchar(255) DEFAULT NULL COMMENT '当前ip',
  `province` varchar(255) DEFAULT NULL COMMENT '省份',
  `city` varchar(255) DEFAULT NULL COMMENT '市',
  `update_time` int(11) DEFAULT '0',
  `token_validity_time` int(11) DEFAULT '0',
  `error_num` int(11) DEFAULT '0',
  `status` tinyint(1) DEFAULT '1',
  `system_type` tinyint(1) DEFAULT '1' COMMENT '1 引才荟  2 企才荟',
  PRIMARY KEY (`aid`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=32 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='管理员表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_admin`
--

LOCK TABLES `yy_admin` WRITE;
/*!40000 ALTER TABLE `yy_admin` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_admin` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_config`
--

DROP TABLE IF EXISTS `yy_config`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_config` (
  `cid` int(11) NOT NULL AUTO_INCREMENT,
  `type` tinyint(1) DEFAULT '1' COMMENT '1基础配置2高级配置',
  `key_title` varchar(255) DEFAULT '',
  `key_name` varchar(255) NOT NULL DEFAULT '',
  `key_value` text,
  `key_desc` varchar(255) DEFAULT '',
  `sort` int(11) DEFAULT '0' COMMENT '0',
  `update_time` int(11) DEFAULT NULL,
  `is_delete` tinyint(1) DEFAULT '1',
  PRIMARY KEY (`cid`,`key_name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=149 DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='系统配置表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_config`
--

LOCK TABLES `yy_config` WRITE;
/*!40000 ALTER TABLE `yy_config` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_config` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_envelope_type`
--

DROP TABLE IF EXISTS `yy_envelope_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_envelope_type` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `type_name` varchar(255) DEFAULT NULL COMMENT '类型名称  1房间红包  2全服红包',
  `max_price` int(11) DEFAULT '0' COMMENT '最大金额',
  `min_price` int(11) DEFAULT '0' COMMENT '最小金额',
  `max_num` int(11) DEFAULT '0' COMMENT '最大数量',
  `min_num` int(11) DEFAULT '0' COMMENT '最小数量',
  `add_time` int(11) DEFAULT '0',
  `update_time` int(11) DEFAULT '0',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='红包类型表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_envelope_type`
--

LOCK TABLES `yy_envelope_type` WRITE;
/*!40000 ALTER TABLE `yy_envelope_type` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_envelope_type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_group_file`
--

DROP TABLE IF EXISTS `yy_group_file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_group_file` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `file_name` varchar(255) DEFAULT '1' COMMENT '文件名称',
  `file_url` varchar(255) DEFAULT NULL COMMENT '文件链接',
  `createtime` int(11) DEFAULT NULL,
  `is_delete` int(1) NOT NULL DEFAULT '1' COMMENT '1正常 2删除',
  `delete_time` int(11) DEFAULT NULL,
  `im_g_id` varchar(50) NOT NULL DEFAULT '0' COMMENT 'im群组id',
  `group_id` int(11) DEFAULT NULL COMMENT '群id',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=99838 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='群文件表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_group_file`
--

LOCK TABLES `yy_group_file` WRITE;
/*!40000 ALTER TABLE `yy_group_file` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_group_file` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_group_v`
--

DROP TABLE IF EXISTS `yy_group_v`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_group_v` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) DEFAULT NULL COMMENT '族长id',
  `group_name` varchar(255) DEFAULT '1' COMMENT '群组名称',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  `is_delete` int(1) NOT NULL DEFAULT '1' COMMENT '1正常 2删除',
  `delete_time` int(11) DEFAULT NULL,
  `im_g_id` varchar(50) NOT NULL DEFAULT '0' COMMENT 'im群组id',
  `open_protect` tinyint(1) NOT NULL DEFAULT '2' COMMENT '群保护 1 开启  2 未开启',
  `system_type` tinyint(1) NOT NULL DEFAULT '1' COMMENT '1 引才荟  2 企才荟 3第三套 4第四套',
  `client_request_id` varchar(64) DEFAULT NULL COMMENT '客户端建群幂等标识',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_group_create_request` (`uid`,`client_request_id`)
) ENGINE=InnoDB AUTO_INCREMENT=99987 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='家族表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_group_v`
--

LOCK TABLES `yy_group_v` WRITE;
/*!40000 ALTER TABLE `yy_group_v` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_group_v` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_invitecode`
--

DROP TABLE IF EXISTS `yy_invitecode`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_invitecode` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `name` varchar(255) NOT NULL COMMENT '邀请码',
  `status` tinyint(1) DEFAULT '0' COMMENT '状态0正常1使用',
  `createtime` int(11) NOT NULL DEFAULT '0' COMMENT '创建时间',
  `user_id` varchar(255) DEFAULT NULL COMMENT '绑定的用户id',
  `system_type` tinyint(2) NOT NULL DEFAULT '3' COMMENT '3 第三后台  4 第四后台',
  `invite_num` int(11) NOT NULL DEFAULT '0' COMMENT '邀请人数',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=MyISAM AUTO_INCREMENT=145 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='公告表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_invitecode`
--

LOCK TABLES `yy_invitecode` WRITE;
/*!40000 ALTER TABLE `yy_invitecode` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_invitecode` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_mysql_back`
--

DROP TABLE IF EXISTS `yy_mysql_back`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_mysql_back` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `file_name` varchar(255) NOT NULL DEFAULT '' COMMENT '文件名称',
  `add_time` int(11) NOT NULL DEFAULT '0' COMMENT '上传时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=56 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_mysql_back`
--

LOCK TABLES `yy_mysql_back` WRITE;
/*!40000 ALTER TABLE `yy_mysql_back` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_mysql_back` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_operation`
--

DROP TABLE IF EXISTS `yy_operation`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_operation` (
  `op_id` int(11) NOT NULL AUTO_INCREMENT,
  `type` tinyint(1) NOT NULL DEFAULT '0' COMMENT '1管理员2用户3厅主',
  `id` int(11) NOT NULL DEFAULT '0' COMMENT 'type 对应用户id',
  `name` varchar(255) NOT NULL DEFAULT '' COMMENT 'type 对应用户名',
  `url` varchar(255) NOT NULL COMMENT '访问url',
  `param` longtext NOT NULL COMMENT '携带参数',
  `header` text COMMENT '请求头',
  `user_agent` text COMMENT '浏览器头',
  `country` varchar(255) DEFAULT '',
  `province` varchar(255) DEFAULT '',
  `city` varchar(255) DEFAULT '',
  `add_ip` varchar(255) DEFAULT '',
  `add_time` int(11) NOT NULL,
  PRIMARY KEY (`op_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=23543 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='系统操作日志表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_operation`
--

LOCK TABLES `yy_operation` WRITE;
/*!40000 ALTER TABLE `yy_operation` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_operation` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_page`
--

DROP TABLE IF EXISTS `yy_page`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_page` (
  `aid` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) DEFAULT NULL,
  `url` varchar(255) DEFAULT NULL,
  `content` longtext,
  `is_delete` tinyint(1) DEFAULT '1',
  `delete_time` int(11) DEFAULT '0',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  `system_type` tinyint(1) DEFAULT '1' COMMENT '1 引才荟  2 企才荟',
  PRIMARY KEY (`aid`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='文章单页';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_page`
--

LOCK TABLES `yy_page` WRITE;
/*!40000 ALTER TABLE `yy_page` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_page` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_red_envelope`
--

DROP TABLE IF EXISTS `yy_red_envelope`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_red_envelope` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `type` tinyint(1) DEFAULT '1' COMMENT '红包类型 1房间 2全服 3 家族',
  `uid` int(11) DEFAULT '0',
  `rid` int(11) DEFAULT '0' COMMENT '房间id',
  `group_id` int(11) DEFAULT '0' COMMENT '家族id',
  `give_type` tinyint(1) DEFAULT '1' COMMENT '发送类型 1后台 2用户',
  `title` varchar(255) DEFAULT NULL,
  `base64_title` varchar(255) DEFAULT NULL,
  `money` decimal(12,3) DEFAULT '0.000' COMMENT '红包金额',
  `price` decimal(12,3) DEFAULT '0.000' COMMENT '实际金额',
  `num` int(11) DEFAULT '0' COMMENT '红包数量',
  `open_num` int(11) DEFAULT '0' COMMENT '已开数量',
  `surplus_price` decimal(12,3) DEFAULT '0.000' COMMENT '剩余金额',
  `interval` int(11) DEFAULT NULL COMMENT '间隔时间 单位：分钟',
  `start_time` int(11) DEFAULT '0' COMMENT '多少分钟后开抢时间',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  `end_time` int(11) DEFAULT NULL,
  `is_delete` tinyint(1) DEFAULT '1' COMMENT '是否删除 1否2是',
  `is_finish` tinyint(1) DEFAULT '2' COMMENT '是否完成 1是2否',
  `is_stop` tinyint(1) DEFAULT '2' COMMENT '是否停止 1是2否',
  `stop_time` int(11) DEFAULT NULL,
  `receive_uid` int(11) DEFAULT '0' COMMENT '收礼物用户id',
  `is_special` tinyint(1) DEFAULT '2' COMMENT '1 专属  2 普通',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `tid` (`type`) USING BTREE,
  KEY `is_finish` (`is_stop`,`is_finish`,`is_delete`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='发红包表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_red_envelope`
--

LOCK TABLES `yy_red_envelope` WRITE;
/*!40000 ALTER TABLE `yy_red_envelope` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_red_envelope` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_report_type`
--

DROP TABLE IF EXISTS `yy_report_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_report_type` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `type_name` varchar(255) DEFAULT '' COMMENT '举报类型名称',
  `is_delete` tinyint(1) DEFAULT '1' COMMENT '1 正常  2删除',
  `delete_time` int(11) DEFAULT NULL,
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='举报类型表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_report_type`
--

LOCK TABLES `yy_report_type` WRITE;
/*!40000 ALTER TABLE `yy_report_type` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_report_type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_sign_config`
--

DROP TABLE IF EXISTS `yy_sign_config`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_sign_config` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `day_num` int(11) NOT NULL DEFAULT '1' COMMENT '天数',
  `money` decimal(3,2) NOT NULL DEFAULT '0.00' COMMENT '赠送金币',
  `add_time` int(11) NOT NULL DEFAULT '0',
  `update_time` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='签到配置';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_sign_config`
--

LOCK TABLES `yy_sign_config` WRITE;
/*!40000 ALTER TABLE `yy_sign_config` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_sign_config` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_sign_user`
--

DROP TABLE IF EXISTS `yy_sign_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_sign_user` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) NOT NULL DEFAULT '0' COMMENT '用户id',
  `day_num` int(11) NOT NULL DEFAULT '0' COMMENT '天数',
  `money` decimal(3,2) NOT NULL DEFAULT '0.00' COMMENT 'u币',
  `day_time` varchar(8) NOT NULL DEFAULT '' COMMENT '日期',
  `add_time` int(11) NOT NULL DEFAULT '0' COMMENT '添加时间',
  `update_time` int(11) NOT NULL DEFAULT '0' COMMENT '更新时间',
  `system_type` tinyint(1) NOT NULL DEFAULT '1' COMMENT '1 引才荟  2 企才荟',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=2625 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='奖励记录';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_sign_user`
--

LOCK TABLES `yy_sign_user` WRITE;
/*!40000 ALTER TABLE `yy_sign_user` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_sign_user` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_sign_user_day`
--

DROP TABLE IF EXISTS `yy_sign_user_day`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_sign_user_day` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) NOT NULL DEFAULT '0' COMMENT '用户id',
  `day_time` varchar(8) NOT NULL DEFAULT '' COMMENT '日期',
  `add_time` int(11) DEFAULT '0',
  `udpate_time` int(11) DEFAULT '0',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='用户签到日期';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_sign_user_day`
--

LOCK TABLES `yy_sign_user_day` WRITE;
/*!40000 ALTER TABLE `yy_sign_user_day` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_sign_user_day` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_sms`
--

DROP TABLE IF EXISTS `yy_sms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_sms` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `mobile` varchar(20) DEFAULT '' COMMENT '短信内容',
  `code` varchar(20) DEFAULT NULL,
  `content` varchar(255) DEFAULT NULL,
  `remarks` text,
  `status` tinyint(1) DEFAULT '1' COMMENT '1无效2待使用3已使用',
  `error_num` tinyint(1) DEFAULT '0' COMMENT '验证码错误次数',
  `over_time` int(11) DEFAULT '0' COMMENT '过期时间',
  `add_time` int(11) DEFAULT '0',
  `update_time` int(11) DEFAULT '0',
  `module` tinyint(1) DEFAULT '1' COMMENT '1 app  2 后台',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `mobile` (`mobile`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=660 DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='发送短信记录';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_sms`
--

LOCK TABLES `yy_sms` WRITE;
/*!40000 ALTER TABLE `yy_sms` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_sms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_system_menu`
--

DROP TABLE IF EXISTS `yy_system_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_system_menu` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `pid` int(11) DEFAULT '0',
  `type` tinyint(1) DEFAULT '0' COMMENT '1目录2菜单3功能',
  `limit_sign` varchar(255) DEFAULT '' COMMENT '权限标识',
  `title` varchar(255) DEFAULT '' COMMENT '名称',
  `icon` varchar(255) DEFAULT '' COMMENT '图标',
  `href` varchar(255) DEFAULT '' COMMENT '地址',
  `target` varchar(255) DEFAULT '' COMMENT '说明',
  `status` tinyint(1) DEFAULT '1' COMMENT '1启用2禁用',
  `sort` int(11) DEFAULT '0' COMMENT '排序',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=26 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='菜单表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_system_menu`
--

LOCK TABLES `yy_system_menu` WRITE;
/*!40000 ALTER TABLE `yy_system_menu` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_system_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user`
--

DROP TABLE IF EXISTS `yy_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user` (
  `uid` int(11) NOT NULL AUTO_INCREMENT,
  `pid` int(11) DEFAULT '0' COMMENT '上级id',
  `user_name` varchar(50) DEFAULT '' COMMENT '用户名手机号',
  `password` varchar(50) DEFAULT '' COMMENT '登录密码',
  `nick_name` varchar(30) DEFAULT '' COMMENT '用户昵称',
  `base64_nick_name` varchar(255) DEFAULT '' COMMENT 'base64 用户昵称',
  `head_pic` varchar(255) DEFAULT '' COMMENT '用户头像',
  `sex` tinyint(1) DEFAULT '1' COMMENT '0未知1男2女',
  `birthday` varchar(50) DEFAULT NULL COMMENT '出生日期',
  `constellation` varchar(255) DEFAULT '' COMMENT '星座',
  `money` decimal(12,4) DEFAULT '0.0000' COMMENT '账户余额',
  `frozen_money` decimal(12,2) DEFAULT '0.00' COMMENT '冻结金额',
  `integral` decimal(12,0) DEFAULT '0' COMMENT '积分',
  `autograph` varchar(255) NOT NULL DEFAULT '' COMMENT '个性签名',
  `hobby` varchar(255) NOT NULL COMMENT '爱好',
  `reg_code` varchar(50) DEFAULT '' COMMENT '邀请码',
  `login_status` tinyint(1) DEFAULT '1' COMMENT '登录状态1允许2禁止',
  `login_token` varchar(32) DEFAULT '' COMMENT '登录token',
  `path` text COMMENT '用户关系',
  `system` tinyint(1) DEFAULT '0' COMMENT '操作系统0未知1安卓2ios',
  `country` varchar(255) DEFAULT '' COMMENT '所属国家',
  `province` varchar(255) DEFAULT '' COMMENT '所属省',
  `city` varchar(255) DEFAULT '' COMMENT '所属城市',
  `login_ip` varchar(50) DEFAULT '' COMMENT '登录ip',
  `last_login_time` int(11) DEFAULT '0' COMMENT '登录时间',
  `last_login_device` varchar(255) DEFAULT '' COMMENT '最后登录设备号',
  `reg_ip` varchar(255) DEFAULT '' COMMENT '注册ip',
  `add_time` int(11) DEFAULT '0',
  `update_time` int(11) DEFAULT '0',
  `user_sig` varchar(255) DEFAULT NULL COMMENT '腾讯IM唯一标识',
  `is_online` tinyint(1) DEFAULT '2',
  `is_vip` tinyint(1) DEFAULT '2' COMMENT '是否vip  1是2否',
  `vip_end_time` bigint(20) DEFAULT '0' COMMENT 'vip截止时间',
  `is_real` int(1) DEFAULT '0' COMMENT '实名状态 1成功 3审核中',
  `real_name` varchar(255) DEFAULT NULL COMMENT '真实姓名',
  `card_id` varchar(18) DEFAULT NULL COMMENT '身份证号',
  `is_customer` tinyint(1) DEFAULT '2' COMMENT '1 官方客服  2 正常用户',
  `trade_password` varchar(255) DEFAULT '' COMMENT '交易密码',
  `is_admin` tinyint(255) DEFAULT '2' COMMENT '1 后台注册  2 app',
  `is_show` tinyint(1) DEFAULT '1' COMMENT '1 显示  2 不显示',
  `reg_type` tinyint(1) DEFAULT '1' COMMENT '1 ',
  `system_type` tinyint(1) DEFAULT '1' COMMENT '1 引才荟  2 企才荟 3新后台 4 后台',
  `invite_code` varchar(255) DEFAULT '' COMMENT '邀请码',
  `is_first_login` tinyint(1) DEFAULT '1' COMMENT '1 是第一次登录  2 否',
  `banned_time` int(11) DEFAULT '0' COMMENT '封禁/解封时间',
  `no_check` tinyint(1) DEFAULT '0' COMMENT '1不用送审',
  PRIMARY KEY (`uid`) USING BTREE,
  UNIQUE KEY `index2` (`user_name`) USING BTREE,
  KEY `index1` (`login_token`) USING BTREE,
  KEY `index3` (`uid`,`update_time`,`is_online`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=106866 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='用户表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user`
--

LOCK TABLES `yy_user` WRITE;
/*!40000 ALTER TABLE `yy_user` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_banned_file`
--

DROP TABLE IF EXISTS `yy_user_banned_file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_banned_file` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `filepath` varchar(255) NOT NULL DEFAULT '' COMMENT '文件路径',
  `type` tinyint(1) NOT NULL DEFAULT '0' COMMENT '1 封禁  2 解封',
  `add_time` int(11) NOT NULL DEFAULT '0',
  `update_time` int(11) NOT NULL DEFAULT '0',
  `banned_type` tinyint(1) NOT NULL DEFAULT '1' COMMENT '1 用户id   2 ip',
  `system_type` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=79 DEFAULT CHARSET=utf8mb4 COMMENT='用户封禁文件';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_banned_file`
--

LOCK TABLES `yy_user_banned_file` WRITE;
/*!40000 ALTER TABLE `yy_user_banned_file` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_banned_file` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_black`
--

DROP TABLE IF EXISTS `yy_user_black`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_black` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) NOT NULL DEFAULT '0' COMMENT '用户id',
  `receive_uid` int(11) NOT NULL DEFAULT '0' COMMENT '拉黑用户id',
  `add_time` int(11) NOT NULL DEFAULT '0' COMMENT '拉黑时间',
  `update_time` int(11) NOT NULL DEFAULT '0' COMMENT '解除拉黑时间',
  PRIMARY KEY (`id`),
  KEY `index1` (`uid`,`receive_uid`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户拉黑列表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_black`
--

LOCK TABLES `yy_user_black` WRITE;
/*!40000 ALTER TABLE `yy_user_black` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_black` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_copy1`
--

DROP TABLE IF EXISTS `yy_user_copy1`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_copy1` (
  `uid` int(11) NOT NULL AUTO_INCREMENT,
  `pid` int(11) DEFAULT '0' COMMENT '上级id',
  `user_name` varchar(50) DEFAULT '' COMMENT '用户名手机号',
  `password` varchar(50) DEFAULT '' COMMENT '登录密码',
  `nick_name` varchar(30) DEFAULT '' COMMENT '用户昵称',
  `base64_nick_name` varchar(255) DEFAULT '' COMMENT 'base64 用户昵称',
  `head_pic` varchar(255) DEFAULT '' COMMENT '用户头像',
  `sex` tinyint(1) DEFAULT '1' COMMENT '0未知1男2女',
  `birthday` varchar(50) DEFAULT NULL COMMENT '出生日期',
  `constellation` varchar(255) DEFAULT '' COMMENT '星座',
  `money` decimal(12,4) DEFAULT '0.0000' COMMENT '账户余额',
  `frozen_money` decimal(12,2) DEFAULT '0.00' COMMENT '冻结金额',
  `integral` decimal(12,0) DEFAULT '0' COMMENT '积分',
  `autograph` varchar(255) NOT NULL DEFAULT '' COMMENT '个性签名',
  `hobby` varchar(255) NOT NULL COMMENT '爱好',
  `reg_code` varchar(50) DEFAULT '' COMMENT '邀请码',
  `login_status` tinyint(1) DEFAULT '1' COMMENT '登录状态1允许2禁止',
  `login_token` varchar(32) DEFAULT '' COMMENT '登录token',
  `path` text COMMENT '用户关系',
  `system` tinyint(1) DEFAULT '0' COMMENT '操作系统0未知1安卓2ios',
  `country` varchar(255) DEFAULT '' COMMENT '所属国家',
  `province` varchar(255) DEFAULT '' COMMENT '所属省',
  `city` varchar(255) DEFAULT '' COMMENT '所属城市',
  `login_ip` varchar(50) DEFAULT '' COMMENT '登录ip',
  `last_login_time` int(11) DEFAULT '0' COMMENT '登录时间',
  `last_login_device` varchar(255) DEFAULT '' COMMENT '最后登录设备号',
  `reg_ip` varchar(255) DEFAULT '' COMMENT '注册ip',
  `add_time` int(11) DEFAULT '0',
  `update_time` int(11) DEFAULT '0',
  `user_sig` varchar(255) DEFAULT NULL COMMENT '腾讯IM唯一标识',
  `is_online` tinyint(1) DEFAULT '2',
  `is_vip` tinyint(1) DEFAULT '2' COMMENT '是否vip  1是2否',
  `vip_end_time` bigint(20) DEFAULT '0' COMMENT 'vip截止时间',
  `is_real` int(1) DEFAULT '0' COMMENT '实名状态 1成功 3审核中',
  `real_name` varchar(255) DEFAULT NULL COMMENT '真实姓名',
  `card_id` varchar(18) DEFAULT NULL COMMENT '身份证号',
  `is_customer` tinyint(1) DEFAULT '2' COMMENT '1 官方客服  2 正常用户',
  `trade_password` varchar(255) DEFAULT '' COMMENT '交易密码',
  `is_admin` tinyint(255) DEFAULT '2' COMMENT '1 后台注册  2 app',
  `is_show` tinyint(1) DEFAULT '1' COMMENT '1 显示  2 不显示',
  PRIMARY KEY (`uid`) USING BTREE,
  UNIQUE KEY `index2` (`user_name`) USING BTREE,
  KEY `index1` (`login_token`) USING BTREE,
  KEY `index3` (`uid`,`update_time`,`is_online`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=10530 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='用户表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_copy1`
--

LOCK TABLES `yy_user_copy1` WRITE;
/*!40000 ALTER TABLE `yy_user_copy1` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_copy1` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_copy2`
--

DROP TABLE IF EXISTS `yy_user_copy2`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_copy2` (
  `uid` int(11) NOT NULL AUTO_INCREMENT,
  `pid` int(11) DEFAULT '0' COMMENT '上级id',
  `user_name` varchar(50) DEFAULT '' COMMENT '用户名手机号',
  `password` varchar(50) DEFAULT '' COMMENT '登录密码',
  `nick_name` varchar(30) DEFAULT '' COMMENT '用户昵称',
  `base64_nick_name` varchar(255) DEFAULT '' COMMENT 'base64 用户昵称',
  `head_pic` varchar(255) DEFAULT '' COMMENT '用户头像',
  `sex` tinyint(1) DEFAULT '1' COMMENT '0未知1男2女',
  `birthday` varchar(50) DEFAULT NULL COMMENT '出生日期',
  `constellation` varchar(255) DEFAULT '' COMMENT '星座',
  `money` decimal(12,4) DEFAULT '0.0000' COMMENT '账户余额',
  `frozen_money` decimal(12,2) DEFAULT '0.00' COMMENT '冻结金额',
  `integral` decimal(12,0) DEFAULT '0' COMMENT '积分',
  `autograph` varchar(255) NOT NULL DEFAULT '' COMMENT '个性签名',
  `hobby` varchar(255) NOT NULL COMMENT '爱好',
  `reg_code` varchar(50) DEFAULT '' COMMENT '邀请码',
  `login_status` tinyint(1) DEFAULT '1' COMMENT '登录状态1允许2禁止',
  `login_token` varchar(32) DEFAULT '' COMMENT '登录token',
  `path` text COMMENT '用户关系',
  `system` tinyint(1) DEFAULT '0' COMMENT '操作系统0未知1安卓2ios',
  `country` varchar(255) DEFAULT '' COMMENT '所属国家',
  `province` varchar(255) DEFAULT '' COMMENT '所属省',
  `city` varchar(255) DEFAULT '' COMMENT '所属城市',
  `login_ip` varchar(50) DEFAULT '' COMMENT '登录ip',
  `last_login_time` int(11) DEFAULT '0' COMMENT '登录时间',
  `last_login_device` varchar(255) DEFAULT '' COMMENT '最后登录设备号',
  `reg_ip` varchar(255) DEFAULT '' COMMENT '注册ip',
  `add_time` int(11) DEFAULT '0',
  `update_time` int(11) DEFAULT '0',
  `user_sig` varchar(255) DEFAULT NULL COMMENT '腾讯IM唯一标识',
  `is_online` tinyint(1) DEFAULT '2',
  `is_vip` tinyint(1) DEFAULT '2' COMMENT '是否vip  1是2否',
  `vip_end_time` bigint(20) DEFAULT '0' COMMENT 'vip截止时间',
  `is_real` int(1) DEFAULT '0' COMMENT '实名状态 1成功 3审核中',
  `real_name` varchar(255) DEFAULT NULL COMMENT '真实姓名',
  `card_id` varchar(18) DEFAULT NULL COMMENT '身份证号',
  `is_customer` tinyint(1) DEFAULT '2' COMMENT '1 官方客服  2 正常用户',
  `trade_password` varchar(255) DEFAULT '' COMMENT '交易密码',
  `is_admin` tinyint(255) DEFAULT '2' COMMENT '1 后台注册  2 app',
  `is_show` tinyint(1) DEFAULT '1' COMMENT '1 显示  2 不显示',
  PRIMARY KEY (`uid`) USING BTREE,
  UNIQUE KEY `index2` (`user_name`) USING BTREE,
  KEY `index1` (`login_token`) USING BTREE,
  KEY `index3` (`uid`,`update_time`,`is_online`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=10530 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='用户表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_copy2`
--

LOCK TABLES `yy_user_copy2` WRITE;
/*!40000 ALTER TABLE `yy_user_copy2` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_copy2` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_follow`
--

DROP TABLE IF EXISTS `yy_user_follow`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_follow` (
  `fid` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) DEFAULT '0' COMMENT '关注者uid',
  `follow_uid` int(11) DEFAULT '0' COMMENT '被关注用户uid',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  PRIMARY KEY (`fid`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='用户关注表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_follow`
--

LOCK TABLES `yy_user_follow` WRITE;
/*!40000 ALTER TABLE `yy_user_follow` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_follow` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_friend_apply`
--

DROP TABLE IF EXISTS `yy_user_friend_apply`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_friend_apply` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) NOT NULL DEFAULT '0' COMMENT '用户id',
  `friend_uid` int(11) NOT NULL DEFAULT '0' COMMENT '好友用户id',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '1 申请中 2 已通过 3 已拒绝  4 已过期',
  `add_time` int(11) NOT NULL DEFAULT '0' COMMENT '添加',
  `update_time` int(11) NOT NULL DEFAULT '0' COMMENT '更新时间',
  `is_delete` tinyint(1) NOT NULL DEFAULT '1' COMMENT '1 正常  2 删除',
  PRIMARY KEY (`id`),
  KEY `index1` (`uid`,`status`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友申请列表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_friend_apply`
--

LOCK TABLES `yy_user_friend_apply` WRITE;
/*!40000 ALTER TABLE `yy_user_friend_apply` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_friend_apply` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_money_log`
--

DROP TABLE IF EXISTS `yy_user_money_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_money_log` (
  `log_id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) DEFAULT '0' COMMENT '用户ID',
  `change_type` tinyint(3) DEFAULT '0',
  `change_value` decimal(14,4) DEFAULT '0.0000',
  `after_money` decimal(14,4) DEFAULT NULL,
  `from_uid` int(11) DEFAULT NULL COMMENT '来自uid',
  `remarks` varchar(255) DEFAULT '',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  `from_id` int(11) DEFAULT '0' COMMENT '来自id',
  PRIMARY KEY (`log_id`) USING BTREE,
  KEY `fffgg` (`uid`,`change_type`,`add_time`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='用户资金日志表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_money_log`
--

LOCK TABLES `yy_user_money_log` WRITE;
/*!40000 ALTER TABLE `yy_user_money_log` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_money_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_real_name`
--

DROP TABLE IF EXISTS `yy_user_real_name`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_real_name` (
  `nid` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uid` int(11) NOT NULL COMMENT '用户id',
  `real_name` varchar(255) NOT NULL COMMENT '真实姓名',
  `card_id` varchar(18) NOT NULL COMMENT '身份证号',
  `identity1` varchar(255) NOT NULL COMMENT '人像照',
  `identity2` varchar(255) NOT NULL COMMENT '国徽照',
  `status` int(1) NOT NULL DEFAULT '1' COMMENT '1待审核 2失败 3通过',
  `remarke` varchar(255) DEFAULT NULL COMMENT '备注',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  PRIMARY KEY (`nid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户实名表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_real_name`
--

LOCK TABLES `yy_user_real_name` WRITE;
/*!40000 ALTER TABLE `yy_user_real_name` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_real_name` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_red_envelope_log`
--

DROP TABLE IF EXISTS `yy_user_red_envelope_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_red_envelope_log` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) DEFAULT '0' COMMENT '用户id',
  `rid` int(11) DEFAULT '0' COMMENT '房间id',
  `group_id` int(11) DEFAULT '0' COMMENT '家族id',
  `type` tinyint(1) DEFAULT '1' COMMENT '红包类型 1房间 2全服 3家族',
  `eid` int(11) DEFAULT '0' COMMENT '红包id',
  `money` decimal(10,2) DEFAULT '0.00' COMMENT '红包金额',
  `price` decimal(10,2) DEFAULT '0.00' COMMENT '红包实际金额',
  `snatch_price` decimal(10,2) DEFAULT '0.00' COMMENT '抢到的金额',
  `num` int(11) DEFAULT '0' COMMENT '红包总数',
  `add_time` int(11) DEFAULT '0',
  `update_time` int(11) DEFAULT '0',
  `is_lucky` tinyint(1) DEFAULT '2' COMMENT '是否手气最佳 1是2否',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `uid` (`uid`,`rid`,`type`,`eid`) USING BTREE,
  KEY `eid` (`eid`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='用户抢红包记录表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_red_envelope_log`
--

LOCK TABLES `yy_user_red_envelope_log` WRITE;
/*!40000 ALTER TABLE `yy_user_red_envelope_log` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_red_envelope_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_report`
--

DROP TABLE IF EXISTS `yy_user_report`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_report` (
  `rid` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) DEFAULT '0' COMMENT '举报人',
  `to_uid` int(11) DEFAULT '0' COMMENT '被举报用户id',
  `type_id` tinyint(11) DEFAULT '0' COMMENT '举报类型',
  `image` varchar(255) DEFAULT '',
  `content` text COMMENT '举报内容',
  `status` tinyint(1) DEFAULT '1' COMMENT '1待处理2已处理',
  `deal_time` int(11) DEFAULT '0',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  PRIMARY KEY (`rid`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=69 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='用户举报表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_report`
--

LOCK TABLES `yy_user_report` WRITE;
/*!40000 ALTER TABLE `yy_user_report` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_report` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_withdrawal`
--

DROP TABLE IF EXISTS `yy_user_withdrawal`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_withdrawal` (
  `wid` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `order_sn` varchar(255) NOT NULL COMMENT '订单',
  `uid` int(11) NOT NULL COMMENT '用户id',
  `address` varchar(255) NOT NULL COMMENT '提现地址',
  `money` decimal(10,2) NOT NULL COMMENT '提现金额',
  `general_money` decimal(10,2) NOT NULL COMMENT '到账金额',
  `service_money` decimal(10,2) NOT NULL COMMENT '手续费',
  `status` int(1) NOT NULL DEFAULT '1' COMMENT '1正常 2失败 3成功',
  `remarke` varchar(255) DEFAULT NULL COMMENT '备注',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  PRIMARY KEY (`wid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户提现表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_withdrawal`
--

LOCK TABLES `yy_user_withdrawal` WRITE;
/*!40000 ALTER TABLE `yy_user_withdrawal` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_withdrawal` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_zone`
--

DROP TABLE IF EXISTS `yy_user_zone`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_zone` (
  `zid` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) DEFAULT '0' COMMENT '用户id',
  `images` text COMMENT '多图',
  `content` text COMMENT '空间内容',
  `base64_content` text,
  `praise_num` int(11) DEFAULT '0' COMMENT '点赞数',
  `read_num` int(11) DEFAULT '0' COMMENT '阅读数量',
  `share_num` int(11) DEFAULT '0' COMMENT '分享次数',
  `comment_num` int(11) DEFAULT '0' COMMENT '评论数量',
  `show_status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '1:未审核  2:审核通过 3审核拒绝',
  `add_time` int(11) DEFAULT '0',
  `update_time` int(11) DEFAULT NULL,
  `is_delete` tinyint(1) DEFAULT '1' COMMENT '是否删除 1否2是',
  `delete_time` int(11) DEFAULT '0',
  PRIMARY KEY (`zid`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='用户空间表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_zone`
--

LOCK TABLES `yy_user_zone` WRITE;
/*!40000 ALTER TABLE `yy_user_zone` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_zone` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_zone_comment`
--

DROP TABLE IF EXISTS `yy_user_zone_comment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_zone_comment` (
  `cid` int(11) NOT NULL AUTO_INCREMENT,
  `zid` int(11) DEFAULT '0' COMMENT '动态id',
  `uid` int(11) DEFAULT '0' COMMENT '评论id',
  `receive_uid` int(11) DEFAULT '0' COMMENT '被评论用户id',
  `content` text COMMENT '评论内容',
  `base64_content` text,
  `is_show` tinyint(1) DEFAULT '1' COMMENT '1显示2隐藏',
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  `is_delete` tinyint(1) DEFAULT '1' COMMENT '是否删除 1否2是',
  PRIMARY KEY (`cid`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='用户评论表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_zone_comment`
--

LOCK TABLES `yy_user_zone_comment` WRITE;
/*!40000 ALTER TABLE `yy_user_zone_comment` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_zone_comment` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yy_user_zone_praise`
--

DROP TABLE IF EXISTS `yy_user_zone_praise`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yy_user_zone_praise` (
  `pid` int(11) NOT NULL AUTO_INCREMENT,
  `zid` int(11) DEFAULT NULL,
  `uid` int(11) DEFAULT NULL,
  `add_time` int(11) DEFAULT NULL,
  `update_time` int(11) DEFAULT NULL,
  PRIMARY KEY (`pid`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='用户点赞表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yy_user_zone_praise`
--

LOCK TABLES `yy_user_zone_praise` WRITE;
/*!40000 ALTER TABLE `yy_user_zone_praise` DISABLE KEYS */;
/*!40000 ALTER TABLE `yy_user_zone_praise` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Dumping events for database 'jishitongxun_com'
--

--
-- Dumping routines for database 'jishitongxun_com'
--
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-08-18  4:00:03
