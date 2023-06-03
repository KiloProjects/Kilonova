import type { katex } from "katex";
import type { UserBrief } from "./api/submissions";

interface CustomEventMap {
	"kn-poll": CustomEvent;
	"kn-upload-update": CustomEvent;
	"kn-contest-question-reload": CustomEvent;
	"kn-contest-announcement-reload": CustomEvent;
}

export declare global {
	interface ProgressEventData {
		computable: boolean;
		cntLoaded: number;
		cntTotal: number;
		id: number; // To distinguish in case there are multiple uploads
		processing: boolean; // When request is uploaded and is awaiting results
	}

	interface PlatformInfo {
		debug: boolean;
		admin: boolean;
		user_id: number;
		user?: UserBrief;
		language: string;
		langs: { [name: string]: WebLanguage };
	}

	interface WebLanguage {
		disabled: boolean;
		name: string;
	}

	interface Window {
		platform_info: PlatformInfo;
		katex: katex;
	}
	interface Document {
		//adds definition to Document, but you can do the same with HTMLElement
		addEventListener<K extends keyof CustomEventMap>(type: K, listener: (this: Document, ev: CustomEventMap[K]) => void): void;
		removeEventListener<K extends keyof CustomEventMap>(type: K, listener: (this: Document, ev: CustomEventMap[K]) => void): void;
	}
}
