import { h, Fragment, render, Component, createRef } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";
import register from "preact-custom-element";
import { prettyLanguages } from "../langs";

import getText from "../translation";
import { createToast, apiToast } from "../toast";
const slugify = (str) =>
	str
		.toLowerCase()
		.trim()
		.replace(/[^\w\s-]/g, "")
		.replace(/[\s_-]+/g, "-")
		.replace(/^-+|-+$/g, "");

import { BigSpinner, Button, OlderSubmissions } from "./common";

import { downloadBlob, parseTime, sizeFormatter, getGradient } from "../util";

import { getCall, postCall } from "../net";

function downloadCode(sub) {
	var file = new Blob([sub.code], { type: "text/plain" });
	var filename = `${slugify(sub.problem.name)}-${sub.id}.${sub.language.replace(/[0-9]+$/g, "").replace("outputOnly", "txt")}`;
	downloadBlob(file, filename);
}

async function copyCode(sub) {
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

function Summary({ sub, pasteAuthor }) {
	return (
		<div class="page-sidebar-box">
			<h2>{getText("info")}</h2>
			<table class="kn-table mx-2">
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
							<a href={`/problems/${sub.problem.id}`}>{sub.problem.name}</a>
						</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("uploadDate")}</td>
						<td class="kn-table-cell">{parseTime(sub.created_at)}</td>
					</tr>
					{sub.status === "finished" && (
						<>
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">{getText("score")}</td>
								<td class="kn-table-cell">
									<span class="badge-lite font-bold" style={{ backgroundColor: getGradient(sub.score, 100) }}>
										{sub.score}
									</span>
								</td>
							</tr>
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
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("codeSize")}</td>
						<td class="kn-table-cell">{sizeFormatter(sub.code_size)}</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("status")}</td>
						<td class="kn-table-cell">{sub.status}</td>
					</tr>
				</tbody>
			</table>
		</div>
	);
}

