import { h, Fragment, render, Component } from "preact";
import { useState } from "preact/hooks";
import register from "preact-custom-element";
import { prettyLanguages } from "../langs";

import getText, { maybeGetText } from "../translation";
import { createToast, apiToast } from "../toast";
const slugify = (str) =>
	str
		.toLowerCase()
		.trim()
		.replace(/[^\w\s-]/g, "")
		.replace(/[\s_-]+/g, "-")
		.replace(/^-+|-+$/g, "");

import { BigSpinner, OlderSubmissions } from "./common";

import { downloadBlob, parseTime, sizeFormatter, getGradient, fromBase64 } from "../util";
import { defaultClient } from "../api/client";

function downloadCode(sub: FullSubmission) {
	if (typeof sub.code === "undefined") {
		console.error("Trying to download code when it isn't available");
		return;
	}
	var file = new Blob([sub.code], { type: "text/plain" });
	var filename = `${slugify(sub.problem.name)}-${sub.id}.${sub.language.replace(/[0-9]+$/g, "").replace("outputOnly", "txt")}`;
	downloadBlob(file, filename);
}

async function copyCode(sub: Submission) {
	if (typeof sub.code === "undefined") {
		console.error("Trying to copy code when it isn't available");
		return;
	}
	await navigator.clipboard.writeText(sub.code).then(
		() => {
			createToast({ status: "success", description: getText("copied") });
		},
		(err) => {
			createToast({ status: "error", description: getText("notCopied") });
			console.error(err);
		}
	);
}

function Summary({ sub, pasteAuthor }: { sub: FullSubmission; pasteAuthor?: UserBrief }) {
	return (
		<div class="segment-panel">
			<h2>{getText("info")}</h2>
			<table class="kn-table">
				<tbody>
					{pasteAuthor && (
						<>
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">{getText("shared_by")}</td>
								<td class="kn-table-cell">
									<a href={`/profile/${pasteAuthor.name}`}>{pasteAuthor.name}</a>
								</td>
							</tr>
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">{getText("sub_id")}</td>
								<td class="kn-table-cell">
									<a href={`/submissions/${sub.id}`}>#{sub.id}</a>
								</td>
							</tr>
						</>
					)}
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("sub_author")}</td>
						<td class="kn-table-cell">
							<a href={`/profile/${sub.author.name}`}>{sub.author.name}</a>
						</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("problemSingle")}</td>
						<td class="kn-table-cell">
							<a href={`${typeof sub.contest_id === "number" ? `/contests/${sub.contest_id}` : ""}/problems/${sub.problem.id}`}>
								{sub.problem.name}
							</a>
						</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("uploadDate")}</td>
						<td class="kn-table-cell">{parseTime(sub.created_at)}</td>
					</tr>
					{(sub.status === "finished" || sub.status == "reevaling") && (
						<>
							{sub.submission_type == "classic" ? (
								<tr class="kn-table-simple-border">
									<td class="kn-table-cell">{getText("score")}</td>
									<td class="kn-table-cell">
										<span class="badge-lite font-bold" style={{ backgroundColor: getGradient(sub.score, 100) }}>
											{sub.score.toFixed(sub.score_precision)}
										</span>
									</td>
								</tr>
							) : (
								<tr class="kn-table-simple-border">
									<td class="kn-table-cell">{getText("verdict")}</td>
									<td class="kn-table-cell">
										<span class="badge-lite font-bold">
											{sub.score == 100 ? (
												<>
													<i class="fas fa-fw fa-check"></i> {getText("accepted")}
												</>
											) : (
												<>{sub.icpc_verdict ? icpcVerdictString(sub.icpc_verdict) : getText("rejected")}</>
											)}
										</span>
									</td>
								</tr>
							)}
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">{getText("time")}</td>
								<td class="kn-table-cell">{sub.max_time == -1 ? "-" : `${Math.floor(sub.max_time * 1000)} ms`}</td>
							</tr>
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">{getText("memory")}</td>
								<td class="kn-table-cell">{sub.max_memory == -1 ? "-" : sizeFormatter(sub.max_memory * 1024)}</td>
							</tr>
						</>
					)}
					{sub.problem.default_points > 0 && (
						<tr class="kn-table-simple-border">
							<td class="kn-table-cell">{getText("defaultPoints")}</td>
							<td class="kn-table-cell">{sub.problem.default_points}</td>
						</tr>
					)}
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("language")}</td>
						<td class="kn-table-cell">{prettyLanguages[sub.language]}</td>
					</tr>
					{sub.code_size > 0 && (
						<tr class="kn-table-simple-border">
							<td class="kn-table-cell">{getText("codeSize")}</td>
							<td class="kn-table-cell">{sizeFormatter(sub.code_size)}</td>
						</tr>
					)}
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("status")}</td>
						<td class="kn-table-cell">{sub.status}</td>
					</tr>
					{sub.compile_time != null && (
						<tr class="kn-table-simple-border">
							<td class="kn-table-cell">{getText("compileTime")}</td>
							<td class="kn-table-cell">{Math.floor(sub.compile_time * 1000)} ms</td>
						</tr>
					)}
				</tbody>
			</table>
		</div>
	);
}

