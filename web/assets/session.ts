import cookie from "js-cookie";
import { postCall } from "./net";
import dayjs from "dayjs";
import { JSONTimestamp } from "./util";

export function setSession(sessionID: string) {
	const checkTimestamp = dayjs().add(10, "day").unix() * 1000;
	cookie.set("kn-session-check-date", checkTimestamp.toString(), { expires: 29, sameSite: "lax" });
	cookie.set("kn-sessionid", sessionID, { expires: 29, sameSite: "lax" });
}

export function getSession(): string {
	return cookie.get("kn-sessionid") || "guest";
}

async function extendSession() {
	const res = await postCall<string>("/auth/extendSession", {});
	if (res.status === "error") {
		return;
	}

	const checkTimestamp = dayjs().add(10, "day").unix() * 1000;
	cookie.set("kn-session-check-date", checkTimestamp.toString(), { expires: 29, sameSite: "lax" });
}

export function setLanguage(lang: "en" | "ro") {
	cookie.set("kn-lang", lang, { expires: 1000, sameSite: "lax" });
	window.location.reload();
}

export function clearLanguageCookie() {
	cookie.remove("kn-lang");
}

export function setSubmitStyle(style: "code" | "file") {
	cookie.set("kn-sub-style", style, { expires: 1000, sameSite: "lax" });
}

export function getSubmitStyle(): "code" | "file" {
	let val = cookie.get("kn-sub-style");
	if (val == "" || typeof val === "undefined" || (val !== "code" && val !== "file")) {
		setSubmitStyle("code");
		return "code";
	}
	return val;
}

document.addEventListener("DOMContentLoaded", () => {
	const checkCookie = cookie.get("kn-session-check-date");
	if (typeof checkCookie == "undefined" || checkCookie === "") {
		if (getSession() != "guest") {
			extendSession();
		}
		return;
	}
	const val = parseInt(checkCookie);
	if (isNaN(val)) {
		return;
	}
	const checkTime = dayjs(val);
	if (dayjs(val).isBefore(dayjs())) {
		extendSession();
	}
});
