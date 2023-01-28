import { vsprintf } from "sprintf-js";
import languageStrings from "./_translations.json";

type translation = {
	ro: string;
	en: string;
};

function getVal(key: string): translation | null {
	let current: any = languageStrings;
	let vals = key.split(".");
	if (current[vals[0]] === undefined) {
		console.warn(`key "${key}" does not exist`);
		return null;
	}
	for (const sub of vals) {
		if (current[sub]["ro"] !== undefined) {
			return current[sub];
		}
		current = current[sub];
	}
	return null;
}

export default function getText(key: string, ...args: any): string {
	const lang = window.platform_info.language;
	const translation = getVal(key);
	if (translation === null) {
		console.error("Unknown key", JSON.stringify(key));
		return "ERR";
	}
	if (lang in translation) {
		return vsprintf(translation[lang], args);
	}
	console.error("Language", lang, "not found in key", key);
	return "ERR";
}