function CompileErrorInfo({ sub }: { sub: FullSubmission }) {
	if (sub.compile_error !== true) {
		if (typeof sub.compile_message !== "undefined" && sub.compile_message.length > 0) {
			return (
				<details>
					<summary>
						<h2 class="inline-block">{getText("compileMsg")}</h2>
					</summary>
					<pre class="mb-2" style={{ wordBreak: "break-all" }}>
						{sub.compile_message}
					</pre>
				</details>
			);
		}
		return <></>;
	}

	return (
		<>
			<h2>{getText("compileErr")}</h2>
			<details open={true}>
				<summary>
					<h2 class="inline-block">{getText("compileMsg")}</h2>
				</summary>
				<pre class="mb-2" style={{ wordBreak: "break-all" }}>
					{typeof sub.compile_message !== "undefined" && sub.compile_message.length > 0 ? sub.compile_message : "No compilation message available"}
				</pre>
			</details>
		</>
	);
}

function SubCode({ sub, codeHTML, isPaste }: { sub: FullSubmission; codeHTML: string; isPaste: boolean }) {
	let [warningDismissed, setWarningDismissed] = useState<boolean>(sub.truly_visible || isPaste);
	console.log(sub);
	if (!warningDismissed) {
		return (
			<div class="segment-panel">
				<h2>{getText("showSourceCodeQ")}</h2>
				<p class="mb-4">{getText("showSourceCodeExpl")}</p>
				<button class="inline-block btn btn-blue mx-auto" onClick={() => setWarningDismissed(true)}>
					{getText("showSourceCodeBtn")}
				</button>
			</div>
		);
	}

	return (
		<div class="segment-panel">
			<h2>{getText("sourceCode")}:</h2>
			<div dangerouslySetInnerHTML={{ __html: codeHTML }}></div>
			<div class="block my-2">
				{window.isSecureContext && (
					/* It only works with https OR localhost */
					<button class="btn btn-blue mr-2 text-semibold text-lg" onClick={() => copyCode(sub)}>
						{getText("copy")}
					</button>
				)}
				<button class="btn btn-blue text-semibold text-lg" onClick={() => downloadCode(sub)}>
					{getText("download")}
				</button>
			</div>
		</div>
	);
}

function testVerdictString(verdict: string): string {
	return verdict.replace(/translate:([a-z_]+)/g, (substr, p1) => {
		console.log(substr, p1);
		return maybeGetText("test_verdict." + p1);
	});
}