function CompileErrorInfo({ sub }) {
	if (sub.compile_error !== true) {
		if (sub.compile_message?.length > 0) {
			return (
				<details>
					<summary>
						<h2 class="inline-block">{getText("compileMsg")}</h2>
					</summary>
					<pre class="mb-2">{sub.compile_message}</pre>
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
				<pre class="mb-2">{sub.compile_message?.length > 0 ? sub.compile_message : "No compilation message available"}</pre>
			</details>
		</>
	);
}

function SubCode({ sub }) {
	return (
		<>
			<h2>{getText("sourceCode")}</h2>
			<pre>
				<code
					class="hljs"
					dangerouslySetInnerHTML={{
						__html: window.hljs.highlight(sub.code, {
							language: sub.language.replace(/[0-9]+$/g, "").replace("outputOnly", "text"),
						}).value,
					}}
				>
					Rendering...
				</code>
			</pre>
			<div class="block my-2">
				<button class="btn btn-blue mr-2 text-semibold text-lg" onClick={() => copyCode(sub)}>
					{getText("copy")}
				</button>
				<button class="btn btn-blue text-semibold text-lg" onClick={() => downloadCode(sub)}>
					{getText("download")}
				</button>
			</div>
		</>
	);
}

function TestTable({ sub }) {
	function testSubTasks(test_id) {
		let stks: number[] = [];
		for (let st of sub.subTasks) {
			if (st.tests.includes(test_id)) {
				stks.push(st.visible_id as number);
			}
		}
		return stks;
	}

	return (
		<table class="kn-table mb-2">
			<thead>
				<tr>
					<th class="py-2" scope="col">
						{getText("id")}
					</th>
					<th scope="col">{getText("time")}</th>
					<th scope="col">{getText("memory")}</th>
					<th scope="col">{getText("verdict")}</th>
					<th scope="col">{getText("score")}</th>
					{sub.subTasks.length > 0 && <th scope="col">{getText("subTasks")}</th>}
					{sub.problemEditor && <th scope="col">{getText("output")}</th>}
				</tr>
			</thead>
			<tbody>
				{sub.subTests.map((subtest) => (
					<tr class="kn-table-row" key={"kn_test" + subtest.test.id}>
						<th class="py-1" scope="row" id={`test-${subtest.test.visible_id}`}>
							{subtest.test.visible_id}
						</th>
						{subtest.done ? (
							<>
								<td>{Math.floor(subtest.time * 1000)} ms</td>
								<td>{sizeFormatter(subtest.memory * 1024, 1, true)}</td>
								<td>{subtest.verdict}</td>
								<td class="text-black" style={{ backgroundColor: getGradient(subtest.score, 100) }}>
									{sub.subTasks.length > 0 ? (
										<>
											{subtest.score}% {getText("correct")}
										</>
									) : (
										<>
											{Math.round((subtest.test.score * subtest.score) / 100.0)} / {subtest.test.score}
										</>
									)}
								</td>
							</>
						) : (
							<>
								<td></td>
								<td></td>
								<td>
									<div class="fas fa-spinner animate-spin" role="status"></div> {getText("waiting")}
								</td>
								<td>-</td>
							</>
						)}
						{sub.subTasks.length > 0 && <td>{testSubTasks(subtest.test.id).join(", ")}</td>}
						{sub.problemEditor && (
							<td>
								<a href={"/proposer/get/subtest_output/" + subtest.id}>{getText("output")}</a>
							</td>
						)}
					</tr>
				))}
			</tbody>
		</table>
	);
}

function SubTask({ sub, subtask, detRef }) {
	var stkScore = useMemo(() => {
		let stk_score = 100;
		for (let testID of subtask.tests) {
			let actualSubtest = sub.subTestIDs[testID];
			if (actualSubtest !== undefined && actualSubtest.score < stk_score) {
				stk_score = actualSubtest.score;
			}
		}
		return stk_score;
	}, [sub, subtask]);

	let allSubtestsDone = useMemo(() => {
		let done = true;
		for (let testID of subtask.tests) {
			if (testID in sub.subTestIDs && !sub.subTestIDs[testID].done) {
				done = false;
			}
		}
		return done;
	}, [sub, subtask]);

	return (
		<details id={`stk-det-${subtask.visible_id}`} class="list-group-item">
			<summary class="pb-1 mt-1">
				{/* <span class="flex justify-between"> */}
				<span>{getText("nthSubTask", subtask.visible_id)}</span>
				{allSubtestsDone ? (
					<span class="float-right badge" style={{ backgroundColor: getGradient(stkScore, 100) }}>
						{Math.round((subtask.score * stkScore) / 100.0)} / {subtask.score}
					</span>
				) : (
					<span class="float-right badge">
						<i class="fas fa-cog animate-spin"></i>
					</span>
				)}
				{/* </span> */}
			</summary>
			<div class="list-group m-1">
				{subtask.tests.map((testID) => {
					if (!(testID in sub.subTestIDs)) {
						return (
							<div class="list-group-item flex justify-between">
								<span>This subtask's test didn't exist when this submission was created.</span>
							</div>
						);
					}
					let actualSubtest = sub.subTestIDs[testID];
					return (
						<a
							href={`#test-${actualSubtest.test.visible_id}`}
							class="list-group-item flex justify-between"
							onClick={() => (detRef.current.open = true)}
						>
							<span>{getText("nthTest", actualSubtest.test.visible_id)}</span>
							{actualSubtest.done ? (
								<span class="badge" style={{ backgroundColor: getGradient(actualSubtest.score, 100) }}>
									{Math.round((subtask.score * actualSubtest.score) / 100.0)} / {subtask.score}
								</span>
							) : (
								<span class="badge">
									<i class="fas fa-cog animate-spin"></i>
								</span>
							)}
						</a>
					);
				})}
			</div>
		</details>
	);
}

function SubTasks({ sub, expandedTests }) {
	let ref = createRef();

	return (
		<>
			<details open={true}>
				<summary>
					<h2 class="inline-block">{getText("subTasks")}</h2>
				</summary>
				<div class="list-group mb-2">
					{sub.subTasks.map((subtask) => (
						<SubTask sub={sub} subtask={subtask} detRef={ref} key={"stk_" + subtask.id} />
					))}
				</div>
			</details>
			<details ref={ref} open={expandedTests}>
				<summary>
					<h2 class="inline-block">{getText("individualTests")}</h2>
				</summary>
				<TestTable sub={sub} />
			</details>
		</>
	);
}

function SubmissionView({ sub, bigCode, pasteAuthor }: { sub: any; bigCode: boolean; pasteAuthor?: any }) {
	if (typeof sub === "undefined" || sub === null || (sub && Object.keys(sub).length == 0)) {
		return (
			<div class="page-holder grid-cols-1">
				<BigSpinner />
			</div>
		);
	}

	let content = (
		<>
			<CompileErrorInfo sub={sub} />
			{sub.subTests.length > 0 &&
				sub.compile_error !== true &&
				(sub.subTasks.length > 0 ? (
					<SubTasks sub={sub} expandedTests={typeof pasteAuthor !== "undefined"} />
				) : (
					<>
						<h2 class="mb-2">{getText("tests")}</h2>
						<TestTable sub={sub} />
					</>
				))}
		</>
	);

	let under = <></>;
	if (sub.code != null) {
		under = <SubCode sub={sub} />;
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
						<>
							<div class="page-sidebar-divider"></div>
							<div class="page-sidebar-box">
								<OlderSubmissions problemid={sub.problem.id} userid={window.platform_info.user_id} />
							</div>
						</>
					)}
				</div>
				<div class="page-content">{content}</div>
			</div>
			{under}
		</>
	);
}

