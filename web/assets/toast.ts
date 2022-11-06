import { Notyf, INotyfOptions, NotyfNotification } from "notyf";

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
		{
			type: "progress",
			background: "blue",
			icon: false,
			dismissible: false,
			duration: 10000000,
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
	status?: "success" | "error" | "info" | "progress";
}

/* createToast options
	title: the toast title
	description: the toast description
	status: the toast status (default "info", can be ["success", "error", "info"])
*/
export function createToast(options: ToastOptions): NotyfNotification {
	if (notyf === null) {
		throw new Error("createToast called before window load");
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

export interface APIResponse {
	status: "success" | "error";
	data: any;
}

export function apiToast(
	res: APIResponse,
	overwrite?: ToastOptions
): NotyfNotification {
	if (overwrite === null || overwrite === undefined) {
		overwrite = {};
	}
	overwrite["status"] = res.status;
	overwrite["description"] = res.data;
	return createToast(overwrite);
}

export function dismissToast(toast: NotyfNotification) {
	notyf?.dismiss(toast);
}
