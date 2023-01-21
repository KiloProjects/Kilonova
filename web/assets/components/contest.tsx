import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import { useEffect, useState } from "preact/hooks";
import { dayjs } from "../util";
import getText from "../translation";
import { sprintf } from "sprintf-js";

export const RFC1123Z = "ddd, DD MMM YYYY HH:mm:ss ZZ";

export function contestToNetworkDate(timestamp: string): string {
	const djs = dayjs(timestamp, "YYYY-MM-DD HH:mm ZZ", true);
	if (!djs.isValid()) {
		throw new Error("Invalid timestamp");
	}
	return djs.format(RFC1123Z);
}

export function ContestRemainingTime({ target_time }: { target_time: dayjs.Dayjs }) {
	let [text, setText] = useState<string>("");

	function updateTime() {
		let diff = target_time.diff(dayjs(), "s");
		if (diff < 0) {
			console.log("Reloading webpage...");
			window.location.reload();
			return;
		}
		const seconds = diff % 60;
		diff = (diff - seconds) / 60;
		const minutes = diff % 60;
		diff = (diff - minutes) / 60;
		const hours = diff;

		if (hours >= 48) {
			// >2 days
			setText(getText("days", Math.floor(diff / 24)));
			return;
		}

		setText(sprintf("%02d:%02d:%02d", hours, minutes, seconds));
	}

	useEffect(() => {
		updateTime();
		const interval = setInterval(() => {
			updateTime();
		}, 500);
		return () => clearInterval(interval);
	}, []);

	return <span>{text}</span>;
}

export function ContestCountdown({ target_time, type }: { target_time: string; type: string }) {
	let timestamp = parseInt(target_time);
	if (isNaN(timestamp)) {
		console.error("unix timestamp is somehow NaN", target_time);
		return <>Invalid Timestamp</>;
	}
	const endTime = dayjs(timestamp);
	return (
		<>
			{endTime.diff(dayjs()) <= 0 ? (
				<span>{{ running: getText("contest_ended"), before_start: getText("contest_running") }[type]}</span>
			) : (
				<ContestRemainingTime target_time={endTime} />
			)}
		</>
	);
}

register(ContestCountdown, "kn-contest-countdown", ["target_time", "type"]);
