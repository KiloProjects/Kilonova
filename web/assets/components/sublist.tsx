import { fromBase64 } from "js-base64";
import { h, Fragment } from "preact";
import register from "preact-custom-element";
import { useEffect, useState } from "preact/hooks";
import { getCall } from "../net.js";
import { apiToast } from "../toast.js";
import getText from "../translation.js";
import { BigSpinner } from "./common.js";

type Sublist = {
	id: number;
	title: string;
	author_id: number;
	list: number[];
};

type FullList = {
	id: number;
	title: string;
	description: string;
	author_id: number;
	list: number[];
	sublists: Sublist[];
};

type Problem = {
	id: number;
	name: string;
	author_id: number;
	visible: boolean;
};

type ProblemScore = { [problem: number]: number };

export function Problems({ pbs, scores }: { pbs: Problem[]; scores: ProblemScore }) {
	return (
		<div class="list-group list-group-mini">
			{pbs.map((pb) => (
				<a href={`/problems/${pb.id}`} class="list-group-item flex justify-between" key={pb.id}>
					<span>
						{pb.name} (#{pb.id})
					</span>
					{window.platform_info.user_id > 0 && (
						<div>
							{(window.platform_info.admin || window.platform_info.user_id == pb.author_id) &&
								(pb.visible ? (
									<span class="badge badge-green">{getText("published")}</span>
								) : (
									<span class="badge badge-red">{getText("unpublished")}</span>
								))}{" "}
							{Object.keys(scores).includes(pb.id.toString()) && <span class="badge">{scores[pb.id]}</span>}
						</div>
					)}
				</a>
			))}
		</div>
	);
}

type QueryResult = {
	list: FullList;
	numSolved: number;
	description: string;
	problems: Problem[];
	problemScores: ProblemScore;
};

export function Sublist({ list }: { list: Sublist }) {
	let [loading, setLoading] = useState(false);
	let [expanded, setExpanded] = useState(false);
	let [fullData, setFullData] = useState<FullList | undefined>(undefined);
	let [numSolved, setNumSolved] = useState(-1);
	let [descHTML, setDescHTML] = useState("");
	let [problems, setProblems] = useState<Problem[]>([]);
	let [problemScores, setProblemScores] = useState<ProblemScore>({});

	async function load() {
		setLoading(true);
		const res = await getCall<QueryResult>("/problemList/getComplex", { id: list.id });
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		setFullData(res.data.list);
		setNumSolved(res.data.numSolved);
		setDescHTML(res.data.description);
		setProblems(res.data.problems);
		setProblemScores(res.data.problemScores);
		setLoading(false);
	}

	useEffect(() => {
		if (expanded && fullData == undefined && !loading) {
			load();
		}
	}, [expanded]);

	return (
		<details class="list-group-head" onToggle={(e) => setExpanded(e.currentTarget.open)}>
			<summary class="pb-1 mt-1">
				<span class="float-left">
					{list.title}{" "}<a href={`/problem_lists/${list.id}`}>(#{list.id})</a>
				</span>
				{list.list.length > 0 &&
					((numSolved >= 0 && <span class="float-right badge">{getText("num_solved", numSolved, list.list.length)}</span>) || (
						<span class="float-right badge">{list.list.length == 1 ? getText("single_problem") : getText("num_problems", list.list.length)}</span>
					))}
			</summary>
			{loading && <BigSpinner />}
			{fullData && (
				<>
					{descHTML && (
						<div class="list-group list-group-mini mt-2">
							<div class="list-group-head" dangerouslySetInnerHTML={{ __html: descHTML }}></div>
						</div>
					)}
					{fullData.sublists.length > 0 && (
						<div class="list-group list-group-mini mt-2">
							{fullData.sublists.map((val) => (
								<Sublist list={val} key={val.id.toString() + "_" + list.id.toString()} />
							))}
						</div>
					)}
					{problems.length > 0 && (
						<div class="mt-2">
							<Problems pbs={problems} scores={problemScores} />
						</div>
					)}
				</>
			)}
		</details>
	);
}

export function DOMSublist({ encoded }: { encoded: string }) {
	console.log(fromBase64(encoded));

	return <Sublist list={JSON.parse(fromBase64(encoded))}></Sublist>;
}

register(DOMSublist, "kn-dom-sublist", ["encoded"]);