export function icpcVerdictString(verdict: string): string {
	return verdict.replace(/test_verdict.([a-z_]+)/g, (substr, p1) => {
		console.log(substr, p1);
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
														{(maxScore * (subtest.percentage / 100.0)).toFixed(precision)} / {maxScore.toFixed(precision)}
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
						{(subtask.score * (subtask.final_percentage / 100.0)).toFixed(precision)} / {subtask.score.toFixed(precision)}
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

function SubTasks({ sub, expandedTests }: { sub: FullSubmission; expandedTests: boolean }) {
	return (
		<>
			<details open={true}>
				<summary>
					<h2 class="inline-block">{getText("subTasks")}</h2>
				</summary>
				<div class="list-group mb-2">
					{sub.subtasks.map((subtask) => (
						<SubTask
							subtests={sub.subtests}
							problem_editor={sub.problem_editor}
							subtask={subtask}
							breakdown_mode={false}
							key={"stk_" + subtask.id}
							precision={sub.score_precision}
							defaultOpen={sub.subtasks.length === 1 ? true : undefined}
						/>
					))}
				</div>
			</details>
			<details open={expandedTests}>
				<summary>
					<h2 class="inline-block">{getText("individualTests")}</h2>
				</summary>
				<TestTable
					subtests={sub.subtests}
					subtasks={sub.subtasks}
					problem_editor={sub.problem_editor}
					precision={sub.score_precision}
					subType={sub.submission_type}
				/>
			</details>
		</>
	);
}

function SubmissionView({ sub, bigCode, codeHTML, pasteAuthor }: { sub: FullSubmission | null; bigCode: boolean; codeHTML: string; pasteAuthor?: any }) {
	if (sub === null) {
		return (
			<div class="page-holder grid-cols-1">
				<BigSpinner />
			</div>
		);
	}
	let content = (
		<div class="segment-panel">
			<CompileErrorInfo sub={sub} />
			{sub.compile_error !== true &&
				(sub.subtasks.length > 0 && sub.submission_type == "classic" ? (
					<SubTasks sub={sub} expandedTests={typeof pasteAuthor !== "undefined"} />
				) : (
					<>
						<h2 class="mb-2">{getText("tests")}</h2>
						<TestTable
							subtests={sub.subtests}
							subtasks={sub.subtasks}
							problem_editor={sub.problem_editor}
							precision={sub.score_precision}
							subType={sub.submission_type}
						/>
					</>
				))}
		</div>
	);

	let under = <></>;
	if (sub.code != null) {
		under = <SubCode sub={sub} codeHTML={codeHTML} isPaste={typeof pasteAuthor !== "undefined"} />;
	}

	if (bigCode) {
		[content, under] = [under, content];
	}

	return (
		<>
			<div class="page-holder">
				<div class="page-sidebar lg:order-last">
					<Summary sub={sub} pasteAuthor={pasteAuthor} />
					{window.platform_info.user_id !== undefined && window.platform_info.user_id > 0 && (
						<div class="segment-panel">
							<OlderSubmissions problemID={sub.problem.id} userID={window.platform_info.user_id} />
						</div>
					)}
				</div>
				<div class="page-content-wrapper">{content}</div>
			</div>
			{under}
		</>
	);
}

type SubMgrState = {
	sub: FullSubmission | null;
};

// TODO: Refactor into function
export class SubmissionManager extends Component<{ id: number; initialData: FullSubmission | null; codeHTML: string; bigCode?: boolean }, SubMgrState> {
	poll_mu: boolean;
	finished: boolean;
	poller: number | null;
	constructor(props) {
		super();
		this.poll_mu = false;
		this.finished = props.initialData?.status == "finished" ?? false;
		this.state = {
			sub: props.initialData,
		};

		this.poller = null;
	}

	async componentDidMount() {
		if (this.props.initialData == null) {
			await this.poll();
		}
		if (!this.finished) {
			console.info("Started poller");
			this.poller = setInterval(async () => {
				document.dispatchEvent(new CustomEvent("kn-poll"));
				await this.poll();
			}, 2000);
		}
	}

	stopPoller() {
		if (this.poller == null) {
			return;
		}
		console.info("Stopped poller");
		clearInterval(this.poller);
		this.poller = null;
	}

	async poll() {
		if (this.poll_mu === false) this.poll_mu = true;
		else return;
		console.info("Poll submission #", this.props.id);
		try {
			var resp = await defaultClient.getSubmission(this.props.id);
		} catch (e) {
			apiToast({ data: (e as Error).message, status: "error" });
			console.error(e);
			this.poll_mu = false;
			return;
		}

		this.setState(() => ({ sub: resp }));

		if (resp.status === "finished") {
			this.stopPoller();
			this.finished = true;
			this.forceUpdate();
		}

		this.poll_mu = false;
	}

	render() {
		return (
			<>
				<h1>
					{getText("sub")} {`#${this.props.id}`}
				</h1>
				<SubmissionView bigCode={false} sub={this.state.sub} codeHTML={this.props.codeHTML} />
			</>
		);
	}
}

export function PasteViewer({ paste_id, sub, author, codeHTML }: { paste_id: string; sub: FullSubmission; author: UserBrief; codeHTML: string }) {
	return (
		<>
			<h1>
				{getText("paste_title")} #{paste_id}
			</h1>
			<SubmissionView bigCode={true} sub={sub} pasteAuthor={author} codeHTML={codeHTML} />
		</>
	);
}

function PasteViewerDOM({ paste_id, authorenc, subenc, code }: { paste_id: string; authorenc: string; subenc: string; code: string }) {
	const author: UserBrief = JSON.parse(fromBase64(authorenc));
	const sub: FullSubmission = JSON.parse(fromBase64(subenc));
	return <PasteViewer paste_id={paste_id} sub={sub} author={author} codeHTML={code}></PasteViewer>;
}

function SubMgrDOM({ id, enc, code }: { id: string; enc: string; code: string }) {
	const subID = parseInt(id);
	if (isNaN(subID)) {
		throw new Error("Invalid submission ID");
	}
	if (fromBase64(enc) === "") {
		throw new Error("Invalid submission data");
	}
	let sub: FullSubmission = JSON.parse(fromBase64(enc));
	return <SubmissionManager id={subID} initialData={sub} codeHTML={code}></SubmissionManager>;
}

register(SubMgrDOM, "kn-sub-mgr", ["id", "enc", "code"]);
register(PasteViewerDOM, "kn-paste-viewer", ["paste_id", "authorenc", "subenc", "code"]);
