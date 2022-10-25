export declare global {
	interface PlatformInfo {
		debug: boolean,
		admin: boolean,
		user_id: number,
		language: string,
	}

	interface Window {
		platform_info?: PlatformInfo
	}
}
