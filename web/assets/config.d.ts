interface CustomEventMap {
	"kn-poll": CustomEvent;
	"kn-upload-update": CustomEvent;
}

export declare global {
	interface ProgressEventData {
		computable: boolean;
		cntLoaded: number;
		cntTotal: number;
		id: number; // To distinguish in case there are multiple uploads
	}

	interface PlatformInfo {
		debug: boolean;
		admin: boolean;
		user_id: number;
		language: string;
	}

	interface Window {
		platform_info?: PlatformInfo;
		hljs: any;
	}
	interface Document {
		//adds definition to Document, but you can do the same with HTMLElement
		addEventListener<K extends keyof CustomEventMap>(
			type: K,
			listener: (this: Document, ev: CustomEventMap[K]) => void
		): void;
		removeEventListener<K extends keyof CustomEventMap>(
			type: K,
			listener: (this: Document, ev: CustomEventMap[K]) => void
		): void;
	}
}
