import { Notyf, INotyfOptions } from "notyf";

let notyfConf: INotyfOptions = {
	position: { x: "right", y: "bottom" },
	duration: 6000,
	ripple: false,
	dismissible: true,
	types: [
		{
			type: "info",
			background: "blue",
			icon: {
				className: "fas fa-info-circle",
				tagName: "i",
				color: "white",
			},
		},
		{
			type: "error",
			background: "red",
			icon: {
				className: "fas fa-exclamation-triangle",
				tagName: "i",
				color: "white",
			},
		},
		{
			type: "success",
			background: "green",
			icon: {
				className: "fas fa-check-circle",
				tagName: "i",
				color: "white",
			},
		},
	],
};

var notyf: Notyf | null = null;

window.addEventListener("load", () => {
	notyf = new Notyf(notyfConf);
});

interface ToastOptions {
	title?: string;
	description?: string;
	status?: "success" | "error" | "info";
}

/* createToast options
	title: the toast title
	description: the toast description
	status: the toast status (default "info", can be ["success", "error", "info"])
*/
export function createToast(options: ToastOptions) {
	if (notyf === null) {
		console.warn("createToast called before window load");
		return;
	}

	if (options.status == null) {
		options.status = "info";
	}

	let msg = "";
	if (
		options.title !== undefined &&
		options.title !== null &&
		options.title !== ""
	) {
		msg += `<h3>${options.title}</h3>`;
	}
	if (
		options.description !== undefined &&
		options.description !== null &&
		options.description !== ""
	) {
		msg += options.description;
	}

	return notyf.open({
		type: options.status,
		message: msg,
	});
}

import { APIResponse } from "./v2/requests";

export function apiToast(res: APIResponse, overwrite?: ToastOptions) {
	if (overwrite === null || overwrite === undefined) {
		overwrite = {};
	}
	overwrite["status"] = res.status;
	overwrite["description"] = res.data;
	createToast(overwrite);
}
