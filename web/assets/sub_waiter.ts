import { FullSubmission } from "./api/submissions";
import { getCall } from "./api/net";
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
			createToast({
				title: getText("finishedEval"),
				description: getText("finalScore", id, res.data.score),
				status: "success",
			});
			clearInterval(interv);
		}
		loadingIDs.set(id, res.data.status);
	}, 500);
	return `Watching ${id}...`;
}