type SubMgrState = {
	sub: any;
};

function transformSubmissionResponse(res: any): any {
	let sub: any = {};

	sub = res.sub;
	sub.problemEditor = res.problem_editor;
	sub.author = res.author;
	sub.problem = res.problem;
	if (res.subtasks) {
		sub.subTasks = res.subtasks;
	} else {
		sub.subTasks = [];
	}

	if (res.subtests) {
		sub.subTests = res.subtests;
		sub.subTestIDs = {};
		for (let subtest of res.subtests) {
			sub.subTestIDs[subtest.test.id] = subtest;
		}
	}
	return sub;
}

export class SubmissionManager extends Component<{ id: number; bigCode?: boolean }, SubMgrState> {
	poll_mu: boolean;
	finished: boolean;
	poller: number | null;
	constructor() {
		super();
		this.poll_mu = false;
		this.finished = false;
		this.setState(() => ({
			sub: {},
		}));

		this.poller = null;
	}

	async componentDidMount() {
		await this.poll();
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
		let resp = await getCall("/submissions/getByID", {
			id: this.props.id,
		});
		if (resp.status === "error") {
			apiToast(resp);
			console.error(resp);
			this.poll_mu = false;
			return;
		}

		this.setState(() => ({ sub: transformSubmissionResponse(resp.data) }));

		// const res = resp.data;
		// let newState: SubMgrState = { sub: {} };

		// newState.sub = res.sub;
		// newState.sub.problemEditor = res.problem_editor;
		// newState.sub.author = res.author;
		// newState.sub.problem = res.problem;
		// if (res.subtasks) {
		// 	newState.sub.subTasks = res.subtasks;
		// } else {
		// 	newState.sub.subTasks = [];
		// }

		// if (res.subtests) {
		// 	newState.sub.subTests = res.subtests;
		// 	newState.sub.subTestIDs = {};
		// 	for (let subtest of res.subtests) {
		// 		newState.sub.subTestIDs[subtest.test.id] = subtest;
		// 	}
		// }

		// this.setState(() => newState);

		if (resp.data.sub.status === "finished") {
			this.stopPoller();
			this.finished = true;
			this.forceUpdate();
		}

		this.poll_mu = false;
	}

	render() {
		return (
			<>
				<h1 class="mb-2">
					{getText("sub")} {`#${this.props.id}`}
				</h1>
				<SubmissionView bigCode={false} sub={this.state.sub} />
			</>
		);
	}
}

export function PasteViewer({ paste_id }: { paste_id: string }) {
	let [sub, setSub] = useState({});
	let [author, setAuthor] = useState({});

	async function load() {
		const res = await getCall(`/paste/${paste_id}`, {});
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		setSub(transformSubmissionResponse(res.data.sub));
		setAuthor(res.data.author);
	}
	console.log("here");

	useEffect(() => {
		load().catch(console.error);
	}, [paste_id]);

	return (
		<>
			<h1 class="mb-2">
				{getText("paste_title")} #{paste_id}
			</h1>
			<SubmissionView bigCode={true} sub={sub} pasteAuthor={author} />
		</>
	);
}

register(SubmissionManager, "kn-sub-mgr", ["id"]);
register(PasteViewer, "kn-paste-viewer", ["paste_id"]);
