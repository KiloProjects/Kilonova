import cookie from "js-cookie";
import { postCall } from "./api/client";
import { languages } from "./langs";

export function setLanguage(lang: "en" | "ro") {
	cookie.set("kn-lang", lang, { expires: 1000, sameSite: "lax", secure: true });
	window.location.reload();
}

export function clearLanguageCookie() {
	cookie.remove("kn-lang");
}

export function setSubmitStyle(style: "code" | "file") {
	cookie.set("kn-sub-style", style, { expires: 1000, sameSite: "lax", secure: true });
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

export function setTheme(theme: "light" | "dark") {
	cookie.set("kn-theme", theme, { expires: 1000, sameSite: "lax", secure: true });
	document.documentElement.classList.toggle("dark", theme === "dark");
	document.documentElement.classList.toggle("light", theme !== "dark");

	// if user is logged in, update default preference
	if (window.platform_info.user_id > 0) {
		postCall<string>("/user/self/setPreferredTheme", { theme })
			.then((res) => {
				if (res.status == "error") {
					console.error(res.data);
				}
				console.info("Updated user preference.");
			})
			.catch(console.error);
	}
	document.dispatchEvent(new CustomEvent("kn-theme-change", {detail: theme}))
}

// should be used by the navbar toggle
export function toggleTheme(e?: Event) {
	e?.preventDefault();
	const curr = getTheme();
	if (curr == "light") {
		setTheme("dark");
		document.getElementById("theme_button")!.innerHTML = `<i class="fas fa-fw fa-lightbulb"></i>`;
		document.getElementById("theme_button_mobile")!.innerHTML = `<i class="fas fa-fw fa-lightbulb"></i>`;
	} else {
		setTheme("light");
		document.getElementById("theme_button")!.innerHTML = `<i class="fas fa-fw fa-moon"></i>`;
		document.getElementById("theme_button_mobile")!.innerHTML = `<i class="fas fa-fw fa-moon"></i>`;
	}
}

// For saving submission language preference

export function getCodeLangPreference(): string {
	let val = cookie.get("kn-lang-pref");
	if (val == "" || typeof val === "undefined" || !Object.keys(window.platform_info.langs).includes(val)) {
		setCodeLangPreference("cpp17");
		return "cpp17";
	}
	return val;
}

export function setCodeLangPreference(lang: string) {
	if (!Object.keys(languages).includes(lang)) {
		console.warn("Skipping invalid lang preference", lang);
		return;
	}
	cookie.set("kn-lang-pref", lang, { expires: 1000, sameSite: "lax", secure: true });
}
