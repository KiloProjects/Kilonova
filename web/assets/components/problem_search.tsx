import { h, Fragment, render } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";
import { apiToast, createToast } from "../toast";
import { bodyCall, getCall } from "../api/client";
import { fromBase64 } from "js-base64";
import { Tag, TagView, selectTags } from "./tags";
import { Paginator } from "./common";
import { rezStr } from "./subs_view";
import { parseTime } from "../util";

type FullProblem = Problem & {
	tags: Tag[];
	max_score: number;
	is_editor: boolean;

	solved_by: number;
	attempted_by: number;
};

function numPagesF(count: number, max: number): number {
	return Math.floor(count / max) + (count % max != 0 ? 1 : 0);
}

export function ProblemView({
	problems,
	showTags,
	scoreView,
	latestView,
}: {
	problems: FullProblem[];
	showTags: boolean;
	scoreView: boolean;
	latestView: boolean;
}) {
	let authed = window.platform_info.user_id >= 1;
	let sizes: string[] = [];
	if (authed || scoreView) {
		sizes = ["w-1/12", "w-4/12", /*"w-3/12",*/ "w-2/12", "w-2/12"];
	} else {
		sizes = ["w-1/12", "w-5/12", /*"w-4/12",*/ "", "w-2/12"];
	}

	return (
		<table class="kn-table table-fixed">
			<thead>
				<tr>
					<th class={`${sizes[0]} py-2`} scope="col">
						#
					</th>
					<th class={sizes[1]} scope="col">
						{getText("name")}
					</th>
					{showTags ? (
						<th class={authed ? "w-3/12" : "w-4/12"} scope="col">
							{getText("tags")}
						</th>
					) : (
						<th class={authed ? "w-3/12" : "w-4/12"} scope="col">
							{getText("source")}
						</th>
					)}
					{(authed || scoreView) && (
						<th class={sizes[2]} scope="col">
							{getText("score")}
						</th>
					)}
					{!scoreView && !latestView && (
						<th class={sizes[3]} scope="col">
							{getText("num_att_solved")}
						</th>
					)}
					{latestView && (
						<th class={sizes[3]} scope="col">
							{getText("published_at")}
						</th>
					)}
				</tr>
			</thead>
			<tbody>
				{problems.length == 0 && (
					<tr class="kn-table-row">
						<td class="kn-table-cell" colSpan={99}>
							<h1>{getText("noPbFound")}</h1>
						</td>
					</tr>
				)}
				{problems.map((pb) => (
					<tr class="kn-table-row" key={pb.id}>
						<td class="text-lg py-2">{pb.id}</td>
						<td>
							<a class="text-lg" href={`/problems/${pb.id}`}>
								{pb.name}
							</a>{" "}
							{((!scoreView && pb.is_editor) || window.platform_info.admin) &&
								(pb.visible ? (
									<span class="badge badge-green text-sm ml-2">{getText("published")}</span>
								) : (
									<span class="badge badge-red text-sm ml-2">{getText("unpublished")}</span>
								))}
						</td>
						{showTags ? (
							<td>{pb.tags.length == 0 ? "-" : pb.tags.map((tag) => <TagView tag={tag} extraClasses="text-sm"></TagView>)}</td>
						) : (
							<td>{pb.source_credits == "" ? "-" : pb.source_credits}</td>
						)}
						{(authed || scoreView) && (
							<td>
								<span class="badge">{pb.max_score < 0 ? "-" : pb.max_score}</span>
							</td>
						)}
						{!scoreView && !latestView && (
							<td>
								<span class="badge">
									{pb.solved_by} {" / "} {pb.attempted_by}
								</span>
							</td>
						)}
						{latestView && <td>{parseTime(pb.published_at) || "N/A"}</td>}
					</tr>
				))}
			</tbody>
		</table>
	);
}

const MAX_PER_PAGE = 50;

