import { h, Fragment, render } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";
import { apiToast, createToast } from "../toast";
import type { Problem, UserBrief, Submission } from "../api/submissions";
import { bodyCall, getCall } from "../api/net";
import { fromBase64 } from "js-base64";
import { KNModal } from "./modal";
import { sizeFormatter } from "../util";
import { Tag, TagView } from "./tags";
import { Problems } from "./sublist";
import { Paginator } from "./common";
import { rezStr } from "./subs_view";

type FullProblem = Problem & {
	tags: Tag[];
	max_score: number;
	is_editor: boolean;

	solved_by: number;
	attempted_by: number;
};

type ProblemFilter = {
	name_fuzzy?: string;

	limit: number;
	offset: number;
};

const widths = {
	noTagsAuthed: ["w-1/12", "w-6/12", "w-2/12", "w-3/12"],
	tagsAuthed: ["w-1/12", "w-4/12", /*"w-3/12",*/ "w-2/12", "w-2/12"],
	noTagsNotAuthed: ["w-1/12", "w-7/12", "", "w-4/12"],
	tagsNotAuthed: ["w-1/12", "w-5/12", /*"w-4/12",*/ "", "w-2/12"],
};

function ProblemView({ problems, showTags }: { problems: FullProblem[]; showTags: boolean }) {
	let authed = window.platform_info.user_id >= 1;
	let sizes: string[] = [];
	if (authed) {
		sizes = widths[showTags ? "tagsAuthed" : "noTagsAuthed"];
	} else {
		sizes = widths[showTags ? "tagsNotAuthed" : "noTagsNotAuthed"];
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
					{showTags && (
						<th class={authed ? "w-3/12" : "w-4/12"} scope="col">
							{getText("tags")}
						</th>
					)}
					{authed && (
						<th class={sizes[2]} scope="col">
							{getText("score")}
						</th>
					)}
					<th class={sizes[3]} scope="col">
						{getText("num_att_solved")}
					</th>
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
							{pb.is_editor &&
								(pb.visible ? (
									<span class="badge badge-green text-sm ml-2">{getText("published")}</span>
								) : (
									<span class="badge badge-red text-sm ml-2">{getText("unpublished")}</span>
								))}
						</td>
						{showTags && <td>{pb.tags.length == 0 ? "-" : pb.tags.map((tag) => <TagView tag={tag} extraClasses="text-sm"></TagView>)}</td>}
						{authed && (
							<td>
								<span class="badge">{pb.max_score < 0 ? "-" : pb.max_score}</span>
							</td>
						)}
						<td>
							<span class="badge">
								{pb.solved_by} {" / "} {pb.attempted_by}
							</span>
						</td>
					</tr>
				))}
			</tbody>
		</table>
	);
}

const MAX_PER_PAGE = 50;

type TagGroup = {
	negate: boolean;
	tag_ids: number[];
};

type ProblemQuery = {
	textQuery: string;
	page: number;

	tags: TagGroup[];

	published?: boolean;
	editor_user: number;
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

	return {
		textQuery: params.get("q") ?? "",
		page: !isNaN(page) && page != 0 ? page : 1,

		published: published,
		editor_user: !isNaN(editorUserID) ? editorUserID : 0,
		tags: groups,
	};
}

function parseTagString(tagString: string | null): TagGroup[] {
	if (tagString == null) {
		return [];
	}
	let tags: TagGroup[] = [];
	for (let group of tagString.split(",")) {
		if (group.length === 0) {
			continue;
		}
		let negate = false;
		if (group[0] == "!") {
			negate = true;
			group = group.replace(/^\!/, "");
		}

		let tag_ids: number[] = group
			.split("_")
			.map((val) => parseInt(val))
			.filter((val) => !isNaN(val));

		if (tag_ids.length > 0) {
			tags.push({
				negate,
				tag_ids,
			});
		}
	}

	return tags;
}

function serializeQuery(f: ProblemQuery, tagString: string): any {
	return {
		name_fuzzy: f.textQuery,
		editor_user_id: f.editor_user > 0 ? f.editor_user : undefined,
		visible: f.published,

		// tags: f.tags,
		tags: parseTagString(tagString),

		limit: MAX_PER_PAGE,
		offset: (f.page - 1) * MAX_PER_PAGE,
	};
}

