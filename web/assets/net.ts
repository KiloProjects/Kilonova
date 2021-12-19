import qs from 'query-string';
import cookie from 'js-cookie';

import {Client, APIResponse} from './v2/requests';

const cl = new Client()

export async function getCall(call: string, params?: Record<string, any>): Promise<APIResponse> {
	return await cl.getRequest(call, params)
}

export async function postCall(call: string, data?: Record<string, any>): Promise<APIResponse> {
	return await cl.postRequest(call, data)
}

export async function bodyCall(call: string, body: any): Promise<APIResponse> {
	return await cl.bodyRequest(call, body)
	
}

export async function multipartCall(call: string, formdata: FormData): Promise<APIResponse> {
	return await cl.multipartRequest(call, formdata)
}


/*
export async function getCall(call: string, params: any) {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	try {
		let resp = await fetch(`/api/${call}?${qs.stringify(params)}`, {headers: {'Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"}});
		return await resp.json()
	} catch(e: any) {
		return {status: "error", data: e.toString()}
	}
}

export async function postCall(call: string, params: any) {
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
	} catch(e: any) {
		return {status: "error", data: e.toString()}
	}
}

export async function bodyCall(call: string, body: any) {
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
	} catch(e: any) {
		return {status: "error", data: e.toString()}
	}
}

export async function multipartCall(call: string, formdata: FormData) {
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
	} catch(e: any) {
		return {status: "error", data: e.toString()}
	}
}
*/
