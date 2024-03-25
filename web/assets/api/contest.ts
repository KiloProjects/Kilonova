// Functions for contest

import { getCall, postCall } from "./client";
import { apiToast } from "../toast";

export async function registerForContest(contestID: number) {
	const res = await postCall(`/contest/${contestID}/register`, {});
	if (res.status === "error") {
		apiToast(res);
		return;
	}
	window.location.reload();
}

export async function startContestRegistration(contestID: number) {
	const res = await postCall(`/contest/${contestID}/startRegistration`, {});
	if (res.status === "error") {
		apiToast(res);
		return;
	}
	if (window.location.pathname.startsWith(`/contests/${contestID}`)) {
		window.location.reload();
	} else {
		window.location.assign(`/contests/${contestID}`);
	}
}

export async function answerQuestion(q: Question, text: string) {
	let res = await postCall(`/contest/${q.contest_id}/answerQuestion`, { questionID: q.id, text });
	apiToast(res);
	if (res.status === "success") {
		reloadQuestions();
		return;
	}
}

export async function updateAnnouncement(ann: Announcement, text: string) {
	let res = await postCall(`/contest/${ann.contest_id}/updateAnnouncement`, { id: ann.id, text });
	apiToast(res);
	if (res.status === "success") {
		reloadAnnouncements();
		return;
	}
}

export async function deleteAnnouncement(ann: Announcement) {
	let res = await postCall(`/contest/${ann.contest_id}/deleteAnnouncement`, { id: ann.id });
	apiToast(res);
	if (res.status === "success") {
		reloadAnnouncements();
		return;
	}
}

export async function getAllQuestions(contestID: number): Promise<Question[]> {
	let res = await getCall<Question[]>(`/contest/${contestID}/allQuestions`, {});
	if (res.status === "error") {
		if (res.statusCode == 401) {
			apiToast({
				status: "error",
				data: "Error reloading questions (possibly logged out), please report this to the developers. Automatic verification will be disabled for this tab.",
			});
			stopReloadingQnA();
			return [];
		}
		throw new Error(res.data);
	}
	return res.data;
}

export async function getUserQuestions(contestID: number): Promise<Question[]> {
	if (window.platform_info.user_id <= 0) {
		// Skip if unauthed
		return [];
	}
	let res = await getCall<Question[]>(`/contest/${contestID}/questions`, {});
	if (res.status === "error") {
		if (res.statusCode == 401) {
			apiToast({
				status: "error",
				data: "Error reloading questions (possibly logged out), please report this to the developers. Automatic verification will be disabled for this tab.",
			});
			stopReloadingQnA();
			return [];
		}
		throw new Error(res.data);
	}
	return res.data;
}

export async function getAnnouncements(contestID: number): Promise<Announcement[]> {
	let res = await getCall<Announcement[]>(`/contest/${contestID}/announcements`, {});
	if (res.status === "error") {
		if (res.statusCode == 401) {
			apiToast({
				status: "error",
				data: "Error reloading announcements (possibly logged out), please report this to the developers. Automatic verification will be disabled for this tab.",
			});
			stopReloadingQnA();
			return [];
		}
		throw new Error(res.data);
	}
	return res.data;
}

export function reloadQuestions() {
	document.dispatchEvent(new CustomEvent("kn-contest-question-reload"));
}

export function reloadAnnouncements() {
	document.dispatchEvent(new CustomEvent("kn-contest-announcement-reload"));
}

var x: number | undefined;

export function startReloadingQnA(interval_ms: number) {
	x = setInterval(() => {
		reloadQuestions();
		reloadAnnouncements();
	}, interval_ms);
}

export function stopReloadingQnA() {
	console.warn("Disabling question/answer reloading");
	clearInterval(x);
}
