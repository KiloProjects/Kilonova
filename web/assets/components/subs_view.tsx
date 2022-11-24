import { h, Fragment, Component } from "preact";
import getText from "../translation";
import register from "preact-custom-element";
import { useEffect, useMemo, useState } from "preact/hooks";
import { getCall } from "../net";
import { apiToast, createToast } from "../toast";
import { BigSpinner, Paginator } from "./common";
import { dayjs, getGradient, sizeFormatter } from "../util";
import _ from "underscore";

type Submission = {
	id: number;
	created_at: string;
	max_time: number;
	max_memory: number;
	score: number;
	status: string;
};

type UserBrief = {
	id: number;
	name: string;
	admin: boolean;
	proposer: boolean;
	bio: string;
};

type Problem = {
	id: number;
	name: string;
};

type ResultSubmission = {
	sub: Submission;
	author: UserBrief;
	problem: Problem;
	hidden: boolean;
};

type Query = {
	user_id: number | null;
	problem_id: number | null;
	score: number | null;
	status?: string;
	lang: string; // TODO: allow undefined?

	page: number;

	compile_error?: boolean;
	ordering: string;
	ascending: boolean;
};

const rezStr = (subCount: number): string => {
	if (subCount == 1) {
		return getText("oneResult");
	}
	if (subCount < 20) {
		return `${subCount} ${getText("u20Results")}`;
	}
	return `${subCount} ${getText("manyResults")}`;
};

const status = (sub: Submission): string => {
	if (sub.status === "finished") {
		return `${getText("evaluated")}: ${sub.score} ${getText("points")}`;
	} else if (sub.status === "working") {
		return getText("evaluating");
	}
	return getText("waiting");
};

function getInitialData(): Query {
	const params = new URLSearchParams(window.location.search);

	const user_id = parseInt(params.get("user_id") ?? "");
	const problem_id = parseInt(params.get("problem_id") ?? "");
	const score = parseInt(params.get("score") ?? "");

	let compile_error_str = params.get("compile_error");
	let compile_error: boolean | undefined;
	if (compile_error_str === "true" || compile_error_str === "false") {
		compile_error = compile_error_str === "true";
	}

	let status = params.get("status") ?? "";
	if (!["working", "waiting", "finished"].includes(status)) {
		status = "";
	}

	const ordering = params.get("ordering");
	const page = parseInt(params.get("page") ?? "");

	return {
		user_id: !isNaN(user_id) ? user_id : null,
		problem_id: !isNaN(problem_id) ? problem_id : null,
		score: !isNaN(score) ? score : null,
		status: status,
		lang: params.get("lang") ?? "",

		page: !isNaN(page) && page != 0 ? page : 1,

		compile_error: compile_error,
		ordering: ordering ? ordering : "id",
		ascending: params.get("ascending") === "true",
	};
}

function serializeQuery(q: Query): object {
	return {
		ordering: q.ordering,
		ascending: q.ascending,
		user_id: q.user_id !== null && q.user_id > 0 ? q.user_id : undefined,
		problem_id: q.problem_id !== null && q.problem_id > 0 ? q.problem_id : undefined,
		status: q.status !== "" ? q.status : undefined,
		score: q.score !== null && q.score >= 0 ? q.score : undefined,
		lang: q.lang !== "" ? q.lang : undefined,
		compile_error: q.compile_error,
		offset: (q.page - 1) * 50,
	};
}

