import {vsprintf} from 'sprintf-js';
import languageStrings from './_translations.json';

export function getText(key: string, ...args: any) {
	const lang = window.platform_info?.language || 'en';
	if(key in languageStrings) {
		if(lang in languageStrings[key]) {
			return vsprintf(languageStrings[key][lang], args)
		}
		console.error("Language", lang, "not found in key", key)
	}
	console.error("Unknown key", JSON.stringify(key))
}
