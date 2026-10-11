var protocol = window.location.protocol;
var hostname = window.location.hostname;
var port = window.location.port;
var WebHost = protocol + '//' + hostname;
if (port) {
	WebHost = WebHost + ':' + port;
}
WebHost = WebHost + '/admin/';
var AdminLoginToken = localStorage.getItem('admin_login_token') //用户本地token

/* 时间戳转化开始 */
Date.prototype.format = function (fmt) {
	//author: meizz
	var o = {
		'M+': this.getMonth() + 1, //月份
		'd+': this.getDate(), //日
		'h+': this.getHours(), //小时
		'm+': this.getMinutes(), //分
		's+': this.getSeconds(), //秒
		'q+': Math.floor((this.getMonth() + 3) / 3), //q是季度
		S: this.getMilliseconds(), //毫秒
	}
	if (/(y+)/.test(fmt))
		fmt = fmt.replace(
			RegExp.$1,
			(this.getFullYear() + '').substr(4 - RegExp.$1.length)
		)
	for (var k in o)
		if (new RegExp('(' + k + ')').test(fmt))
			fmt = fmt.replace(
				RegExp.$1,
				RegExp.$1.length == 1
					? o[k]
					: ('00' + o[k]).substr(('' + o[k]).length)
			)
	return fmt
}

String.prototype.format = function (args) {
	var result = this
	if (arguments.length > 0) {
		if (arguments.length == 1 && typeof args == 'loginTime') {
			for (var key in args) {
				if (args[key] != undefined) {
					var reg = new RegExp('({' + key + '})', 'g')
					result = result.replace(reg, args[key])
				}
			}
		} else {
			for (var i = 0; i < arguments.length; i++) {
				if (arguments[i] != undefined) {
					//var reg = new RegExp("({[" + i + "]})", "g");//这个在索引大于9时会有问题
					var reg = new RegExp('({)' + i + '(})', 'g')
					result = result.replace(reg, arguments[i])
				}
			}
		}
	}
	return result
}

function dateFormat(value) {
	return value ? new Date(value * 1000).format('yyyy-MM-dd hh:mm:ss') : ''
}

function money_format(
	number = 0,
	decimals = 2,
	dec_point = '.',
	thousands_sep = ','
) {
	/*
	 * 参数说明：
	 * number：要格式化的数字
	 * decimals：保留几位小数
	 * dec_point：小数点符号
	 * thousands_sep：千分位符号
	 * */
	number = (number + '').replace(/[^0-9+-Ee.]/g, '')
	var n = !isFinite(+number) ? 0 : +number,
		prec = !isFinite(+decimals) ? 0 : Math.abs(decimals),
		sep = typeof thousands_sep === 'undefined' ? ',' : thousands_sep,
		dec = typeof dec_point === 'undefined' ? '.' : dec_point,
		s = '',
		toFixedFix = function (n, prec) {
			var k = Math.pow(10, prec)
			return '' + Math.floor(n * k) / k
		}
	s = (prec ? toFixedFix(n, prec) : '' + Math.floor(n)).split('.')
	var re = /(-?\d+)(\d{3})/
	while (re.test(s[0])) {
		s[0] = s[0].replace(re, '$1' + sep + '$2')
	}

	if ((s[1] || '').length < prec) {
		s[1] = s[1] || ''
		s[1] += new Array(prec - s[1].length + 1).join('0')
	}
	return s.join(dec)
}
