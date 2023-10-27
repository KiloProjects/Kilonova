import { getCall } from "./api/client";
import { icpcVerdictString } from "./components";
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
			if (res.data.submission_type == "classic") {
				rezStr = getText("finalScore", id) + " " + res.data.score.toFixed(res.data.score_precision);
			} else {
				if (res.data.icpc_verdict) {
					rezStr = icpcVerdictString(res.data.icpc_verdict);
				} else {
					if (res.data.score == 100) {
						rezStr = getText("accepted");
					} else {
						rezStr = getText("rejected");
					}
				}
			}

			createToast({
				title: getText("finishedEval"),
				description: rezStr,
				status: "success",
			});
			clearInterval(interv);
		}
		loadingIDs.set(id, res.data.status);
	}, 500);
	return `Watching ${id}...`;
}
