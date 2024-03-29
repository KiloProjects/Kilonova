import { getCall } from "./api/client";
import { formatScoreStr, icpcVerdictString } from "./components";
import { createToast } from "./toast";
import getText from "./translation";

let loadingIDs = new Map<number, string>();

export function makeSubWaiter(id: number): string {
	if (loadingIDs.has(id)) {
		return `Will not watch ${id}`;
	}
	loadingIDs.set(id, "reserved");
	let interv = setInterval(async () => {
		let res = await getCall<FullSubmission>("/submissions/getByID", { id: id });
		if (res.status == "error") {
			console.error(res);
			return;
		}
		var lastStatus = loadingIDs.get(id);
		if (res.data.status !== lastStatus) {
			document.dispatchEvent(new CustomEvent("kn-poll"));
		}
		if (res.data.status == "finished") {
			let rezStr = "";
			let statusVal: "success" | "error" = "success";
			if (res.data.submission_type == "classic") {
				rezStr = getText("finalScore", id) + " " + formatScoreStr(res.data.score.toFixed(res.data.score_precision));
			} else {
				if (res.data.score == 100) {
					rezStr = `<i class="fas fa-fw fa-check"></i> ${getText("accepted")}`;
				} else {
					statusVal = "error";
					rezStr = res.data.icpc_verdict ? icpcVerdictString(res.data.icpc_verdict) : getText("rejected");
				}
			}

			createToast({
				title: getText("finishedEval"),
				description: rezStr,
				status: statusVal,
			});
			clearInterval(interv);
		}
		loadingIDs.set(id, res.data.status);
	}, 500);
	return `Watching ${id}...`;
}
