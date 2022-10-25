import {vsprintf} from 'sprintf-js';
import languageStrings from './_translations.json';

type translation = {
	ro: string;
	en: string;
}

function getVal(key: string): translation|null {
	let current: any = languageStrings;
	let vals = key.split('.');
	for(const sub of vals) {
		if(current[sub]["ro"] !== undefined) {
			return current[sub]
		}
		current = current[sub]
	}
	return null 
}

export function getText(key: string, ...args: any) {
	const lang = window.platform_info?.language || 'en';
	const translation = getVal(key)
	if(translation === null) {
		console.error("Unknown key", JSON.stringify(key))
		return
	}
	if(lang in translation) {
		return vsprintf(languageStrings[key][lang], args)
	}
	console.error("Language", lang, "not found in key", key)
}

export default getText;