function ProblemSearch(params: { count: number; problems: FullProblem[]; groups: TagGroup[]; initialTags: Tag[] }) {
	let [query, setQuery] = useState<ProblemQuery>(initialQuery(new URLSearchParams(window.location.search), params.groups));
	let [problems, setProblems] = useState<FullProblem[]>(params.problems);
	let [count, setCount] = useState<number>(params.count);
	let numPages = useMemo(() => Math.floor(count / MAX_PER_PAGE) + (count % MAX_PER_PAGE != 0 ? 1 : 0), [count]);
	let [tags, setTags] = useState<Tag[]>(params.initialTags);

	// TODO: Remove once proper tag filtering is implemented
	let [tagString, setTagString] = useState<string>(makeTagString(query.tags));

	const mounted = useRef(false);

	let [showTags, setShowTags] = useState<boolean>(window.platform_info.admin);

	let [advOptions, setAdvOptions] = useState<boolean>(false);

	async function load() {
		const rez = await bodyCall<{ problems: FullProblem[]; count: number }>("/problem/search", serializeQuery(query, tagString));
		if (rez.status === "error") {
			apiToast(rez);
			return;
		}
		setCount(rez.data.count);
		setProblems(rez.data.problems);
	}

	async function copyQuery() {
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
		if (query.editor_user > 0) {
			p.append("editor_user", query.editor_user.toString());
		}

		if (query.tags.length > 0) {
			// TODO: Uncomment when advanced tag filtering is finished
			// p.append("tags", makeTagString(query.tags));
			p.append("tags", tagString);
		}

		let url = window.location.origin + window.location.pathname + "?" + p.toString();
		try {
			await navigator.clipboard.writeText(url);
			createToast({ status: "success", title: getText("copied") });
		} catch (e) {
			console.error(e);
			createToast({ status: "error", title: getText("notCopied") });
		}
	}

	useEffect(() => {
		if (mounted.current) load()?.catch(console.error);
		else mounted.current = true;
	}, [query]);

	return (
		<div class="segment-panel">
			<h1>{getText("problems")}</h1>
			<h2>{rezStr(count)}</h2>
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
				/>
				<button class="btn btn-blue" onClick={() => setAdvOptions(!advOptions)}>
					{getText("advancedOptions")} <i class={`ml-1 fas ${advOptions ? "fa-caret-up" : "fa-caret-down"}`}></i>
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
					<label class="block my-2">
						<span class="form-label">{getText("tag_string")}: </span>
						<input
							type="text"
							class="form-input"
							value={tagString}
							placeholder="!1,2_3,4_5_6,9"
							onChange={(e) => {
								setTagString(e.currentTarget.value);
							}}
						/>
					</label>
				</div>
			)}

			<div class="block my-2">
				<button class="btn btn-blue mr-2" onClick={() => load()}>
					{getText("button.filter")}
				</button>
				<button class="btn" onClick={async () => await copyQuery()}>
					{getText("filterLink")}
				</button>
			</div>

			<label class="block my-2">
				<input
					type="checkbox"
					onChange={(e) => {
						setShowTags(e.currentTarget.checked);
					}}
					checked={showTags}
				></input>{" "}
				<span class="form-label">{getText("show_tags")}</span>
			</label>
			{count > 0 && (
				<Paginator
					numpages={numPages}
					page={query.page}
					setPage={(num) => {
						setQuery({ ...query, page: num });
					}}
					showArrows={true}
				/>
			)}
			<ProblemView problems={problems} showTags={showTags} />
		</div>
	);
}

function ProblemSearchDOM({ enc, count, groupenc, tagenc }: { enc: string; count: string; groupenc: string; tagenc: string }) {
	let pbs: FullProblem[] = JSON.parse(fromBase64(enc));
	let cnt = parseInt(count);
	if (isNaN(cnt)) {
		throw new Error("Invalid count");
	}
	let groups: TagGroup[] = JSON.parse(fromBase64(groupenc));
	let tags: Tag[] = JSON.parse(fromBase64(tagenc));
	console.log(groups, tags);
	return <ProblemSearch problems={pbs} count={cnt} groups={groups} initialTags={tags}></ProblemSearch>;
}

register(ProblemSearchDOM, "kn-pb-search", ["enc", "count", "groupenc", "tagenc"]);
