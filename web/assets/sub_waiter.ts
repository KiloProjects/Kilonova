import { getCall } from "./net";
import { createToast } from "./toast";
import getText from "./translation";

let loadingIDs = new Map<number, string>();

export function makeSubWaiter(id: number): string {
	if (loadingIDs.has(id)) {
		return `Will not watch ${id}`;
	}
	loadingIDs.set(id, "reserved");
	let interv = setInterval(async () => {
		let res = await getCall("/submissions/getByID", { id: id });
		if (res.status == "error") {
			console.error(res);
			return;
		}
		var lastStatus = loadingIDs.get(id);
		if (res.data.sub.status !== lastStatus) {
			document.dispatchEvent(new CustomEvent("kn-poll"));
		}
		if (res.data.sub.status == "finished") {
			createToast({
				title: getText("finishedEval"),
				description: getText("finalScore", id, res.data.sub.score),
				status: "success",
			});
			clearInterval(interv);
		}
		loadingIDs.set(id, res.data.sub.status);
	}, 500);
	return `Watching ${id}...`;
}
