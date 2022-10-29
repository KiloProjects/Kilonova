import {
	h,
	Fragment,
	render,
	Component,
	AnyComponent,
	createRef,
} from "preact";
import { useMemo } from "preact/hooks";
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

import { BigSpinner, OlderSubmissions } from "./common";

import { downloadBlob, parseTime, sizeFormatter, getGradient } from "../util";

import { getCall, postCall } from "../net";

// TODO: Test all buttons

function downloadCode(sub) {
	var file = new Blob([sub.code], { type: "text/plain" });
	var filename = `${slugify(sub.problem.name)}-${sub.id}.${sub.language
		.replace(/[0-9]+$/g, "")
		.replace("outputOnly", "txt")}`;
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

function Summary({ sub }) {
	return (
		<div class="lg:pl-4 lg:pb-2 lg:pr-2">
			<h2>{getText("info")}</h2>
			<table class="kn-table">
				<tbody>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("author")}</td>
						<td class="kn-table-cell">
							<a href={`/profile/${sub.author.name}`}>
								{sub.author.name}
							</a>
						</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">
							{getText("problemSingle")}
						</td>
						<td class="kn-table-cell">
							<a href={`/problems/${sub.problem.id}`}>
								{sub.problem.name}
							</a>
						</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("uploadDate")}</td>
						<td class="kn-table-cell">
							{parseTime(sub.created_at)}
						</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("status")}</td>
						<td class="kn-table-cell">{sub.status}</td>
					</tr>
					<tr class="kn-table-simple-border">
						<td class="kn-table-cell">{getText("language")}</td>
						<td class="kn-table-cell">
							{prettyLanguages[sub.language]}
						</td>
					</tr>
					{sub.code && (
						<tr class="kn-table-simple-border">
							<td class="kn-table-cell">{getText("size")}</td>
							<td class="kn-table-cell">
								{sizeFormatter(sub.code.length)}
							</td>
						</tr>
					)}
					{sub.problem.default_points > 0 && (
						<tr class="kn-table-simple-border">
							<td class="kn-table-cell">
								{getText("defaultPoints")}
							</td>
							<td class="kn-table-cell">
								{sub.problem.default_points}
							</td>
						</tr>
					)}
					{sub.status === "finished" && (
						<>
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">
									{getText("score")}
								</td>
								<td class="kn-table-cell">{sub.score}</td>
							</tr>
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">
									{getText("maxTime")}
								</td>
								<td class="kn-table-cell">
									{sub.max_time == -1
										? "-"
										: `${Math.floor(
												sub.max_time * 1000
										  )} ms`}
								</td>
							</tr>
							<tr class="kn-table-simple-border">
								<td class="kn-table-cell">
									{getText("maxMemory")}
								</td>
								<td class="kn-table-cell">
									{sub.max_memory == -1
										? "-"
										: sizeFormatter(
												sub.max_memory * 1024,
												1,
												true
										  )}
								</td>
							</tr>
						</>
					)}
				</tbody>
			</table>
		</div>
	);
}

