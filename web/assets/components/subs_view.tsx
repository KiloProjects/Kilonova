import { h, Fragment } from "preact";
import getText from "../translation";
import register from "preact-custom-element";
import { useEffect, useMemo, useState } from "preact/hooks";
import { apiToast, createToast } from "../toast";
import { BigSpinner, Paginator } from "./common";
import { dayjs, getGradient, sizeFormatter } from "../util";
import { SubmissionQuery, Submissions, defaultClient } from "../api/client";
import { icpcVerdictString } from "./sub_mgr";

export function rezStr(count: number, truncatedCount: boolean = false): string {
	let truncCnt = truncatedCount ? `>${count}` : count.toString();
	if (count < 0) {
		return `- ${getText("u20Results")}`;
	}
	if (count == 0) {
		return `0 ${getText("u20Results")}`;
	}
	if (count == 1) {
		return getText("oneResult");
	}
	if (count < 20) {
		return `${truncCnt} ${getText("u20Results")}`;
	}
	return `${truncCnt} ${getText("manyResults")}`;
}

const status = (sub: Submission): string | h.JSX.Element => {
	if (sub.status === "finished") {
		if (sub.submission_type == "classic") {
			return `${getText("evaluated")}: ${sub.score.toFixed(sub.score_precision)} ${getText("points")}`;
		}
		// else, icpc
		if (sub.score == 100) {
			return (
				<>
					<i class="fas fa-fw fa-check"></i> {getText("accepted")}
				</>
			);
		}
		return <>{sub.icpc_verdict ? icpcVerdictString(sub.icpc_verdict) : getText("rejected")}</>;
	} else if (sub.status === "working") {
		return getText("evaluating");
	} else if ((sub.status = "reevaling")) {
		return getText("reevaling");
	}
	return getText("waiting");
};

type Overwrites = {
	contestID?: number;
	problemID?: number;
	userID?: number;
};

function getInitialData(overwrites: Overwrites): SubmissionQuery {
	const params = new URLSearchParams(window.location.search);

	const userIDParam = parseInt(params.get("user_id") ?? "");
	const problemIDParam = parseInt(params.get("problem_id") ?? "");
	const contestIDParam = parseInt(params.get("contest_id") ?? "");
	const score = parseFloat(params.get("score") ?? "");

	let compile_error_str = params.get("compile_error");
	let compile_error: boolean | undefined;
	if (compile_error_str === "true" || compile_error_str === "false") {
		compile_error = compile_error_str === "true";
	}

	let status = params.get("status") ?? "";
	if (!["working", "waiting", "finished", "reevaling"].includes(status)) {
		status = "";
	}

	const ordering = params.get("ordering");
	const page = parseInt(params.get("page") ?? "");

	let problemID = !isNaN(problemIDParam) ? problemIDParam : undefined;
	if (typeof overwrites.problemID !== "undefined") {
		problemID = overwrites.problemID;
	}

	let userID = !isNaN(userIDParam) ? userIDParam : undefined;
	if (typeof overwrites.userID !== "undefined") {
		userID = overwrites.userID;
	}

	let contestID = !isNaN(contestIDParam) ? contestIDParam : undefined;
	if (typeof overwrites.contestID !== "undefined") {
		contestID = overwrites.contestID;
	}

	return {
		user_id: userID,
		problem_id: problemID,
		contest_id: contestID,
		score: !isNaN(score) ? score : undefined,
		status: status,
		lang: params.get("lang") ?? "",

		page: !isNaN(page) && page != 0 ? page : 1,

		compile_error: compile_error,
		ordering: ordering ? ordering : "id",
		ascending: params.get("ascending") === "true",
	};
}

export type SubsViewProps = {
	problemid?: number;
	userid?: number;
	contestid?: number;
	title?: string;
};

