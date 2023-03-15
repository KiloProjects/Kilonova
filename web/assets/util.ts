import dayjs from "dayjs";
// import "dayjs/locale/ro";
import relativeTime from "dayjs/plugin/relativeTime";
import customParseFormat from "dayjs/plugin/customParseFormat";

dayjs.extend(relativeTime);
dayjs.extend(customParseFormat);

/*
dayjs.locale("ro");

document.addEventListener("DOMContentLoaded", () => {
	dayjs.locale(window.platform_info.language);
});
*/

export { dayjs };

export { fromBase64 } from "js-base64";

export const JSONTimestamp = "YYYY-MM-DDTHH:mm:ss.SSSSSSZ";

// if max_step is 0, it will format until the MB region
// else if max_step is 1, it will only format until the KB region
// else if max_step >= 2, it will append a " B" to the end of the number
export function sizeFormatter(size: number, max_step?: number, floor?: boolean) {
	var units: number = size,
		suffix: string = "B";
	if (max_step === undefined) {
		max_step = 0;
	}
	if (size > 1024 * 1024 && max_step == 0) {
		units = Math.round((size / 1024 / 1024) * 1e2) / 1e2;
		suffix = "MB";
	} else if (size > 1024 && max_step <= 1) {
		units = Math.round((size / 1024) * 1e2) / 1e2;
		suffix = "KB";
	}
	if (floor) {
		units = Math.floor(units);
	}
	return units + " " + suffix;
}

export function downloadBlob(blob: Blob, filename: string) {
	var a = document.createElement("a");
	a.href = URL.createObjectURL(blob);
	a.download = filename;
	a.click();
}

export function parseTime(str?: string | number, extended?: boolean) {
	if (!str) {
		return "";
	}
	return dayjs(str).format("DD/MM/YYYY HH:mm" + (extended ? ":ss" : ""));
}

export function getGradient(score: number, maxscore: number) {
	let col = "#e3dd71",
		rap = 0.0;
	if (maxscore != 0 && score != 0) {
		rap = score / maxscore;
	}
	if (rap == 1.0) {
		col = "#7fff00";
	}
	if (rap < 1.0) {
		col = "#67cf39";
	}
	if (rap <= 0.8) {
		col = "#9fdd2e";
	}
	if (rap <= 0.6) {
		col = "#d2eb19";
	}
	if (rap <= 0.4) {
		col = "#f1d011";
	}
	if (rap <= 0.2) {
		col = "#f6881e";
	}
	if (rap == 0) {
		col = "#f11722";
	}
	return col;
}

export function stringIntToNumber(ints: string[]): number[] {
	let result: number[] = [];
	for (let val of ints.map((val) => parseInt(val))) {
		if (!isNaN(val)) {
			result.push(val);
		}
	}
	return result;
}

document.addEventListener("DOMContentLoaded", () => {
	document.querySelectorAll(".server_timestamp").forEach((val) => {
		let timestamp = parseInt(val.innerHTML);
		if (isNaN(timestamp)) {
			console.warn("NaN timestamp");
			return;
		}
		val.innerHTML = parseTime(timestamp);
		val.classList.remove("server_timestamp");
	});
});
