(function ($) {

	function xml2json(Xml) {
		var tempvalue, tempJson = {};
		$(Xml).each(function() {
			var tagName = ($(this).attr('id') || this.tagName);
			tempvalue = (this.childElementCount == 0) ? this.textContent : xml2json($(this).children());
			switch ($.type(tempJson[tagName])) {
				case 'undefined':
					tempJson[tagName] = tempvalue;
					break;
				case 'object':
					tempJson[tagName] = Array(tempJson[tagName]);
				case 'array':
					tempJson[tagName].push(tempvalue);
			}
		});
		return tempJson;
	}

	function setCookie (c_name, value, expiredays) {
		var exdate = new Date();
		exdate.setDate(exdate.getDate() + expiredays);
		document.cookie = c_name + '=' + escape(value) + ((expiredays == null) ? '' : ';expires=' + exdate.toGMTString())+ '; path='+window.nps.web_base_url+'/;';
	}

	function getCookie (c_name) {
		if (document.cookie.length > 0) {
			c_start = document.cookie.indexOf(c_name + '=');
			if (c_start != -1) {
				c_start = c_start + c_name.length + 1;
				c_end = document.cookie.indexOf(';', c_start);
				if (c_end == -1) c_end = document.cookie.length;
				return unescape(document.cookie.substring(c_start, c_end));
			}
		}
		return null;
	}

	function setchartlang (langobj,chartobj) {
		if ( $.type (langobj) == 'string' ) return langobj;
		if ( $.type (langobj) == 'chartobj' ) return false;
		var flag = true;
		for (key in langobj) {
			var item = key;
			children = (chartobj.hasOwnProperty(item)) ? setchartlang (langobj[item],chartobj[item]) : setchartlang (langobj[item],undefined);
			switch ($.type(children)) {
				case 'string':
					if ($.type(chartobj[item]) != 'string' ) continue;
				case 'object':
					chartobj[item] = (children['value'] || children);
				default:
					flag = false;
			}
		}
		if (flag) { return {'value':(langobj[languages['current']] || langobj[languages['default']] || 'N/A')}}
	}

	$.fn.cloudLang = function () {
		$.ajax({
			type: 'GET',
			url: window.nps.web_base_url + '/static/page/languages.xml',
			dataType: 'xml',
			success: function (xml) {
				languages['content'] = xml2json($(xml).children())['content'];
				languages['menu'] = languages['content']['languages'];
				languages['default'] = languages['content']['default'];
				languages['navigator'] = (getCookie ('lang') || navigator.language || navigator.browserLanguage);
				for(var key in languages['menu']){
					$('#languagemenu').next().append('<li lang="' + key + '"><a><img src="' + window.nps.web_base_url + '/static/img/flag/' + key + '.png"> ' + languages['menu'][key] +'</a></li>');
					if ( key == languages['navigator'] ) languages['current'] = key;
				}
				$('#languagemenu').attr('lang',(languages['current'] || languages['default']));
				$('body').setLang ('');
			}
		});
	};

	$.fn.setLang = function (dom) {
		languages['current'] = $('#languagemenu').attr('lang');
		if ( dom == '' ) {
			$('#languagemenu span').text(' ' + languages['menu'][languages['current']]);
			if (languages['current'] != getCookie('lang')) setCookie('lang', languages['current']);
			if($("#table").length>0) $('#table').bootstrapTable('refreshOptions', { 'locale': languages['current']});
		}
		$.each($(dom + ' [langtag]'), function (i, item) {
			var index = $(item).attr('langtag');
			string = languages['content'][index.toLowerCase()];
			switch ($.type(string)) {
				case 'string':
					break;
				case 'array':
					string = string[Math.floor((Math.random()*string.length))];
				case 'object':
					string = (string[languages['current']] || string[languages['default']] || null);
					break;
				default:
					string = 'Missing language string "' + index + '"';
					$(item).css('background-color','#ffeeba');
			}
			if($.type($(item).attr('placeholder')) == 'undefined') {
				$(item).text(string);
			} else {
				$(item).attr('placeholder', string);
			}
		});

		if ( !$.isEmptyObject(chartdatas) ) {
			setchartlang(languages['content']['charts'],chartdatas);
			for(var key in chartdatas){
				if ($('#'+key).length == 0) continue;
				if($.type(chartdatas[key]) == 'object')
				charts[key] = echarts.init(document.getElementById(key));
				charts[key].setOption(chartdatas[key], true);
			}
		}
	}

})(jQuery);

