import { h } from "preact";

import getText, { maybeGetText } from "../translation";

import { formatScoreStr } from "./common";

import { sizeFormatter, getGradient } from "../util";

function testVerdictString(verdict: string): string | h.JSX.Element {
	let txt = verdict
		.replace(/translate:([a-z_]+)/g, (substr, p1) => {
			return maybeGetText("test_verdict." + p1);
		})
		.trim();
	if (txt.split("\n").length == 1) {
		return txt;
	}
	console.log("Multiline verdict");
	return (
		<>
			{txt.split("\n").map((val, i, arr) => (
				<>
					{val}
					{i != arr.length - 1 && <br />}
				</>
			))}
		</>
	);
}

export function icpcVerdictString(verdict: string): string {
	return verdict.replace(/test_verdict.([a-z_]+)/g, (substr, p1) => {
		return maybeGetText("test_verdict." + p1);
	});
}

// If subtask is not null, then it's inside a subtask view, so filter and show tests only for that subtask
export function TestTable({
	subtests,
	subtasks,
	problem_editor,
	subtask,
	precision,
	subType = "classic",
}: {
	subtests: SubTest[];
	subtasks: SubmissionSubTask[];
	problem_editor: boolean;
	subtask?: SubmissionSubTask;
	precision: number;
	subType: "classic" | "acm-icpc";
}) {
	function testSubTasks(subtestID) {
		let stks: number[] = [];
		for (let st of subtasks) {
			if (st.subtests.includes(subtestID)) {
				stks.push(st.visible_id);
			}
		}
		return stks;
	}
	return (
		<table class={`kn-table ${typeof subtask !== "undefined" ? "default-background" : ""} mb-2`}>
			<thead>
				<tr>
					<th class="py-2" scope="col">
						{getText("id")}
					</th>
					<th scope="col">{getText("time")}</th>
					<th scope="col">{getText("memory")}</th>
					<th scope="col">{getText("verdict")}</th>
					{subType == "classic" && (
						<>
							<th scope="col">{getText("score")}</th>
							{subtasks.length > 0 && <th scope="col">{getText("subTasks")}</th>}
						</>
					)}
					{problem_editor && <th scope="col">{getText("output")}</th>}
				</tr>
			</thead>
			<tbody>
				{subtests
					.filter((subtest) => typeof subtask === "undefined" || subtask.subtests.includes(subtest.id))
					.sort((a, b) => a.visible_id - b.visible_id)
					.map((subtest) => {
						let maxScore = subtest.score;
						if (typeof subtask !== "undefined") {
							maxScore = subtask.score;
						}
						return (
							<tr class="kn-table-row" key={"kn_test" + subtest.id}>
								<th class="py-1" scope="row" id={`test-${subtest.visible_id}`}>
									{subtest.visible_id}
								</th>
								{subtest.skipped ? (
									<>
										<td>-</td>
										<td>-</td>
										<td>{testVerdictString(subtest.verdict)}</td>
										{subType == "classic" && <td>-</td>}
									</>
								) : subtest.done ? (
									<>
										<td>{Math.floor(subtest.time * 1000)} ms</td>
										<td>{sizeFormatter(subtest.memory * 1024, 1, true)}</td>
										<td>{testVerdictString(subtest.verdict)}</td>
										{subType == "classic" && (
											<td class="text-black" style={{ backgroundColor: getGradient(subtest.percentage, 100) }}>
												{subtasks.length > 0 ? (
													<>
														{subtest.percentage}% {getText("correct")}
													</>
												) : (
													<>
														{formatScoreStr((maxScore * (subtest.percentage / 100.0)).toFixed(precision))} /{" "}
														{formatScoreStr(maxScore.toFixed(precision))}
													</>
												)}
											</td>
										)}
									</>
								) : (
									<>
										<td></td>
										<td></td>
										<td>
											<div class="fas fa-spinner animate-spin" role="status"></div> {getText("waiting")}
										</td>
										{subType == "classic" && <td>-</td>}
									</>
								)}
								{subType == "classic" && subtasks.length > 0 && <td>{testSubTasks(subtest.id).join(", ")}</td>}
								{problem_editor && (
									<td>
										<a href={"/assets/subtest/" + subtest.id}>{getText("output")}</a>
									</td>
								)}
							</tr>
						);
					})}
			</tbody>
		</table>
	);
}

export function SubTask({
	subtests,
	subtask,
	problem_editor,
	breakdown_mode,
	precision,
	defaultOpen,
}: {
	subtests: SubTest[];
	subtask: SubmissionSubTask;
	problem_editor: boolean;
	breakdown_mode: boolean;
	precision: number;
	defaultOpen?: boolean;
}) {
	return (
		<details id={`stk-det-${subtask.visible_id}`} class="list-group-item" open={defaultOpen}>
			<summary class="pb-1 mt-1">
				<span>
					{getText("nthSubTask", subtask.visible_id)}{" "}
					{breakdown_mode && (
						<>
							({getText("from_sub")} <a href={`/submissions/${subtask.submission_id}`}>#{subtask.submission_id}</a>)
						</>
					)}
				</span>
				{typeof subtask.final_percentage !== "undefined" ? (
					<span class="float-right badge" style={{ backgroundColor: getGradient(subtask.final_percentage, 100) }}>
						{formatScoreStr((subtask.score * (subtask.final_percentage / 100.0)).toFixed(precision))} /{" "}
						{formatScoreStr(subtask.score.toFixed(precision))}
					</span>
				) : (
					<span class="float-right badge">
						<i class="fas fa-cog animate-spin"></i>
					</span>
				)}
			</summary>
			<TestTable subtests={subtests} subtask={subtask} problem_editor={problem_editor} subtasks={[]} precision={precision} subType="classic" />
		</details>
	);
}