import { h, Fragment, render } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";
import { apiToast } from "../toast";
import type { Problem, UserBrief, Submission } from "../api/submissions";
import { bodyCall, getCall } from "../api/net";
import { fromBase64 } from "js-base64";
import { KNModal } from "./modal";
import { sizeFormatter } from "../util";
import { Tag, TagView } from "./tags";
import { Problems } from "./sublist";
import { Paginator } from "./common";
import { rezStr } from "./subs_view";
import { throttle } from "lodash-es";

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
const initialParams = new URLSearchParams(window.location.search);

function ProblemSearch(params: { count: number; problems: FullProblem[] }) {
	let [textQuery, setTextQuery] = useState<string>(initialParams.get("q") ?? "");

	let [count, setCount] = useState<number>(params.count);
	let [problems, setProblems] = useState<FullProblem[]>(params.problems);

	let [page, setPage] = useState<number>(1);
	let [numPages, setNumPages] = useState<number>(Math.floor(count / MAX_PER_PAGE) + (count % MAX_PER_PAGE != 0 ? 1 : 0));

	const mounted = useRef(false);

	let [showTags, setShowTags] = useState<boolean>(window.platform_info.admin);

	async function load() {
		const rez = await bodyCall<{ problems: FullProblem[]; count: number }>("/problem/search", {
			name_fuzzy: textQuery,
			limit: MAX_PER_PAGE,
			offset: (page - 1) * MAX_PER_PAGE,
		});
		if (rez.status === "error") {
			apiToast(rez);
			return;
		}
		setCount(rez.data.count);
		setProblems(rez.data.problems);
		setNumPages(Math.floor(rez.data.count / MAX_PER_PAGE) + (rez.data.count % MAX_PER_PAGE != 0 ? 1 : 0));
	}

	useEffect(() => {
		if (mounted.current) load()!.catch(console.error);
		else mounted.current = true;
	}, [page, textQuery]);

	return (
		<div class="segment-panel">
			<h1>{getText("problems")}</h1>
			<h2>{rezStr(count)}</h2>
			<input
				class="form-input"
				type="text"
				onInput={(e) => {
					setPage(1);
					setTextQuery(e.currentTarget.value);
				}}
				value={textQuery}
			/>

			<Paginator numpages={numPages} page={page} setPage={setPage} showArrows={true} />

			<label class="block my-2">
				<input
					type="checkbox"
					onChange={(e) => {
						setShowTags(e.currentTarget.checked);
					}}
					checked={showTags}
				></input>{" "}
				<span class="form-label">{getText("tags")}</span>
			</label>

			<ProblemView problems={problems} showTags={showTags} />
		</div>
	);
}

function ProblemSearchDOM({ enc, count }: { enc: string; count: string }) {
	let pbs: FullProblem[] = JSON.parse(fromBase64(enc));
	let cnt = parseInt(count);
	if (isNaN(cnt)) {
		throw new Error("Invalid count");
	}
	return <ProblemSearch problems={pbs} count={cnt}></ProblemSearch>;
}

register(ProblemSearchDOM, "kn-pb-search", ["enc", "count"]);
