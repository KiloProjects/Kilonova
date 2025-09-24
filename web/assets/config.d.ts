import type { katex } from "katex";
import type { UserBrief } from "./api/submissions";

import type htmx from "htmx.org";

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
		language: "en" | "ro";
		langs: { [name: string]: string };
		sentryDSN?: string;
		internalVersion: string;
	}

	interface WebLanguage {
		disabled: boolean;
		name: string;
	}

	let bundled: typeof import("./app");
	let vendored: typeof import("./vendored");

	interface Window {
		platform_info: PlatformInfo;
		katex: katex;
		htmx: typeof htmx;
		bundled: typeof import("./app");
		vendored: typeof import("./vendored");
	}

	interface Document {
		//adds definition to Document, but you can do the same with HTMLElement
		addEventListener<K extends keyof CustomEventMap>(type: K, listener: (this: Document, ev: CustomEventMap[K]) => void): void;
		removeEventListener<K extends keyof CustomEventMap>(type: K, listener: (this: Document, ev: CustomEventMap[K]) => void): void;
	}
}
