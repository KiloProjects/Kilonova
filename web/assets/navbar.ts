function hasParentWithID(element: Element, id: string): boolean {
	for (let p = element && element.parentElement; p; p = p.parentElement) {
		if (p.id == id) {
			return true;
		}
	}
	return false;
}

export class NavBarManager {
	isNavbarOpen: boolean = false;
	isDropdownOpen: boolean = false;

	constructor() {
		this.isNavbarOpen = false;
		this.isDropdownOpen = false;

		this.setNavbar(false);
		this.setDropdown(false);

		document.addEventListener("keydown", (e) => this.checkKey(e));
		document.addEventListener("click", (e) => this.checkClick(e));
	}

	checkKey(e: KeyboardEvent) {
		if (e.key === "Esc" || e.key === "Escape") {
			this.setDropdown(false);
		}
	}

	checkClick(e: MouseEvent) {
		if (this.isDropdownOpen && e.target instanceof Element && !(e.target.id === "profile-dropdown-button" || hasParentWithID(e.target, "pr-dropdown"))) {
			this.setDropdown(false);
		}
	}

	toggleNavbar() {
		this.setNavbar(!this.isNavbarOpen);
	}

	toggleDropdown() {
		this.setDropdown(!this.isDropdownOpen);
	}

	setNavbar(open) {
		this.isNavbarOpen = open;
		document.getElementById("nav-toggler")?.classList.toggle("fa-times", this.isNavbarOpen);
		document.getElementById("nav-toggler")?.classList.toggle("fa-bars", !this.isNavbarOpen);
		document.getElementById("nav-dropdown")?.classList.toggle("block", this.isNavbarOpen);
		document.getElementById("nav-dropdown")?.classList.toggle("hidden", !this.isNavbarOpen);
	}

	setDropdown(open) {
		this.isDropdownOpen = open;
		document.getElementById("profile-dropdown")?.classList.toggle("hidden", !this.isDropdownOpen);
		document.getElementById("dropdown-caret")?.classList.toggle("fa-caret-down", !this.isDropdownOpen);
		document.getElementById("dropdown-caret")?.classList.toggle("fa-caret-up", this.isDropdownOpen);
	}
}
