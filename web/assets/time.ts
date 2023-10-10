import dayjs, { Dayjs } from "dayjs";
import getText from "./translation";
import { sprintf } from "sprintf-js";

let diffMilli: number = -1;

export function initTime(nowTime: Dayjs, serverTime: Dayjs) {
	diffMilli = nowTime.diff(serverTime, "milliseconds", true);
	console.log(`Client-Server time difference: ${diffMilli}ms`);
}

export function serverTime(): Dayjs {
	return dayjs().subtract(diffMilli, "millisecond");
}

// targetTime can be:
//   - a Dayjs object, in which case the duration is computed at the moment of the function call
//   - a number, representing the number of seconds to format
export function formatDuration(targetTime: Dayjs | number, hideSeconds: boolean = false, showFull: boolean = false): string {
	// NOTE: dayjs diff must take into consideration diffMilli from initTime (to compute server duration)
	// If user has clock forwards or backwards he will get the wrong time in this current implementation.
	let diff = -1;
	if (typeof targetTime == "number") {
		diff = targetTime;
	} else {
		diff = targetTime.diff(serverTime(), "s");
	}
	if (diff < 0) {
		return getText("time_elapsed");
	}
	const seconds = diff % 60;
	diff = (diff - seconds) / 60;
	const minutes = diff % 60;
	diff = (diff - minutes) / 60;
	const hours = diff;
	if (hours >= 48 && !showFull) {
		// >2 days
		return getText("days", Math.floor(diff / 24));
	}
	if (hours >= 24 && showFull) {
		const hours = diff % 24;
		diff = (diff - hours) / 24;
		const days = diff;
		return sprintf("%02d:%02d:%02d", days, hours, minutes) + (hideSeconds ? "" : sprintf(":%02d", seconds));
	}
	return sprintf("%02d:%02d", hours, minutes) + (hideSeconds ? "" : sprintf(":%02d", seconds));
}

// Formats datetime-local input into an ISO3601 timezone-aware string
export function formatISO3601(timestamp: string): string {
	const djs = dayjs(timestamp, "YYYY-MM-DDTHH:mm", true);
	if (!djs.isValid()) {
		throw new Error("Invalid timestamp");
	}
	return djs.format("YYYY-MM-DDTHH:mm:ss.SSSZ");
}

setInterval(() => {
	const el = document.getElementById("footer_server_time");
	if (el != null) {
		el.innerText = serverTime().format("HH:mm:ss");
	}
}, 500);
