// Functions for contest

import { postCall } from "./net";
import { apiToast } from "./toast";

export async function registerForContest(contestID: number) {
	const res = await postCall(`/contest/${contestID}/register`, {});
	if (res.status === "error") {
		apiToast(res);
		return;
	}
	window.location.reload();
}
