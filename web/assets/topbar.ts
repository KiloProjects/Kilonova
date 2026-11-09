import { startReloadingQnA } from "./api/contest";

// Behaviors for the server-rendered topbars (web/ui/tabs.go, web/ui/problems/topbar.go).
export function initTopbar(el: HTMLElement) {
	// ui.CollapsedTabs
	el.querySelectorAll<HTMLElement>("[data-tabs-collapsible]").forEach((group) => {
		const toggle = group.querySelector<HTMLElement>("[data-tabs-toggle]")!;
		const panel = group.querySelector<HTMLElement>("[data-tabs-panel]")!;
		let pinned = "tabsOpen" in group.dataset;
		const show = (open: boolean) => {
			toggle.classList.toggle("hidden", open);
			panel.classList.toggle("hidden", !open);
		};
		if (pinned) return;
		toggle.addEventListener("click", (e) => {
			e.preventDefault();
			pinned = true;
			show(true);
		});
		group.addEventListener("mouseenter", () => show(true));
		(group.closest("nav") ?? group).addEventListener("mouseleave", () => show(pinned));
	});

	// Running contest: poll questions/announcements.
	el.querySelectorAll<HTMLElement>("[data-qna-poll]").forEach((x) => {
		startReloadingQnA(parseInt(x.dataset.qnaPoll!));
		delete x.dataset.qnaPoll;
	});

	// Ended contest: link to the same page outside the contest.
	el.querySelectorAll<HTMLAnchorElement>("a[data-strip-contest]").forEach((a) => {
		a.href = window.location.href.replace(/contests\/[0-9]+\//, "");
	});
}
