import cookie from 'js-cookie';
import axios from 'axios';

export interface RequestConfig {
	method: 'GET' | 'POST',
	url: string,
	
	data?: string,
	headers?: Record<string, string>,
	uploadProgress?: (event: ProgressEvent) => void,
}

export interface APIResponse {
	status: 'success' | 'error',
	data: any,
}

export type HTTPClient = (config: RequestConfig) => Promise<object>;

export class Client {
	session_id: string = cookie.get('kn-sessionid') || 'guest';
	doRequest: HTTPClient = axiosRequest

	defaultHeaders(): Record<string, string> {
		return {
			'Accept': 'application/json', 
			'Authorization': this.session_id,
		}
	}

	async apiRequest(config: RequestConfig): Promise<APIResponse> {
		if(config.url.startsWith('/')) {
			config.url = `/api${config.url}`
		} else {
			config.url = `/api/${config.url}`
		}
		config.headers = Object.assign({}, this.defaultHeaders(), config.headers)
		try {
			let res = await this.doRequest(config)
			return res as APIResponse
		} catch(e) {
			return {status: 'error', data: (e as TypeError).toString()}
		}
	}

	async getRequest(call: string, params?: Record<string, any>): Promise<APIResponse> {
		return await this.apiRequest({
			method: 'GET', 
			url: `${call}?${new URLSearchParams(params).toString()}`
		})
	}

	async postRequest(call: string, data?: Record<string, any>): Promise<APIResponse> {
		return await this.apiRequest({
			method: 'POST',
			url: call,
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			data: new URLSearchParams(data).toString()	
		})
	}

};

export default new Client()

/*
	async function fetchRequest(config: RequestConfig): Promise<object> {
		let data = await fetch(config.url, {
			method: config.method, 
			headers: config.headers,
			credentials: 'same-origin',

			body: config.data,
		})
		if(config.uploadProgress !== undefined) {
			console.error("Upload progress field specified, but fetch doesn't support upload progress.")
		}
		return await data.json()
	}
*/

async function axiosRequest(config: RequestConfig): Promise<object> {
	let data = await axios.request({
		method: config.method,
		url: config.url,
		headers: config.headers,
		onUploadProgress: config.uploadProgress,
		data: config.data,

		validateStatus: null,
	})
	return data.data
}
