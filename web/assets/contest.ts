// Functions for contest

import { getCall, postCall } from "./net";
import { apiToast } from "./toast";

export type Question = {
	id: number;
	asked_at: string;
	responded_at?: string;
	text: string;
	response?: string;
	author_id: number;
	contest_id: number;
};

export type Announcement = {
	id: number;
	created_at: string;
	contest_id: number;
	text: string;
};

export async function registerForContest(contestID: number) {
	const res = await postCall(`/contest/${contestID}/register`, {});
	if (res.status === "error") {
		apiToast(res);
		return;
	}
	window.location.reload();
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
		throw new Error(res.data);
	}
	return res.data;
}

export async function getUserQuestions(contestID: number): Promise<Question[]> {
	let res = await getCall<Question[]>(`/contest/${contestID}/questions`, {});
	if (res.status === "error") {
		throw new Error(res.data);
	}
	return res.data;
}

export async function getAnnouncements(contestID: number): Promise<Announcement[]> {
	let res = await getCall<Announcement[]>(`/contest/${contestID}/announcements`, {});
	if (res.status === "error") {
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