function SubsView() {
	let [loading, setLoading] = useState(true);
	let [query, setQuery] = useState<Query>(getInitialData());
	let [subs, setSubs] = useState<ResultSubmission[]>([]);
	let [count, setCount] = useState<number>(-1);

	const numPages = useMemo(() => Math.floor(count / 50) + (count % 50 != 0 ? 1 : 0), [count]);

	const poll = _.throttle(async () => {
		setLoading(true);
		if (query.page === 1) {
			setCount(0);
		}

		let res = await getCall<{
			count: number;
			subs: ResultSubmission[];
		}>("/submissions/get", serializeQuery(query));

		if (res.status === "error") {
			apiToast(res);
			setLoading(false);
			return;
		}

		setSubs(res.data.subs);
		setCount(res.data.count);
		setLoading(false);
	}, 200);

	useEffect(() => {
		poll();
	}, [query]);

	async function copyQuery() {
		var p = new URLSearchParams();
		if (query.user_id !== null && query.user_id > 0) {
			p.append("user_id", query.user_id.toString());
		}
		if (query.problem_id !== null && query.problem_id > 0) {
			p.append("problem_id", query.problem_id.toString());
		}
		if (query.status !== undefined && query.status !== "") {
			p.append("status", query.status);
		}
		if (query.score !== null && query.score >= 0) {
			p.append("score", query.score?.toString());
		}
		if (query.lang !== "") {
			p.append("lang", query.lang);
		}
		if (query.compile_error !== undefined) {
			p.append("compile_error", String(query.compile_error));
		}
		if (query.ordering !== "id") {
			p.append("ordering", query.ordering);
		}
		if (query.ascending == true) {
			p.append("ascending", "true");
		}
		p.append("page", query.page.toString());
		let url = window.location.origin + window.location.pathname + "?" + p.toString();
		try {
			await navigator.clipboard.writeText(url);
			createToast({ status: "success", title: getText("copied") });
		} catch (e) {
			console.error(e);
			createToast({ status: "error", title: getText("notCopied") });
		}
	}

	return (
		<div class="page-holder">
			<div class="page-sidebar lg:order-last">
				<div class="page-sidebar-box">
					<h2>{getText("filters")}</h2>
					<label class="block mb-2">
						<span class="form-label">{getText("status")}:</span>
						<select
							class="form-select"
							value={query.status}
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
						</select>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("language")}:</span>
						<select
							class="form-select"
							value={query.lang}
							onChange={(e) => {
								setQuery({
									...query,
									page: 1,
									lang: e.currentTarget.value,
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
						<span class="form-label">{getText("sorting")}:</span>
						<select
							class="form-select"
							value={query.ordering}
							onChange={(e) => {
								setQuery({
									...query,
									page: 1,
									ordering: e.currentTarget.value,
								});
							}}
						>
							<option value="id" default>
								{getText("id")}
							</option>
							<option value="score">{getText("score")}</option>
							<option value="max_time">{getText("maxTime")}</option>
							<option value="max_mem">{getText("maxMemory")}</option>
						</select>
					</label>
					<label class="block mb-2">
						<input
							type="checkbox"
							class="form-checkbox mr-2"
							checked={query.ascending}
							onInput={(e) => setQuery({ ...query, page: 1, ascending: e.currentTarget.checked })}
						/>
						<span class="form-label">{getText("ascending")}</span>
					</label>
					<details class="block mb-2">
						<summary class="form-label">{getText("advancedOptions")}</summary>
						<label class="block mb-2">
							<span class="form-label">{getText("userID")}:</span>
							<input
								class="form-input"
								type="number"
								min="0"
								value={query.user_id ?? "a"}
								onInput={(e) => {
									let val: number | null = parseInt(e.currentTarget.value);
									if (isNaN(val) || val <= 0) {
										val = null;
									}
									setQuery({
										...query,
										page: 1,
										user_id: val,
									});
								}}
							/>
						</label>
						<label class="block mb-2">
							<span class="form-label">{getText("problemID")}:</span>
							<input
								class="form-input"
								type="number"
								min="0"
								value={query.problem_id ?? "a"}
								onInput={(e) => {
									let val: number | null = parseInt(e.currentTarget.value);
									if (isNaN(val) || val <= 0) {
										val = null;
									}
									setQuery({
										...query,
										page: 1,
										problem_id: val,
									});
								}}
							/>
						</label>
						<label class="block mb-2">
							<span class="form-label">{getText("score")}:</span>
							<input
								class="form-input"
								type="number"
								min="-1"
								max="100"
								value={query.score ?? "a"}
								onInput={(e) => {
									let val: number | null = parseInt(e.currentTarget.value);
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
										score: val,
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
				{count > 0 && (
					<>
						<h2 class="inline-block">{rezStr(count)}</h2>
						<div class="flex justify-center">
							<Paginator
								page={query.page}
								numpages={numPages}
								setPage={(num) => {
									setQuery({ ...query, page: num });
								}}
								ctxSize={2}
							/>
						</div>
					</>
				)}
				{loading ? (
					<BigSpinner />
				) : subs.length > 0 ? (
					<div>
						{query.problem_id != null && query.problem_id > 0 && (
							<p>
								{getText("problemSingle")} <a href={"/problems/" + subs[0].problem.id}>{subs[0].problem.name}</a>
							</p>
						)}
						<table class="kn-table">
							<thead>
								<tr>
									<th scope="col" class="w-12 text-center px-4 py-2">
										{getText("id")}
									</th>
									<th scope="col">{getText("author")}</th>
									<th scope="col">{getText("uploadDate")}</th>
									{(query.problem_id == 0 || query.problem_id == null) && <th scope="col">{getText("problemSingle")}</th>}
									<th scope="col">{getText("time")}</th>
									<th scope="col">{getText("memory")}</th>
									<th scope="col" class="w-1/6">
										{getText("status")}
									</th>
								</tr>
							</thead>
							<tbody>
								{subs.map((sub) => (
									<tr class="kn-table-row" key={sub.sub.id}>
										<th scope="row" class="text-center px-2 py-1">
											{sub.sub.id}
										</th>
										<td class="px-2 py-1">
											{/* <a class="inline-flex align-middle items-center" href={'/profile/'+sub.author.name}><img class="flex-none mr-2 rounded" src={'/api/user/getGravatar?s=32&name='+sub.author.name} width="32" height="32" alt="Avatar" /><span class="flex-1">{sub.author.name}</span></a>--> */}
											<a href={"/profile/" + sub.author.name}>{sub.author.name}</a>
										</td>
										<td class="text-center px-2 py-1">{dayjs(sub.sub.created_at).format("DD/MM/YYYY HH:mm")}</td>
										{(query.problem_id == 0 || query.problem_id == null) && (
											<td class="text-center px-2 py-1">
												{sub.hidden ? <span>---</span> : <a href={"/problems/" + sub.problem.id}>{sub.problem.name}</a>}
											</td>
										)}
										<td class="text-center px-2 py-1">{sub.sub.max_time == -1 ? "-" : Math.floor(sub.sub.max_time * 1000) + "ms"}</td>
										<td class="text-center px-2 py-1">{sub.sub.max_memory == -1 ? "-" : sizeFormatter(sub.sub.max_memory * 1024)}</td>
										<td
											class={(sub.sub.status === "finished" && !sub.hidden ? "text-black" : "") + " text-center"}
											style={sub.sub.status == "finished" && !sub.hidden ? "background-color: " + getGradient(sub.sub.score, 100) : ""}
										>
											{sub.hidden ? <span>---</span> : <a href={"/submissions/" + sub.sub.id}>{status(sub.sub)}</a>}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				) : (
					<div class="text-4xl mx-auto my-auto w-full mt-10 mb-10 text-center">{getText("noSubFound")}</div>
				)}
			</div>
		</div>
	);
}

register(SubsView, "kn-sub-viewer", []);

export { SubsView };
