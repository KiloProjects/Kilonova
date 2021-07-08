import dayjs from 'dayjs';
import 'dayjs/locale/ro';

import cookie from 'js-cookie';

import { Notyf } from 'notyf';

import qs from 'query-string';

dayjs.locale('ro')

// if max_step is 0, it will format until the MB region
// else if max_step is 1, it will only format until the KB region
// else if max_step >= 2, it will append a " B" to the end of the number
export function sizeFormatter(size, max_step, floor) {
	var units = size
	var suffix = "B"
	if(max_step === null || max_step === undefined) {
		max_step = 0
	}
	if(size > 1024*1024 && max_step == 0) {
		units = (size/1024/1024).toFixed(2) 
		suffix = "MB"
	} else if(size > 1024 && max_step <= 1) {
		units = (size/1024).toFixed(2)
		suffix = "KB"
	}
	if(floor) {
		units = Math.floor(units)
	}
	return units + " " + suffix
}

export function downloadBlob(blob, filename) {
	if (window.navigator.msSaveBlob) { // IE10+
		window.navigator.msSaveBlob(file, filename)
	} else {
		var a = document.createElement('a');
		a.href = URL.createObjectURL(blob);
		a.download = filename;
		a.click();
	}
}

export function parseTime(str) {
	if(!str) {
		return ""
	}
	return dayjs(str).format('DD/MM/YYYY HH:mm')
}

// util functions

let notyfConf = {
	duration: 6000,
	ripple: false,
	dismissible: true,
	types: [
		{
			type: 'info',
			background: 'blue',
			icon: {
				className: 'fas fa-info-circle',
				tagName: 'i',
				color: 'white'
			}
		},
		{
			type: 'error',
			background: 'red',
			icon: {
				className: 'fas fa-exclamation-triangle',
				tagName: 'i',
				color: 'white'
			}
		},
		{
			type: 'success',
			background: 'green',
			icon: {
				className: 'fas fa-check-circle',
				tagName: 'i',
				color: 'white'
			}
		},
	]
}

var notyf = undefined;

window.addEventListener('load', () => {
	notyf = new Notyf(notyfConf);
})

/* createToast options
	title: the toast title
	description: the toast description
	status: the toast status (default "info", can be ["success", "error", "info"])
*/
let createToast = options => {
	if(notyf === undefined) {
		console.warn("createToast called before window load")
		return
	}

	if (options.status == null) {
		options.status = "info"
	}

	let msg = "";
	if(options.title !== undefined && options.title !== null && options.title !== "") {
		msg += `<h3>${options.title}</h3>`;
	}
	if(options.description !== undefined && options.description !== null && options.description !== "") {
		msg += options.description;
	}

	var notification = notyf.open({
		type: options.status,
		message: msg
	})
}

let apiToast = (res, overwrite) => {
	if(overwrite === null || overwrite === undefined) {
		overwrite = {}
	}
	overwrite["status"] = res.status 
	overwrite["description"] = res.data
	createToast(overwrite)
}

let getGradient = (score, maxscore) => {
	let col = "#e3dd71", rap = 0.0;
	if(maxscore != 0 && score != 0) { rap = score / maxscore }	
	if(rap == 1.0) { col = "#7fff00" }
	if(rap < 1.0) { col = "#67cf39" }
	if(rap <= 0.8) { col = "#9fdd2e" }
	if(rap <= 0.6) { col = "#d2eb19" }
	if(rap <= 0.4) { col = "#f1d011" }
	if(rap <= 0.2) { col = "#f6881e" }
	if(rap == 0) { col = "#f11722" }
	return col
}

let getCall = async (call, params) => {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	try {
		let resp = await fetch(`/api/${call}?${qs.stringify(params)}`, {headers: {'Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"}});
		return await resp.json()
	} catch(e) {
		return {status: "error", data: e.toString()}
	}
}

let postCall = async (call, params) => {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	try {
		let resp = await fetch(`/api/${call}`, {
			method: 'POST',
			headers: {'Content-Type': 'application/x-www-form-urlencoded','Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"},
			body: qs.stringify(params)
		});
		return await resp.json();
	} catch(e) {
		return {status: "error", data: e.toString()}
	}
}

let bodyCall = async (call, body) => {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	try {
		let resp = await fetch(`/api/${call}`, {
			method: 'POST',
			headers: {'Content-Type': 'application/json', 'Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"},
			body: JSON.stringify(body)
		});
		return await resp.json();
	} catch(e) {
		return {status: "error", data: e.toString()}
	}
}

let multipartCall = async (call, formdata) => {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	try {
		let resp = await fetch(`/api/${call}`, {
			method: 'POST',
			headers: {'Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"},
			body: formdata
		});
		return await resp.json();
	} catch(e) {
		return {status: "error", data: e.toString()}
	}
}

let resendEmail = async () => {
	let res = await postCall('/user/resendEmail')	
	apiToast(res)
	return res
}

let languages = {
	"c": "text/x-csrc",
	"cpp": "text/x-c++src",
	"golang": "text/x-go",
	"haskell": "text/x-haskell",
	"java": "text/x-java",
	"python": "text/x-python",
}

export { languages };

export { 
	dayjs, cookie, 
	createToast, getGradient, apiToast,
	getCall, postCall, bodyCall, multipartCall,
	resendEmail
};

export { SubmissionsApp } from './subs_view.js';
export { NavBarManager } from './navbar.js';
export { SubmissionManager } from './sub_mgr.js';
export { CheckboxManager } from './checkbox_mgr.js';
export { getFileIcon, extensionIcons } from './cdn_mgr.js';
