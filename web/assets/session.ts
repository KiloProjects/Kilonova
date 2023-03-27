import cookie from "js-cookie";
import { postCall } from "./net";
import dayjs from "dayjs";
import type { Editor } from "codemirror";
import { languages } from "./langs";

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

export function getTheme(): "light" | "dark" {
	let val = cookie.get("kn-theme");
	if (val == "" || typeof val === "undefined" || (val !== "light" && val !== "dark")) {
		setTheme("dark");
		return "dark";
	}
	return val;
}

export function isDarkMode() {
	return getTheme() == "dark";
}

var editors: Editor[] = [];

export function CodeMirrorThemeHook(cm: Editor) {
	editors.push(cm);
}

export function setTheme(theme: "light" | "dark") {
	cookie.set("kn-theme", theme, { expires: 1000, sameSite: "lax" });
	document.documentElement.classList.toggle("dark", theme === "dark");

	// if user is logged in, update default preference
	if (window.platform_info.user_id > 0) {
		postCall<string>("/user/setPreferredTheme", { theme })
			.then((res) => {
				if (res.status == "error") {
					console.error(res.data);
				}
				console.info("Updated user preference.");
			})
			.catch(console.error);
	}
	for (let cm of editors) {
		cm.setOption("theme", theme === "dark" ? "monokai" : "default");
	}
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

// should be used by the navbar toggle
export function toggleTheme(e?: Event) {
	e?.preventDefault();
	const curr = getTheme();
	if (curr == "light") {
		setTheme("dark");
		document.getElementById("theme_button")!.innerHTML = `<i class="fas fa-fw fa-sun"></i>`;
		document.getElementById("theme_button_mobile")!.innerHTML = `<i class="fas fa-fw fa-sun"></i>`;
	} else {
		setTheme("light");
		document.getElementById("theme_button")!.innerHTML = `<i class="fas fa-fw fa-moon"></i>`;
		document.getElementById("theme_button_mobile")!.innerHTML = `<i class="fas fa-fw fa-moon"></i>`;
	}
}

// For saving submission language preference

export function getCodeLangPreference(): string {
	let val = cookie.get("kn-lang-pref");
	if (val == "" || typeof val === "undefined" || !Object.keys(languages).includes(val)) {
		setCodeLangPreference("cpp14");
		return "cpp14";
	}
	return val;
}

export function setCodeLangPreference(lang: string) {
	if (!Object.keys(languages).includes(lang)) {
		console.warn("Skipping invalid lang preference", lang);
		return;
	}
	cookie.set("kn-lang-pref", lang, { expires: 1000, sameSite: "lax" });
}