// Currently used for user profile and tag pages
export function CustomProblemListing(params: {
	count: number;
	problems: FullProblem[];
	filter: ProblemQuery;
	showFull: boolean;
	showTags: boolean;
	scoreView: boolean;
	latestView: boolean;
	saveHistory: boolean;
	showPages: boolean;
	maxCount?: number;
}) {
	const [page, setPage] = useState(1);
	const [problems, setProblems] = useState<FullProblem[]>(params.problems);
	const [count, setCount] = useState(params.count);

	const mounted = useRef(false);

	async function load() {
		const rez = await bodyCall<{ problems: FullProblem[]; count: number }>(
			"/problem/search",
			serializeQuery({ ...params.filter, page }, params.maxCount ?? MAX_PER_PAGE)
		);
		if (rez.status === "error") {
			apiToast(rez);
			return;
		}
		setCount(rez.data.count);
		setProblems(rez.data.problems);
		if (page > 1) {
			const newName = window.location.pathname + "?page=" + page.toString();
			if (window.location.pathname + window.location.search != newName) {
				history.pushState({}, "", newName);
			}
		} else {
			if (window.location.search.length > 0) {
				history.pushState({}, "", window.location.pathname);
			}
		}
	}

	useEffect(() => {
		if (mounted.current || problems.length == 0) load()?.catch(console.error);
		else mounted.current = true;
	}, [params.filter, page]);

	useEffect(() => {
		const historyPopEvent = async (e) => {
			// TODO: Keep tag groups
			let page = new URLSearchParams(window.location.search).get("page");
			let val = parseInt(page ?? "");
			if (!isNaN(val)) {
				setPage(val);
			} else {
				setPage(1);
			}
		};
		if (params.saveHistory) {
			window.addEventListener("popstate", historyPopEvent);
			return () => window.removeEventListener("popstate", historyPopEvent);
		} else {
			// Make sure
			window.removeEventListener("popstate", historyPopEvent);
		}
	}, [params.saveHistory]);

	return (
		<>
			{params.showPages && numPagesF(count, MAX_PER_PAGE) > 1 && <Paginator numpages={numPagesF(count, MAX_PER_PAGE)} page={page} setPage={setPage} />}
			<ProblemView problems={problems} showTags={params.showTags} scoreView={params.scoreView} latestView={params.latestView} />
		</>
	);
}

type TagGroup = {
	negate: boolean;
	tag_ids: number[];
};

type ProblemOrdering = "" | "id" | "name" | "published_at" | "hot";

type ProblemQuery = {
	textQuery: string;
	page: number;

	tags: TagGroup[];

	deep_list_id?: number;

	published?: boolean;
	editor_user?: number;

	solved_by?: number;
	attempted_by?: number;

	lang?: "ro" | "en";

	score_user_id?: number;

	ordering: ProblemOrdering;
	descending: boolean;
};

function makeTagString(groups: TagGroup[]): string {
	return groups.map((gr) => (gr.negate ? "!" : "") + gr.tag_ids.map((tag) => tag.toString()).join("_")).join(",");
}

function initialQuery(params: URLSearchParams, groups: TagGroup[]): ProblemQuery {
	const page = parseInt(params.get("page") ?? "");
	let published: boolean | undefined;
	let published_str = params.get("published");
	if (published_str === "true" || published_str === "false") {
		published = published_str === "true";
	}

	const editorUserID = parseInt(params.get("editor_user") ?? "");
	const deepListID = parseInt(params.get("deep_list_id") ?? "");

	let ordering: ProblemOrdering = "";
	const ord = params.get("ordering");
	switch (ord) {
		case "id":
		case "name":
		case "published_at":
		case "hot":
			ordering = ord;
			break;
		default:
			ordering = "id";
	}

	let language: "ro" | "en" | undefined;
	const lang = params.get("lang");
	if (lang == "ro" || lang == "en") {
		language = lang;
	}

	return {
		textQuery: params.get("q") ?? "",
		page: !isNaN(page) && page != 0 ? page : 1,

		deep_list_id: !isNaN(deepListID) ? deepListID : undefined,

		published: published,
		editor_user: !isNaN(editorUserID) ? editorUserID : undefined,
		tags: groups,

		lang: language,

		ordering: ordering,
		descending: params.get("descending") === "true",
	};
}

function serializeQuery(f: ProblemQuery, max_cnt: number = MAX_PER_PAGE): any {
	return {
		name_fuzzy: f.textQuery,
		editor_user_id: typeof f.editor_user !== "undefined" && f.editor_user > 0 ? f.editor_user : undefined,
		visible: f.published,

		tags: f.tags,

		deep_list_id: typeof f.deep_list_id !== "undefined" ? f.deep_list_id : undefined,

		solved_by: f.solved_by,
		attempted_by: f.attempted_by,

		lang: f.lang,

		score_user_id: typeof f.score_user_id !== "undefined" ? f.score_user_id : undefined,

		limit: max_cnt,
		offset: (f.page - 1) * max_cnt,

		ordering: f.ordering,
		descending: f.descending,
	};
}

type TagFilterMode = "simple" | "complex";

