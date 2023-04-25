import { h, Fragment, render } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";
import { createToast, apiToast } from "../toast";
import { BigSpinner } from "./common";
import { Problem, SubTest, SubmissionSubTask } from "../api/submissions";
import { getCall } from "../net";
import { SubTask } from "./sub_mgr";

type BreakdownResult = {
	max_score: number;
	problem: Problem;
	subtasks: SubmissionSubTask[];
	subtests: SubTest[];
	problem_editor: boolean;
};

export function MaxScoreBreakdown({ problemID, userID, contestID }: { problemID: number; userID?: number; contestID?: number }) {
	let [subtasks, setSubtasks] = useState<SubmissionSubTask[]>([]);
	let [subtests, setSubtests] = useState<SubTest[] | null>(null);
	let [loading, setLoading] = useState(true);
	let [problem, setProblem] = useState<Problem | null>(null);
	let [maxScore, setMaxScore] = useState(-1);
	let [problemEditor, setProblemEditor] = useState(false);

	async function loadBreakdown() {
		const rez = await getCall<BreakdownResult>(`/problem/${problemID}/maxScoreBreakdown`, { userID, contestID });
		if (rez.status === "error") {
			apiToast(rez);
			return;
		}
		setMaxScore(rez.data.max_score);
		setProblem(rez.data.problem);
		setSubtasks(rez.data.subtasks);
		setSubtests(rez.data.subtests);
		setProblemEditor(rez.data.problem_editor);
	}

	useEffect(() => {
		setLoading(true);
		loadBreakdown()
			.then(() => setLoading(false))
			.catch(console.error);
	}, [problemID, userID, contestID]);

	if (!(subtests != null && problem != null && !loading)) {
		return <BigSpinner />;
	}

	let content = <>{getText("score_breakdown_unattempted")}</>;
	if (maxScore >= 0) {
		content = (
			<>
				<div class="list-group mb-2">
					{subtasks.map((subtask) => (
						<SubTask subtests={subtests ?? []} problem_editor={problemEditor} subtask={subtask} breakdown_mode={true} key={"stk_" + subtask.id} />
					))}
				</div>
			</>
		);
	}

	return (
		<>
			<h2>
				{getText("problemSingle")}{" "}
				<a href={`${typeof contestID !== "undefined" ? `/contests/${contestID}` : ""}/problems/${problem.id}`}>{problem.name}</a>:{" "}
				{maxScore >= 0 ? `${maxScore}p` : `0p`}
			</h2>

			{content}
		</>
	);
}

function KNModal({ open, title, children }: { open: boolean; title: string; children: preact.ComponentChildren }) {
	let [lastState, setLastState] = useState<boolean | null>(null);
	let ref = useRef<HTMLDialogElement>(null);

	useEffect(() => {
		if (open == lastState) {
			return;
		}

		if (open) {
			ref.current?.showModal();
		} else {
			ref.current?.close();
		}

		setLastState(open);
		return () => {
			ref.current?.close();
		};
	}, [open]);

	return (
		<dialog ref={ref} class="modal-container" id="max_score_dialog">
			<div class="modal-header">
				<h1>{title}</h1>
				<form method="dialog">
					<button type="submit">
						<i class="modal-close"></i>
					</button>
				</form>
			</div>
			<div class="modal-content" id="max_score_content">
				{children}
			</div>
		</dialog>
	);
}

function MaxScoreBreakdownDOM({ problemid, userid, contestid }: { problemid: string; userid?: string; contestid?: string }) {
	const problemID = parseInt(problemid);
	if (isNaN(problemID)) {
		throw new Error("Invalid problem ID");
	}
	const userID = parseInt(userid ?? "asdf");
	const contestID = parseInt(contestid ?? "asdf");
	return (
		<KNModal open={true} title={getText("score_breakdown_title")}>
			<MaxScoreBreakdown
				problemID={problemID}
				contestID={!isNaN(contestID) ? contestID : undefined}
				userID={!isNaN(userID) ? userID : undefined}
			></MaxScoreBreakdown>
		</KNModal>
	);
}

register(MaxScoreBreakdownDOM, "kn-score-breakdown", ["problemid", "userid", "contestid"]);

document.addEventListener("DOMContentLoaded", () => {
	const modals = document.getElementById("modals")!;

	function buildModal(problemID, contestID) {
		const val = document.getElementById("max_score_preact");
		if (val != null) {
			modals.removeChild(val);
		}
		const newVal = document.createElement("kn-score-breakdown");
		newVal.id = "max_score_preact";
		newVal.setAttribute("problemid", problemID);
		if (typeof contestID !== "undefined") {
			newVal.setAttribute("contestid", contestID);
		}

		modals.appendChild(newVal);
	}

	Array.from(document.getElementsByClassName("max_score_breakdown")).forEach((val) => {
		val.addEventListener("click", (e) => {
			e.preventDefault();
			if (e.currentTarget == null) {
				return;
			}

			let problemID = parseInt((e.currentTarget as HTMLElement).dataset.problemid ?? "asd");
			if (isNaN(problemID)) {
				apiToast({ status: "error", data: "Invalid problem id for max score modal" });
				return;
			}
			let contestID: number | undefined = parseInt((e.currentTarget as HTMLElement).dataset.contestid ?? "asd");
			if (isNaN(contestID)) {
				contestID = undefined;
			}
			buildModal(problemID, contestID);
		});
	});
});