function SubsView(props: SubsViewProps) {
	let overwrites: Overwrites = { problemID: props.problemid, userID: props.userid, contestID: props.contestid };
	let [loading, setLoading] = useState(true);
	let [query, updQuery] = useState<SubmissionQuery>(getInitialData(overwrites));
	let [subs, setSubs] = useState<Submissions | undefined>(undefined);
	let [count, setCount] = useState<number>(-1);
	let [truncatedCount, setTruncatedCount] = useState<boolean>(false);
	let [initialLoad, setInitialLoad] = useState(true);

	const setQuery = (q: SubmissionQuery, ignoreReset?: boolean) => {
		updQuery(q);
		if (!ignoreReset) {
			setCount(-1);
			setTruncatedCount(false);
		}
	};

	const numPages = useMemo(() => Math.floor(count / 50) + (count % 50 != 0 ? 1 : 0), [count]);

	async function poll(noLoad?: boolean) {
		if (typeof noLoad === "undefined" || !noLoad) {
			setLoading(true);
		}

		try {
			var data = await defaultClient.getSubmissions(query);
		} catch (e) {
			apiToast({ data: (e as Error).message, status: "error" });
			setLoading(false);
			return;
		}

		setSubs(data);
		setCount(data.count);
		setTruncatedCount(data.truncated_count);
		setLoading(false);
		setInitialLoad(false);
		let str = getQuery().toString();
		if (str.length > 0) str = "?" + str;
		const newName = window.location.pathname + str;
		if (window.location.pathname + window.location.search != newName) {
			history.pushState({}, "", newName);
		}
	}

	useEffect(() => {
		poll()?.catch(console.error);
	}, [query]);

	useEffect(() => {
		const eventPoll = async (e) => poll(true);
		document.addEventListener("kn-poll", eventPoll);
		return () => document.removeEventListener("kn-poll", eventPoll);
	}, []);

	useEffect(() => {
		const historyPopEvent = async (e) => {
			setSubs(undefined);
			setCount(-1);
			setTruncatedCount(false);
			setLoading(true);
			setInitialLoad(true);
			updQuery(getInitialData(overwrites));
		};
		window.addEventListener("popstate", historyPopEvent);
		return () => window.removeEventListener("popstate", historyPopEvent);
	});

	function getQuery() {
		var p = new URLSearchParams();
		// add to query only if they were not supplied by default
		if (typeof overwrites.userID === "undefined" && typeof query.user_id !== "undefined" && query.user_id > 0) {
			p.append("user_id", query.user_id.toString());
		}
		if (typeof overwrites.problemID === "undefined" && typeof query.problem_id !== "undefined" && query.problem_id > 0) {
			p.append("problem_id", query.problem_id.toString());
		}
		if (typeof overwrites.contestID === "undefined" && typeof query.contest_id !== "undefined" && query.contest_id >= 0) {
			p.append("contest_id", query.contest_id.toString());
		}

		if (query.status !== undefined && query.status !== "") {
			p.append("status", query.status);
		}
		if (typeof query.score !== "undefined" && query.score >= 0) {
			p.append("score", query.score?.toString());
		}
		if (typeof query.lang !== "undefined" && query.lang !== "") {
			p.append("lang", query.lang);
		}
		if (typeof query.compile_error !== "undefined") {
			p.append("compile_error", String(query.compile_error));
		}
		if (typeof query.ordering !== "undefined" && query.ordering !== "id") {
			p.append("ordering", query.ordering);
		}
		if (query.ascending == true) {
			p.append("ascending", "true");
		}
		if (query.page != 1) {
			p.append("page", query.page.toString());
		}
		return p;
	}

	async function copyQuery() {
		let url = window.location.origin + window.location.pathname + "?" + getQuery().toString();
		try {
			await navigator.clipboard.writeText(url);
			createToast({ status: "success", title: getText("copied") });
		} catch (e) {
			console.error(e);
			createToast({ status: "error", title: getText("notCopied") });
		}
	}

	function doSort(type: string) {
		let ordering = typeof query.ordering === "undefined" ? "id" : query.ordering;
		let ascending = typeof query.ascending === "undefined" ? false : query.ascending;

		if (type === ordering) {
			ascending = !ascending;
		} else {
			ordering = type;
			ascending = false;
		}

		setQuery({ ...query, ascending, ordering });
	}

	function sortIndicator(type: string) {
		let ordering = typeof query.ordering === "undefined" ? "id" : query.ordering;
		let ascending = typeof query.ascending === "undefined" ? false : query.ascending;
		if (ordering !== type) {
			return <i class="fas fa-sort"></i>;
		}
		if (ascending) {
			return <i class="fas fa-sort-up"></i>;
		}
		return <i class="fas fa-sort-down"></i>;
	}

	// Page-holder has mt-0 so it looks neat on the problem submit page
	return (
		<div class="page-holder mt-0">
			<div class="page-sidebar lg:order-last">
				<div class="segment-panel">
					<h2>{getText("filters")}</h2>
					<label class="block mb-2">
						<span class="form-label">{getText("language")}:</span>
						<select
							class="form-select"
							value={query.lang === undefined ? "" : query.lang}
							onChange={(e) => {
								setQuery({
									...query,
									page: 1,
									lang: e.currentTarget.value == "" ? undefined : e.currentTarget.value,
								});
							}}
						>
							<option value="">-</option>
							{Object.entries(window.platform_info.langs).map(
								([name, lang]) =>
									!lang.disabled && (
										<option value={name} key={name}>
											{lang.name}
										</option>
									)
							)}
						</select>
					</label>
					<label class="block mb-2">
						<input
							type="checkbox"
							class="form-checkbox mr-2"
							checked={query.status === "finished" && query.score === 100}
							onInput={(e) => {
								if (e.currentTarget.checked) {
									setQuery({ ...query, page: 1, status: "finished", score: 100 });
								} else {
									setQuery({ ...query, page: 1, status: undefined, score: undefined });
								}
							}}
						/>
						<span class="form-label">{getText("acceptedSubs")}</span>
					</label>
					<details class="block mb-2">
						<summary class="form-label">{getText("advancedOptions")}</summary>
						<label class="block mb-2">
							<span class="form-label">{getText("status")}:</span>
							<select
								class="form-select"
								value={typeof query.status === "undefined" ? "" : query.status}
								onChange={(e) => {
									setQuery({
										...query,
										page: 1,
										status: e.currentTarget.value,
									});
								}}
							>
								<option value="">-</option>
								<option value="finished">{getText("finished")}</option>
								<option value="working">{getText("working")}</option>
								<option value="waiting">{getText("waiting")}</option>
								<option value="reevaling">{getText("reevaling")}</option>
							</select>
						</label>
						{typeof overwrites.userID === "undefined" && (
							<label class="block mb-2">
								<span class="form-label">{getText("userID")}:</span>
								<input
									class="form-input"
									type="number"
									min="0"
									value={typeof query.user_id == "undefined" ? "a" : query.user_id}
									onInput={(e) => {
										let val: number | null = parseInt(e.currentTarget.value);
										if (isNaN(val) || val <= 0) {
											val = null;
										}
										setQuery({
											...query,
											page: 1,
											user_id: val == null ? undefined : val,
										});
									}}
								/>
							</label>
						)}
						{typeof overwrites.problemID === "undefined" && (
							<label class="block mb-2">
								<span class="form-label">{getText("problemID")}:</span>
								<input
									class="form-input"
									type="number"
									min="0"
									value={typeof query.problem_id == "undefined" ? "a" : query.problem_id}
									onInput={(e) => {
										let val: number | null = parseInt(e.currentTarget.value);
										if (isNaN(val) || val <= 0) {
											val = null;
										}
										setQuery({
											...query,
											page: 1,
											problem_id: val == null ? undefined : val,
										});
									}}
								/>
							</label>
						)}
						{typeof overwrites.contestID === "undefined" && (
							<label class="block mb-2">
								<span class="form-label">{getText("contestID")}:</span>
								<input
									class="form-input"
									type="number"
									min="0"
									value={typeof query.contest_id == "undefined" ? "a" : query.contest_id}
									onInput={(e) => {
										let val: number | null = parseInt(e.currentTarget.value);
										if (isNaN(val) || val <= 0) {
											val = null;
										}
										setQuery({
											...query,
											page: 1,
											contest_id: val == null ? undefined : val,
										});
									}}
								/>
							</label>
						)}
						<label class="block mb-2">
							<span class="form-label">{getText("score")}:</span>
							<input
								class="form-input"
								type="number"
								min={-1}
								max={100}
								step={0.0001}
								value={typeof query.score == "undefined" ? "a" : query.score}
								onInput={(e) => {
									let val: number | null = parseFloat(e.currentTarget.value);
									if (isNaN(val)) {
										val = null;
									} else {
										if (val > 100) {
											val = 100;
										}
										if (val < 0) {
											val = null;
										}
									}
									setQuery({
										...query,
										page: 1,
										score: val == null ? undefined : val,
									});
								}}
							/>
						</label>
						<label class="block mb-2">
							<span class="form-label">{getText("compileErr")}:</span>
							<select
								class="form-select"
								value={query.compile_error === undefined ? "" : String(query.compile_error)}
								onChange={(e) => {
									let val = e.currentTarget.value;
									let cerr: boolean | undefined;
									if (val == "true" || val == "false") {
										cerr = val === "true";
									}
									setQuery({
										...query,
										page: 1,
										compile_error: cerr,
									});
								}}
							>
								<option value="">-</option>
								<option value="true">{getText("yes")}</option>
								<option value="false">{getText("no")}</option>
							</select>
						</label>
					</details>
					<button class="btn btn-blue mb-4 mr-2" onClick={() => poll()}>
						{getText("fetch")}
					</button>
					<button class="btn mb-4" onClick={async () => await copyQuery()}>
						{getText("filterLink")}
					</button>
				</div>
			</div>
			<div class="page-content">
				{typeof props.title !== "undefined" && <h1>{props.title}</h1>}
				{!initialLoad && (
					<>
						<h2 class="inline-block">{rezStr(count, truncatedCount)}</h2>
						<div class="flex justify-center">
							{count > 0 ? (
								<Paginator
									page={query.page}
									numpages={numPages}
									setPage={(num) => {
										setQuery({ ...query, page: num }, true);
									}}
									ctxSize={2}
									showArrows={true}
								/>
							) : typeof subs !== "undefined" && subs.submissions.length > 0 ? (
								<Paginator page={1} numpages={1} setPage={() => {}} ctxSize={2} showArrows={true} />
							) : (
								<></>
							)}
						</div>
					</>
				)}
				{!loading &&
					typeof overwrites.problemID === "undefined" &&
					query.problem_id != null &&
					query.problem_id > 0 &&
					typeof subs !== "undefined" &&
					Object.keys(subs.problems).length > 0 && (
						<p>
							{getText("problemSingle")}{" "}
							<a href={"/problems/" + subs.problems[Object.keys(subs.problems)[0]].id}>{subs.problems[Object.keys(subs.problems)[0]].name}</a>
						</p>
					)}
				{initialLoad ? (
					<>
						<div class="lg:mt-6" />
						<BigSpinner />
					</>
				) : typeof subs !== "undefined" && subs.submissions.length > 0 ? (
					<div>
						<table class="kn-table">
							<thead>
								<tr>
									<th scope="col" class="w-20 text-center px-4 py-2">
										{getText("id")}
									</th>
									<th scope="col">{getText("author")}</th>
									<th scope="col" class="cursor-pointer" onClick={() => doSort("id")}>
										{getText("uploadDate")} {sortIndicator("id")}
									</th>
									{((query.problem_id == 0 || query.problem_id == null) && <th scope="col">{getText("problemSingle")}</th>) || (
										<th scope="col" class="cursor-pointer" onClick={() => doSort("code_size")}>
											{getText("codeSize")} {sortIndicator("code_size")}
										</th>
									)}
									<th scope="col" class="cursor-pointer" onClick={() => doSort("max_time")}>
										{getText("time")} {sortIndicator("max_time")}
									</th>
									<th scope="col" class="cursor-pointer" onClick={() => doSort("max_mem")}>
										{getText("memory")} {sortIndicator("max_mem")}
									</th>
									<th scope="col" class="w-1/6 cursor-pointer" onClick={() => doSort("score")}>
										{getText("status")} {sortIndicator("score")}
									</th>
								</tr>
							</thead>
							{loading ? (
								<tbody>
									<tr class="my-6">
										<td colSpan={20}>
											<BigSpinner />
										</td>
									</tr>
								</tbody>
							) : (
								<tbody>
									{subs.submissions.map((sub) => (
										<tr class="kn-table-row" key={sub.id}>
											<th scope="row" class="text-center px-2 py-1">
												{sub.id}
											</th>
											<td class="px-2 py-1">
												<a href={"/profile/" + subs!.users[sub.user_id].name}>{subs!.users[sub.user_id].name}</a>
											</td>
											<td class="text-center px-2 py-1">{dayjs(sub.created_at).format("DD/MM/YYYY HH:mm")}</td>
											{((query.problem_id == 0 || query.problem_id == null) && (
												<td class="text-center px-2 py-1">
													{subs!.problems[sub.problem_id] ? (
														<>
															<a
																href={
																	(typeof sub.contest_id === "number" ? `/contests/${sub.contest_id}` : "") +
																	`/problems/${subs!.problems[sub.problem_id].id}`
																}
															>
																{subs!.problems[sub.problem_id].name}
															</a>
														</>
													) : (
														<>-</>
													)}
												</td>
											)) || (
												<td class="text-center px-2 py-1">
													<span>{sub.code_size > 0 ? sizeFormatter(sub.code_size) : "-"}</span>
												</td>
											)}
											<td class="text-center px-2 py-1">{sub.max_time == -1 ? "-" : Math.floor(sub.max_time * 1000) + "ms"}</td>
											<td class="text-center px-2 py-1">{sub.max_memory == -1 ? "-" : sizeFormatter(sub.max_memory * 1024)}</td>
											<td
												class={(sub.status === "finished" ? "text-black" : "") + " text-center"}
												style={
													sub.status == "finished"
														? "background-color: " +
														  (sub.submission_type == "classic"
																? getGradient(sub.score, 100)
																: getGradient(sub.score == 100 ? 1 : 0, 1))
														: ""
												}
											>
												<a href={"/submissions/" + sub.id}>{status(sub)}</a>
											</td>
										</tr>
									))}
								</tbody>
							)}
						</table>
					</div>
				) : (
					<div class="text-4xl mx-auto my-auto w-full mt-10 mb-10 text-center">{getText("noSubFound")}</div>
				)}
			</div>
		</div>
	);
}

register(SubsView, "kn-sub-viewer", ["problemid", "userid", "contestid", "title"]);

export { SubsView };
