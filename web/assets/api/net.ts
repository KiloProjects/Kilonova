import qs from "query-string";
import { createToast, dismissToast } from "../toast";
import getText from "../translation";
import { getSession } from "../session";

type Response<T> = { status: "error"; data: string } | { status: "success"; data: T };

export async function getCall<T = any>(call: string, params: any): Promise<Response<T>> {
	if (call.startsWith("/")) {
		call = call.substring(1);
	}
	try {
		let resp = await fetch(`/api/${call}?${qs.stringify(params)}`, {
			headers: {
				Accept: "application/json",
				Authorization: getSession(),
			},
		});
		return (await resp.json()) as Response<T>;
	} catch (e: any) {
		return { status: "error", data: e.toString() };
	}
}

export async function postCall<T = any>(call: string, params: any): Promise<Response<T>> {
	if (call.startsWith("/")) {
		call = call.substring(1);
	}
	try {
		let resp = await fetch(`/api/${call}`, {
			method: "POST",
			headers: {
				"Content-Type": "application/x-www-form-urlencoded",
				Accept: "application/json",
				Authorization: getSession(),
			},
			body: qs.stringify(params),
		});
		return (await resp.json()) as Response<T>;
	} catch (e: any) {
		return { status: "error", data: e.toString() };
	}
}

export async function bodyCall<T = any>(call: string, body: any): Promise<Response<T>> {
	if (call.startsWith("/")) {
		call = call.substring(1);
	}
	try {
		let resp = await fetch(`/api/${call}`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				Accept: "application/json",
				Authorization: getSession(),
			},
			body: JSON.stringify(body),
		});
		return (await resp.json()) as Response<T>;
	} catch (e: any) {
		return { status: "error", data: e.toString() };
	}
}

export async function multipartCall<T = any>(call: string, formdata: FormData): Promise<Response<T>> {
	if (call.startsWith("/")) {
		call = call.substring(1);
	}
	try {
		let resp = await fetch(`/api/${call}`, {
			method: "POST",
			headers: {
				Accept: "application/json",
				Authorization: getSession(),
			},
			body: formdata,
		});
		return (await resp.json()) as Response<T>;
	} catch (e: any) {
		return { status: "error", data: e.toString() };
	}
}

export async function multipartProgressCall<T = any>(call: string, formdata: FormData): Promise<Response<T>> {
	if (call.startsWith("/")) {
		call = call.substring(1);
	}
	try {
		const id = Math.random();
		const toast = createToast({
			status: "progress",
			title: getText("uploading"),
			description: `<upload-progress id="${id}">`,
		});
		const xhr = new XMLHttpRequest();
		const resp = await new Promise<Response<T>>((resolve) => {
			xhr.open("POST", `/api/${call}`, true);
			xhr.responseType = "json";
			xhr.upload.addEventListener("progress", (e) => {
				document.dispatchEvent(
					new CustomEvent<ProgressEventData>("kn-upload-update", {
						detail: {
							id: id,
							cntLoaded: e.loaded,
							cntTotal: e.total,
							computable: e.lengthComputable,
							processing: false,
						},
					})
				);
			});
			xhr.upload.addEventListener("load", (e) => {
				document.dispatchEvent(
					new CustomEvent<ProgressEventData>("kn-upload-update", {
						detail: {
							id: id,
							cntLoaded: e.loaded,
							cntTotal: e.total,
							computable: e.lengthComputable,
							processing: true,
						},
					})
				);
			});
			xhr.upload.addEventListener("error", (e) => {
				console.error(e, xhr.statusText);
				resolve({
					status: "error",
					data: "Upload error",
				});
			});
			xhr.addEventListener("load", () => {
				resolve(xhr.response);
			});
			xhr.addEventListener("error", (e) => {
				console.error(e);
				resolve({
					status: "error",
					data: xhr.statusText,
				});
			});
			xhr.setRequestHeader("Accept", "application/json");
			xhr.setRequestHeader("Authorization", getSession());
			xhr.send(formdata);
		});
		document.dispatchEvent(
			new CustomEvent<ProgressEventData>("kn-upload-update", {
				detail: {
					id: id,
					cntLoaded: 1,
					cntTotal: 1,
					computable: true,
					processing: false,
				},
			})
		);
		dismissToast(toast);
		return resp;
	} catch (e: any) {
		return { status: "error", data: e.toString() };
	}
}