function CompileErrorInfo({ sub }) {
	if (!sub.compile_error.Bool) {
		if (sub.compile_message.String.length > 0) {
			return (
				<details>
					<summary>
						<h2 class="inline-block">{getText("compileMsg")}</h2>
					</summary>
					<pre class="mb-2">{sub.compile_message.String}</pre>
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
				<pre class="mb-2">
					{sub.compile_message.String.length > 0
						? sub.compile_message.String
						: "No compilation message provided"}
				</pre>
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
							language: sub.language
								.replace(/[0-9]+$/g, "")
								.replace("outputOnly", "text"),
						}).value,
					}}
				>
					Rendering...
				</code>
			</pre>
			<div class="block my-2">
				<button
					class="btn btn-blue mr-2 text-semibold text-lg"
					onClick={() => copyCode(sub)}
				>
					{getText("copy")}
				</button>
				<button
					class="btn btn-blue text-semibold text-lg"
					onClick={() => downloadCode(sub)}
				>
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
					{sub.subTasks.length > 0 && (
						<th scope="col">{getText("subTasks")}</th>
					)}
					{sub.problemEditor && (
						<th scope="col">{getText("output")}</th>
					)}
				</tr>
			</thead>
			<tbody>
				{sub.subTests.map((subtest) => (
					<tr class="kn-table-row" key={"kn_test" + subtest.test.id}>
						<th
							class="py-1"
							scope="row"
							id={`test-${subtest.test.visible_id}`}
						>
							{subtest.test.visible_id}
						</th>
						{subtest.done ? (
							<>
								<td>{Math.floor(subtest.time * 1000)} ms</td>
								<td>
									{sizeFormatter(
										subtest.memory * 1024,
										1,
										true
									)}
								</td>
								<td>{subtest.verdict}</td>
								<td
									class="text-black"
									style={
										"background-color: " +
										getGradient(subtest.score, 100)
									}
								>
									{sub.subTasks.length > 0 ? (
										<>
											{subtest.score}%{" "}
											{getText("correct")}
										</>
									) : (
										<>
											{Math.round(
												(subtest.test.score *
													subtest.score) /
													100.0
											)}{" "}
											/ {subtest.test.score}
										</>
									)}
								</td>
							</>
						) : (
							<>
								<td></td>
								<td></td>
								<td>
									<div
										class="fas fa-spinner animate-spin"
										role="status"
									></div>{" "}
									{getText("waiting")}
								</td>
								<td>-</td>
							</>
						)}
						{sub.subTasks.length > 0 && (
							<td>{testSubTasks(subtest.test.id).join(", ")}</td>
						)}
						{sub.problemEditor && (
							<td>
								<a
									href={
										"/proposer/get/subtest_output/" +
										subtest.id
									}
								>
									{getText("output")}
								</a>
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
			if (
				actualSubtest !== undefined &&
				actualSubtest.score < stk_score
			) {
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
				<span class="float-left">
					{getText("nthSubTask", subtask.visible_id)}
				</span>
				{allSubtestsDone ? (
					<span
						class="float-right rounded-full py-1 px-2 text-base text-white font-semibold"
						style={`background-color: ${getGradient(
							stkScore,
							100
						)}`}
					>
						{Math.round((subtask.score * stkScore) / 100.0)} /{" "}
						{subtask.score}
					</span>
				) : (
					<span class="float-right rounded-full py-1 px-2 text-base text-white font-semibold bg-teal-700">
						<i class="fas fa-cog animate-spin"></i>
					</span>
				)}
				{/* </span> */}
			</summary>
			<div class="list-group m-1 list-group-mini">
				{subtask.tests.map((testID) => {
					if (!(testID in sub.subTestIDs)) {
						return (
							<div class="list-group-item flex justify-between">
								<span>
									This subtask's test didn't exist when this
									submission was created.
								</span>
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
							<span>
								{getText(
									"nthTest",
									actualSubtest.test.visible_id
								)}
							</span>
							{actualSubtest.done ? (
								<span
									class="rounded-full py-1 px-2 text-base text-white font-semibold"
									style={`background-color: ${getGradient(
										actualSubtest.score,
										100
									)}`}
								>
									{Math.round(
										(subtask.score * actualSubtest.score) /
											100.0
									)}{" "}
									/ {subtask.score}
								</span>
							) : (
								<span class="rounded-full py-1 px-2 text-base text-white font-semibold bg-teal-700">
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

function SubTasks({ sub }) {
	let ref = createRef();

	return (
		<>
			<details open={true}>
				<summary>
					<h2 class="inline-block">{getText("subTasks")}</h2>
				</summary>
				<div class="list-group mb-2 list-group-mini">
					{sub.subTasks.map((subtask) => (
						<SubTask
							sub={sub}
							subtask={subtask}
							detRef={ref}
							key={"stk_" + subtask.id}
						/>
					))}
				</div>
			</details>
			<details ref={ref}>
				<summary>
					<h2 class="inline-block">{getText("individualTests")}</h2>
				</summary>
				<TestTable sub={sub} />
			</details>
		</>
	);
}

type SubMgrState = {
	sub: any;
};

export class SubmissionManager extends Component<{ id: number }, SubMgrState> {
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
				document.dispatchEvent(new Event("kn-poll"));
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
		console.log("Poll submission #", this.props.id);
		let res = await getCall("/submissions/getByID", {
			id: this.props.id,
		});
		if (res.status !== "success") {
			apiToast(res);
			console.error(res);
			this.poll_mu = false;
			return;
		}

		res = res.data;
		let newState: SubMgrState = { sub: {} };

		newState.sub = res.sub;
		newState.sub.problemEditor = res.problem_editor;
		newState.sub.author = res.author;
		newState.sub.problem = res.problem;
		if (res.subtasks) {
			newState.sub.subTasks = res.subtasks;
		} else {
			newState.sub.subTasks = [];
		}

		if (res.subtests) {
			newState.sub.subTests = res.subtests;
			newState.sub.subTestIDs = {};
			for (let subtest of res.subtests) {
				newState.sub.subTestIDs[subtest.test.id] = subtest;
			}
		}

		this.setState(() => newState);

		if (res.sub.status === "finished") {
			this.stopPoller();
			this.finished = true;
			this.forceUpdate();
		}

		this.poll_mu = false;
	}

	render() {
		let { sub } = this.state;
		if (sub && Object.keys(sub).length == 0) {
			return (
				<>
					<h1 class="mb-2">
						{getText("sub")} {`#${this.props.id}`}
					</h1>
					<div class="border-t-2 min-h-screen">
						<BigSpinner />
					</div>
				</>
			);
		}
		return (
			<>
				<h1 class="mb-2">
					{getText("sub")} {`#${sub.id}`}
				</h1>
				<div class="border-t-2 lg:border-b-2 grid grid-cols-1 lg:grid-cols-4">
					<div class="col-span-1 lg:pt-2 lg:order-last lg:border-l lg:pb-4">
						<Summary sub={sub} />
						{window.platform_info?.user_id !== undefined &&
							window.platform_info?.user_id > 0 && (
								<>
									<div class="h-0 w-full border-t border-gray-200"></div>
									<div class="my-2 lg:pl-2 lg:pb-4">
										<OlderSubmissions
											problemid={sub.problem.id}
											userid={
												window.platform_info?.user_id
											}
										/>
									</div>
									{/* <div class="h-0 w-full border-t border-gray-200 lg:pb-4"></div> */}
								</>
							)}
					</div>
					<div class="col-span-1 lg:pt-2 lg:col-span-3 lg:border-r lg:pr-4 lg:pb-4">
						<CompileErrorInfo sub={sub} />
						{sub.subTests.length > 0 &&
							!sub.compile_error.Bool &&
							(sub.subTasks.length > 0 ? (
								<>
									<SubTasks sub={sub} />
								</>
							) : (
								<>
									<h2 class="mb-2">{getText("tests")}</h2>
									<TestTable sub={sub} />
								</>
							))}
					</div>
				</div>
				{sub.code != null && <SubCode sub={sub} />}
			</>
		);
	}
}

register(SubmissionManager, "kn-sub-mgr", ["id"]);
