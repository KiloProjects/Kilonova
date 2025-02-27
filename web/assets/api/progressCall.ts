import { createToast, dismissToast } from "../toast";
import getText from "../translation";
import { Response, defaultClient } from "./client";

export async function multipartProgressCall<T = any>(call: string, formdata: FormData): Promise<Response<T>> {
	if (call.startsWith("/")) {
		call = call.substring(1);
	}
	try {
		const id = Math.random();
		const toast = createToast({
			status: "progress",
			title: getText("uploading"),
			description: `<upload-progress id="${id}"></upload-progress>`,
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
					statusCode: xhr.status,
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
					statusCode: xhr.status,
				});
			});
			xhr.setRequestHeader("Accept", "application/json");
			xhr.setRequestHeader("Authorization", defaultClient.session);
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
		return { status: "error", data: e.toString(), statusCode: 500 };
	}
}