function getModeByGroups(groups: TagGroup[]): TagFilterMode {
	return groups.some((val) => val.negate || val.tag_ids.length > 1) ? "complex" : "simple";
}

function ProblemSearch(params: { count: number; problems: FullProblem[]; groups: TagGroup[]; initialTags: Tag[]; pblist: ProblemList | null }) {
	let [query, setQuery] = useState<ProblemQuery>(initialQuery(new URLSearchParams(window.location.search), params.groups));
	let [problems, setProblems] = useState<FullProblem[]>(params.problems);
	let [count, setCount] = useState<number>(params.count);
	let numPages = useMemo(() => numPagesF(count, MAX_PER_PAGE), [count]);

	let [tagFilterMode, setTagFilterMode] = useState<TagFilterMode>(getModeByGroups(params.groups));
	let [tags, setTags] = useState<Tag[]>(params.initialTags);

	let [problemList, setProblemList] = useState<ProblemList | null | boolean>(params.pblist);

	const mounted = useRef(false);

	let [showTags, setShowTags] = useState<boolean>(window.platform_info.admin);

	let [advOptions, setAdvOptions] = useState<boolean>(false);

	async function load() {
		const rez = await bodyCall<{ problems: FullProblem[]; count: number }>("/problem/search", serializeQuery(query));
		if (rez.status === "error") {
			apiToast(rez);
			return;
		}
		setCount(rez.data.count);
		setProblems(rez.data.problems);
		let str = getQuery().toString();
		if (str.length > 0) str = "?" + str;
		const newName = window.location.pathname + str;
		if (window.location.pathname + window.location.search != newName) {
			history.pushState({}, "", newName);
		}
	}

	function getQuery() {
		var p = new URLSearchParams();
		console.log(query);
		if (query.textQuery != "") {
			p.append("q", query.textQuery);
		}
		if (query.page != 1) {
			p.append("page", query.page.toString());
		}

		if (typeof query.published !== "undefined") {
			p.append("published", String(query.published));
		}
		if (typeof query.editor_user !== "undefined" && query.editor_user > 0) {
			p.append("editor_user", query.editor_user.toString());
		}

		if (query.tags.length > 0) {
			p.append("tags", makeTagString(query.tags));
		}

		if (typeof query.lang !== "undefined") {
			p.append("lang", query.lang);
		}

		if (typeof query.deep_list_id !== "undefined" && query.deep_list_id > 0) {
			p.append("deep_list_id", query.deep_list_id.toString());
		}
		if (query.ordering !== "id") {
			p.append("ordering", query.ordering);
		}
		if (query.descending) {
			p.append("descending", "true");
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

	function updateTagMode(newMode: TagFilterMode) {
		if (newMode == tagFilterMode) return;
		setTagFilterMode(newMode);
		if (newMode == "simple") {
			if (getModeByGroups(query.tags) == "complex") {
				// It's not possible to turn a complex query into a simple one
				setQuery({ ...query, tags: [] });
			}
		}
	}

	useEffect(() => {
		if (mounted.current) load()?.catch(console.error);
		else mounted.current = true;
	}, [query]);

	useEffect(() => {
		const historyPopEvent = async (e) => {
			// TODO: Keep tag groups
			setQuery(initialQuery(new URLSearchParams(window.location.search), []));
		};
		window.addEventListener("popstate", historyPopEvent);
		return () => window.removeEventListener("popstate", historyPopEvent);
	});

	useEffect(() => {
		console.log(problemList, query);
		const controller = new AbortController();
		if (problemList == null && typeof query.deep_list_id !== "undefined" && query.deep_list_id > 0) {
			getCall<ProblemList>(`/problemList/${query.deep_list_id}`, {}, controller.signal).then((rez) => {
				if (rez.status == "error") {
					setProblemList(false);
					return;
				}
				setProblemList(rez.data);
			});
		}
		return () => controller.abort("Stale");
	}, [query, problemList]);

	return (
		<div class="segment-panel">
			<h1>{getText("problems")}</h1>
			<h2>{rezStr(count, false)}</h2>
			<div class="flex mx-auto gap-2 align-middle my-2">
				<input
					class="form-input grow"
					type="text"
					placeholder={getText("search_pb")}
					onInput={(e) => {
						setQuery({
							...query,
							page: 1,
							textQuery: e.currentTarget.value,
						});
					}}
					value={query.textQuery}
					autofocus={true}
				/>
				<button class="btn btn-blue" onClick={() => setAdvOptions(!advOptions)}>
					{getText("advancedFilters")} <i class={`ml-1 fas ${advOptions ? "fa-caret-up" : "fa-caret-down"}`}></i>
				</button>
			</div>

			{advOptions && (
				<div class="segment-panel">
					{window.platform_info.user?.proposer && (
						<label class="block my-2">
							<span class="form-label">{getText("author")}: </span>
							<input
								type="number"
								class="form-input"
								value={query.editor_user == 0 ? "" : query.editor_user}
								onChange={(e) => {
									let val: number | null = parseInt(e.currentTarget.value);
									if (isNaN(val) || val <= 0) {
										val = null;
									}
									setQuery({
										...query,
										page: 1,
										editor_user: val == null ? 0 : val,
									});
								}}
							/>
						</label>
					)}
					{window.platform_info.admin && (
						<label class="block mb-2">
							<span class="form-label">{getText("published")}:</span>
							<select
								class="form-select"
								value={query.published === undefined ? "" : String(query.published)}
								onChange={(e) => {
									let val = e.currentTarget.value;
									let published: boolean | undefined;
									if (val == "true" || val == "false") {
										published = val === "true";
									}
									setQuery({
										...query,
										page: 1,
										published: published,
									});
								}}
							>
								<option value="">-</option>
								<option value="true">{getText("yes")}</option>
								<option value="false">{getText("no")}</option>
							</select>
						</label>
					)}
					<div class="block my-2">
						<span class="form-label">{getText("filter_tags")}</span>
						<select class="form-select" value={tagFilterMode} onChange={(e) => updateTagMode(e.currentTarget.value as TagFilterMode)}>
							<option value={"simple"}>{getText("tag_filter_simple_mode")}</option>
							<option value={"complex"}>{getText("tag_filter_complex_mode")}</option>
						</select>
						{": "}
						{tagFilterMode == "simple" ? (
							<>
								{query.tags.length == 0
									? getText("no_selected_tags")
									: query.tags.map((gr) => gr.tag_ids.map((id) => <TagView tag={tags.find((t) => t.id == id)!} link={false} />))}
								<a
									class="mx-1"
									href="#"
									onClickCapture={(e) => {
										e.preventDefault();
										selectTags(tags).then((rez) => {
											if (rez.updated) {
												setTags(rez.tags);
												setQuery({
													...query,
													tags: rez.tags.map((t) => ({ negate: false, tag_ids: [t.id] })),
												});
											}
										});
									}}
								>
									<i class="fas fa-pen-to-square"></i> {query.tags.length === 0 ? getText("add_tags") : getText("select_tags")}
								</a>
							</>
						) : (
							<div class="block my-2 reset-list">
								<span class="form-label">{getText("tag_complex_explainer")}:</span>
								<ul>
									{query.tags.map((val, idx) => (
										<li>
											<span
												onClick={(e) => {
													e.preventDefault();
													setQuery({ ...query, tags: query.tags.filter((_, idx1) => idx != idx1) });
												}}
												class="light-btn fas fa-xmark text-red-600"
											></span>{" "}
											{getText("tag_complex_group", idx + 1)}:{" "}
											<span
												onClick={(e) => {
													e.preventDefault();
													setQuery({
														...query,
														tags: query.tags.map((val, idx1) => {
															if (idx != idx1) return val;
															return { negate: !val.negate, tag_ids: val.tag_ids };
														}),
													});
												}}
												class="light-btn px-2"
											>
												!
											</span>{" "}
											{val.negate && getText("tag_complex_not")}{" "}
											{val.tag_ids.map((id) => (
												<TagView tag={tags.find((t) => t.id == id)!} link={false} />
											))}
											<a
												class="mx-1"
												href="#"
												onClickCapture={(e) => {
													e.preventDefault();
													selectTags(
														val.tag_ids.map((id) => tags.find((t) => t.id == id)!),
														false
													).then((rez) => {
														if (rez.updated && rez.tags.length > 0) {
															let missingTags: Tag[] = [];
															rez.tags.forEach((t) => {
																if (!tags.some((tag) => tag.id == t.id)) missingTags.push(t);
															});
															if (missingTags.length > 0) {
																setTags([...tags, ...missingTags]);
															}
															console.log(tags);
															setQuery({
																...query,
																tags: query.tags.map((val, idx1) => {
																	if (idx != idx1) return val;
																	return { negate: val.negate, tag_ids: rez.tags.map((t) => t.id) };
																}),
															});
														}
													});
												}}
											>
												<i class="fas fa-pen-to-square"></i> {getText("update_tags")}
											</a>
										</li>
									))}
									<li>
										<a
											class="mx-1"
											href="#"
											onClickCapture={(e) => {
												e.preventDefault();
												selectTags([], false).then((rez) => {
													if (rez.updated && rez.tags.length > 0) {
														let missingTags: Tag[] = [];
														rez.tags.forEach((t) => {
															if (!tags.some((tag) => tag.id == t.id)) missingTags.push(t);
														});
														if (missingTags.length > 0) {
															setTags([...tags, ...missingTags]);
														}
														console.log(tags);
														setQuery({
															...query,
															tags: [...query.tags, { negate: false, tag_ids: rez.tags.map((t) => t.id) }],
														});
													}
												});
											}}
										>
											<i class="fas fa-pen-to-square"></i> {getText("tag_complex_add_group")}
										</a>
									</li>
								</ul>
							</div>
						)}
					</div>
					<label class="block my-2">
						<span class="form-label">{getText("pblist_id")}: </span>
						<input
							type="number"
							class="form-input"
							value={query.deep_list_id == 0 || typeof query.deep_list_id == "undefined" ? "" : query.deep_list_id}
							onChange={(e) => {
								let val: number | null = parseInt(e.currentTarget.value);
								if (isNaN(val) || val <= 0) {
									val = null;
								}
								setQuery({
									...query,
									page: 1,
									deep_list_id: val == null ? undefined : val,
								});
								setProblemList(null);
							}}
						/>
					</label>
					<label class="block my-2">
						<span class="form-label">{getText("statementLanguage")}:</span>
						<select
							class="form-select"
							autocomplete="off"
							value={typeof query.lang == "undefined" ? "" : query.lang}
							onChange={(e) => {
								let val: "ro" | "en" | undefined;
								let cval = e.currentTarget.value;
								if (cval != "") {
									if (cval == "ro" || cval == "en") {
										val = cval;
									} else {
										val = undefined;
									}
								}
								setQuery({ ...query, page: 1, lang: val });
							}}
						>
							<option value="">-</option>
							<option value="ro">🇷🇴 Română</option>
							<option value="en">🇬🇧 English</option>
						</select>
					</label>
				</div>
			)}

			{typeof query.deep_list_id !== "undefined" && query.deep_list_id > 0 && typeof problemList !== "boolean" && (
				<span class="block my-2">
					{getText("viewingFromList")}{" "}
					<a href={`/problem_lists/${query.deep_list_id}`}>{problemList == null ? getText("loading") : problemList.title}</a>.
				</span>
			)}

			<div class="block my-2">
				<button class="btn btn-blue mr-2" onClick={() => load()}>
					{getText("button.filter")}
				</button>
				<button class="btn" onClick={async () => await copyQuery()}>
					{getText("filterLink")}
				</button>
			</div>

			{count > 0 && (
				<div class="block my-2">
					<Paginator
						numpages={numPages}
						page={query.page}
						setPage={(num) => {
							setQuery({ ...query, page: num });
						}}
						showArrows={true}
					/>
					<div class="topbar-separator text-xl mx-2"></div>
					<label>
						<input
							type="checkbox"
							onChange={(e) => {
								setShowTags(e.currentTarget.checked);
							}}
							checked={showTags}
						></input>{" "}
						<span class="form-label">{getText("show_tags")}</span>
					</label>
				</div>
			)}
			<ProblemView problems={problems} showTags={showTags} scoreView={false} latestView={false} />
		</div>
	);
}

function ProblemSearchDOM({ enc, count, groupenc, tagenc, pblistenc }: { enc: string; count: string; groupenc: string; tagenc: string; pblistenc: string }) {
	let pbs: FullProblem[] = JSON.parse(fromBase64(enc));
	let cnt = parseInt(count);
	if (isNaN(cnt)) {
		throw new Error("Invalid count");
	}
	let groups: TagGroup[] = JSON.parse(fromBase64(groupenc));
	let tags: Tag[] = JSON.parse(fromBase64(tagenc));
	let pblist: ProblemList | null = JSON.parse(fromBase64(pblistenc));
	return <ProblemSearch problems={pbs} count={cnt} groups={groups} initialTags={tags} pblist={pblist}></ProblemSearch>;
}

function ProblemListingWrapper({
	enc,
	count,
	showTags,
	showfull,
	filter,
	scoreView,
	latestView,
	saveHistory,
	showPages,
	maxCount,
}: {
	enc: string;
	count: string;
	showTags: boolean;
	showfull: string;
	filter: ProblemQuery;
	scoreView: boolean;
	latestView?: boolean;
	saveHistory: boolean;
	showPages?: boolean;
	maxCount?: number;
}) {
	let pbs: FullProblem[] = JSON.parse(fromBase64(enc));
	let cnt = parseInt(count);
	if (isNaN(cnt)) {
		throw new Error("Invalid count");
	}
	return (
		<CustomProblemListing
			problems={pbs}
			count={cnt}
			showTags={showTags}
			showFull={showfull === "true"}
			filter={filter}
			scoreView={scoreView}
			latestView={typeof latestView === "undefined" ? true : latestView}
			saveHistory={saveHistory}
			showPages={typeof showPages === "undefined" ? true : showPages}
			maxCount={maxCount}
		></CustomProblemListing>
	);
}

function TagProblemsDOM({ enc, count, tagid }: { enc: string; count: string; tagid: string }) {
	let tagID = parseInt(tagid);
	if (isNaN(tagID)) {
		throw new Error("Invalid tag ID");
	}
	let [showTags, setShowTags] = useState<boolean>(false);
	return (
		<>
			<div class="block mb-2">
				<a class="btn btn-blue" href={`/problems?tags=${tagID}`}>
					{getText("use_in_search")}
				</a>
				<div class="topbar-separator text-xl mx-2"></div>
				<label>
					<input
						type="checkbox"
						onChange={(e) => {
							setShowTags(e.currentTarget.checked);
						}}
						checked={showTags}
					></input>{" "}
					<span class="form-label">{getText("show_tags")}</span>
				</label>
			</div>

			<div>
				<ProblemListingWrapper
					enc={enc}
					count={count}
					showTags={showTags}
					showfull="true"
					filter={{ textQuery: "", tags: [{ tag_ids: [tagID], negate: false }], page: 1, descending: false, ordering: "" }}
					scoreView={false}
					saveHistory={true}
				/>
			</div>
		</>
	);
}

function ProblemSolvedByDOM({ enc, count, userid }: { enc: string; count: string; userid: string }) {
	let uid = parseInt(userid);
	if (isNaN(uid)) {
		throw new Error("Invalid user ID");
	}
	return (
		<ProblemListingWrapper
			enc={enc}
			count={count}
			showfull="false"
			filter={{ textQuery: "", tags: [], page: 1, solved_by: uid, score_user_id: uid, descending: false, ordering: "" }}
			scoreView={true}
			showTags={false}
			saveHistory={false}
		/>
	);
}

function ProblemAttemptedByDOM({ enc, count, userid }: { enc: string; count: string; userid: string }) {
	let uid = parseInt(userid);
	if (isNaN(uid)) {
		throw new Error("Invalid user ID");
	}
	return (
		<ProblemListingWrapper
			enc={enc}
			count={count}
			showfull="false"
			filter={{ textQuery: "", tags: [], page: 1, attempted_by: uid, score_user_id: uid, descending: false, ordering: "" }}
			scoreView={true}
			showTags={false}
			saveHistory={false}
		/>
	);
}

function LatestProblemsDOM({ enc }: { enc: string }) {
	let [showTags, setShowTags] = useState<boolean>(false);
	return (
		<>
			<div class="block mb-2">
				<label>
					<input
						type="checkbox"
						onChange={(e) => {
							setShowTags(e.currentTarget.checked);
						}}
						checked={showTags}
					></input>{" "}
					<span class="form-label">{getText("show_tags")}</span>
				</label>
			</div>
			<ProblemListingWrapper
				enc={enc}
				count={"-1"}
				showfull="true"
				filter={{ textQuery: "", tags: [], page: 1, descending: true, ordering: "published_at" }}
				scoreView={false}
				latestView={true}
				showTags={showTags}
				saveHistory={false}
				showPages={false}
				maxCount={20}
			/>
		</>
	);
}

register(ProblemSearchDOM, "kn-pb-search", ["enc", "count", "groupenc", "tagenc", "pblistenc"]);
register(TagProblemsDOM, "kn-tag-pbs", ["enc", "count", "tagid"]);
register(LatestProblemsDOM, "kn-pb-latest", ["enc"]);
register(ProblemSolvedByDOM, "kn-pb-solvedby", ["enc", "count", "userid"]);
register(ProblemAttemptedByDOM, "kn-pb-attemptedby", ["enc", "count", "userid"]);