$(document).ready(function () {
	$('body').cloudLang();
	$('body').on('click','li[lang]',function(){
		$('#languagemenu').attr('lang',$(this).attr('lang'));
		$('body').setLang ('');
	});
});

var languages = {};
var charts = {};
var chartdatas = {};
var postsubmit;

function langreply(langstr) {
    var langobj = languages['content']['reply'][langstr.replace(/[\s,\.\?]*/g,"").toLowerCase()];
    if ($.type(langobj) == 'undefined') return langstr
    langobj = (langobj[languages['current']] || langobj[languages['default']] || langstr);
    return langobj
}

function submitform(action, url, postdata) {
    postsubmit = false;
    switch (action) {
        case 'start':
        case 'stop':
        case 'delete':
            var langobj = languages['content']['confirm'][action];
            action = (langobj[languages['current']] || langobj[languages['default']] || 'Are you sure you want to ' + action + ' it?');
            if (! confirm(action)) return;
            postsubmit = true;
        case 'add':
        case 'edit':
            $.ajax({
                type: "POST",
                url: url,
                data: postdata,
                success: function (res) {
                    alert(langreply(res.msg));
                    if (res.status) {
                        if (postsubmit) {document.location.reload();}else{history.back(-1);}
                    }
                }
            });
    }
}
function changeunit(len) {
	//1 Byte(B) = 8bit = 8b
	//1 Kilo    Byte(KB) = 1024B
	//1 Mega    Byte(MB) = 1024KB
	//1 Giga    Byte(GB) = 1024MB
	//1 Tera    Byte(TB) = 1024GB
	//1 Peta    Byte(PB) = 1024TB
	//1 Exa     Byte(EB) = 1024PB
	//1 Zetta   Byte(ZB) = 1024EB
	//1 Yotta   Byte(YB) = 1024ZB
	//1 Bronto  Byte(BB) = 1024YB
	//1 Nona    Byte(NB) = 1024BB
	//1 Dogga   Byte(DB) = 1024NB
	//1 Corydon Byte(CB) = 1024DB
	//1 Xero    Byte(XB) = 1024CB

	let Bit = len
	let KB = Bit / 1024
	let MB = KB / 1024
	let GB = MB / 1024
	let TB = GB / 1024
	let PB = TB / 1024
	let EB = PB / 1024
	let ZB = EB / 1024
	let YB = ZB / 1024
	let BB = YB / 1024
	let NB = BB / 1024
	let CB = NB / 1024
	let XB = CB / 1024
	if (Bit < 1024) {
		return Bit.toFixed(2) + "B"
	} else if (KB < 1024) {
		return KB.toFixed(2) + "KB"
	} else if (MB < 1024) {
		return MB.toFixed(2) + "MB"
	} else if (GB < 1024) {
		return GB.toFixed(2) + "GB"
	} else if (TB < 1024) {
		return TB.toFixed(2) + "TB"
	} else if (PB < 1024) {
		return PB.toFixed(2) + "PB"
	} else if (EB < 1024) {
		return EB.toFixed(2) + "EB"
	} else if (ZB < 1024) {
		return ZB.toFixed(2) + "ZB"
	} else if (YB < 1024) {
		return YB.toFixed(2) + "YB"
	} else if (BB < 1024) {
		return BB.toFixed(2) + "BB"
	} else if (NB < 1024) {
		return NB.toFixed(2) + "NB"
	} else if (CB < 1024) {
		return CB.toFixed(2) + "CB"
	} else {
		return XB.toFixed(2)+ "XB"
	}

}
// function changeunit(limit) {
//     var size = "";
//     if (limit < 0.1 * 1024) {
//         size = limit.toFixed(2) + "B";
//     } else if (limit < 0.1 * 1024 * 1024) {
//         size = (limit / 1024).toFixed(2) + "KB";
//     } else if (limit < 0.1 * 1024 * 1024 * 1024) {
//         size = (limit / (1024 * 1024)).toFixed(2) + "MB";
//     } else {
//         size = (limit / (1024 * 1024 * 1024)).toFixed(2) + "GB";
//     }
//
//     var sizeStr = size + "";
//     var index = sizeStr.indexOf(".");
//     var dou = sizeStr.substr(index + 1, 2);
//     if (dou == "00") {
//         return sizeStr.substring(0, index) + sizeStr.substr(index + 3, 2);
//     }
//     return size;
// }