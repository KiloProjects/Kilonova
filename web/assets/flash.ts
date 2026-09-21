const UNAUTHED_SUFFIX = "_dismissed:unauthed";

function flashKey(name: string): string {
	const uid = window.platform_info.user_id;
	return uid > 0 ? `${name}_dismissed:${uid}` : name + UNAUTHED_SUFFIX;
}

function isDismissed(name: string): boolean {
	if (localStorage.getItem(flashKey(name)) !== null) {
		return true;
	}
	// Dismissed before logging in, carry it over to the account.
	if (localStorage.getItem(name + UNAUTHED_SUFFIX) === null) {
		return false;
	}
	localStorage.removeItem(name + UNAUTHED_SUFFIX);
	localStorage.setItem(flashKey(name), "disabled");
	return true;
}

export function dismissFlash(name: string) {
	localStorage.setItem(flashKey(name), "disabled");
	document.querySelectorAll(`[data-flashname="${CSS.escape(name)}"]`).forEach((el) => el.remove());
}

export function initFlashes(el: HTMLElement) {
	el.querySelectorAll<HTMLElement>("[data-flashname]").forEach((flash) => {
		if (!isDismissed(flash.dataset.flashname!)) {
			flash.classList.remove("hidden");
		}
	});
	el.querySelectorAll<HTMLElement>("[data-flashdiscard]").forEach((btn) => {
		btn.addEventListener("click", () => dismissFlash(btn.dataset.flashdiscard!));
	});
}
