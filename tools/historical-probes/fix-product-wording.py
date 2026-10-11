from pathlib import Path
root=Path(__file__).resolve().parent.parent/'hw-bridge-handoff-20260929'
android=root/'02-authoritative-qch-source/企才汇源码交付-20260810/Android'
for name,field in [('RegistActivity.java','pass2'),('RegistActivity2.java','pass3')]:
 p=android/'qch/app/src/main/java/com/qchm/xyz/bj'/name
 s=p.read_text('utf-8')
 old='strictBridgeRegistration && (!phone.matches("(?:[A-Za-z][A-Za-z0-9_]{7,21}|[0-9]{8,22})") || TextUtils.isEmpty('+field+'))'
 new='strictBridgeRegistration && !phone.matches("(?:[A-Za-z][A-Za-z0-9_]{7,21}|[0-9]{8,22})")'
 assert old in s
 s=s.replace(old,new).replace('用户名须为8至22位，英文字母开头，可含数字和下划线；纯数字账号需邀请码允许。请输入有效叨叨邀请码。','用户名须为8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线。')
 s=s.replace('叨叨邀请码（必填）','邀请码（选填）').replace('手机号（需邀请码允许数字账号）','手机号').replace('注册已保存，叨叨桥接确认中。请稍后登录，无需重复注册','注册信息已保存，账号正在开通，请稍后登录，无需重复注册')
 p.write_text(s,'utf-8')
replacements={
'叨叨群编号无效':'群编号无效','叨叨用户账号无效':'用户账号无效',
'桥接群操作失败':'群操作失败','桥接好友申请失败':'好友申请失败',
'桥接服务响应异常':'服务响应异常，请稍后重试','桥接响应过大':'服务响应异常，请稍后重试',
'桥接操作失败，请稍后重试':'操作失败，请稍后重试',
'未找到叨叨用户':'未找到该用户','叨叨用户资料无效':'用户资料无效','叨叨用户搜索失败':'用户搜索失败',
'叨叨群请使用邀请成员入口':'请使用邀请成员入口', '[桥接提示]':'[系统提示]',
'桥接好友操作失败，请稍后重试':'好友操作失败，请稍后重试','桥接服务暂不可用':'服务暂不可用，请稍后重试',
'叨叨黑名单更新失败':'黑名单更新失败',
'本地账号已创建，叨叨桥接确认中，请稍后重试':'账号已创建，聊天服务开通中，请稍后重试',
'叨叨桥接账号暂不可用，请确认账号状态后重试':'聊天服务暂不可用，请确认账号状态后重试',
'叨叨桥接确认未完成，请稍后重试':'聊天服务开通中，请稍后重试',
'叨叨用户尚未登录':'用户尚未登录','叨叨用户凭据已失效，请重新登录':'登录已失效，请重新登录',
'叨叨好友申请':'好友申请','叨叨账号已被停用，请联系管理员':'聊天服务已停用，请联系管理员',
'叨叨服务已停用，请联系管理员':'聊天服务已停用，请联系管理员',
'好友桥接服务未配置':'好友服务暂不可用，请稍后重试','当前账号尚未开通桥接':'聊天服务尚未就绪，请稍后重试',
'文件助手不参与跨系统桥接':'该联系人暂不支持此操作','请先搜索叨叨用户':'请先搜索用户',
'该群不是叨叨镜像群':'该群暂不支持此操作','成员尚未开通桥接，请先搜索或登录':'成员聊天服务尚未就绪，请稍后重试',
'请至少选择一位叨叨好友建群':'请至少选择一位好友建群','叨叨支持禁言':'可选择禁言',
'当前账号不在叨叨群内':'当前账号不在群内','叨叨群已解散':'群已解散',
'叨叨联系人未映射':'联系人资料尚未就绪，请稍后重试','桥接群未映射':'群资料尚未就绪，请稍后重试',
'当前账号不在桥接群中':'当前账号不在群中','叨叨联系人映射不匹配':'联系人资料异常，请刷新后重试',
'桥接登录待重试':'账号开通中，请稍后重试',
'用户名须为8至22位英文字母开头的字母、数字或下划线；纯数字账号需邀请码允许':'用户名须为8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线',
'用户名须为8至22位字母、数字或下划线，纯数字账号需邀请码允许':'用户名须为8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线',
'8至22位，英文字母开头，可含数字和下划线；纯数字账号需邀请码允许':'8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线',
}
files=[]
for base,ext in [(android,'.java'),(android,'.xml'),(root/'03-php-deployment/backend/application/imaccess','.php'),(root/'01-bridge/tencent-tangseng-bridge/internal','.go')]:
 files.extend(p for p in base.rglob('*'+ext) if 'build' not in p.relative_to(base).parts and not p.name.endswith('_test.go'))
files.extend([root/'03-php-deployment/backend/application/api/model/User.php',root/'03-php-deployment/backend/application/api/controller/Login.php'])
for p in files:
 try:s=p.read_text('utf-8')
 except (FileNotFoundError,UnicodeError):continue
 n=s
 for a,b in replacements.items():n=n.replace(a,b)
 n=n.replace('fmt.Sprintf("[暂不支持的叨叨消息类型:%d]", v.Type)','"[暂不支持此消息]"')
 if n!=s:p.write_text(n,'utf-8')
print('PRODUCT_WORDING_UPDATED')
