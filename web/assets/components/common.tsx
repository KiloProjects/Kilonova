import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import getText from "../translation.js";
import { getCall } from "../net";
import { dayjs } from "../util";

interface PaginatorParams {
	page: number;
	numPages: number;
	setPage: (num: number) => void;
}

import { apiToast } from "../toast";
export function Paginator({ page, numPages, setPage }: PaginatorParams) {
	// props format: {page: number, setPage(): handler for updating, numPages: number}
	if (page < 1 || page > numPages) {
		console.warn("Invalid page number");
		apiToast({ status: "error", data: "Invalid page number" });
	}
	let elements: preact.JSX.Element[] = [];
	const old_sp = setPage;
	setPage = (pg) => {
		if (pg < 1) {
			pg = 1;
		}
		if (pg > numPages) {
			pg = numPages;
		}
		old_sp(pg);
	};

	//elements.push(<button class="paginator-item" onClick={() => setPage(1)}><i class="fas fa-angle-double-left"></i></button>);
	//elements.push(<button class="paginator-item" onClick={() => setPage(page-1)}><i class="fas fa-angle-left"></i></button>);
	if (page > 3) {
		for (let i = 1; i <= 3 && page - i >= 3; i++) {
			elements.push(
				<button class="paginator-item" onClick={() => setPage(i)}>
					{i}
				</button>
			);
		}
		if (page > 6) {
			elements.push(<button class="paginator-item">...</button>);
		}
	}

	if (page - 2 > 0) {
		elements.push(
			<button class="paginator-item" onClick={() => setPage(page - 2)}>
				{page - 2}
			</button>
		);
	}
	if (page - 1 > 0) {
		elements.push(
			<button class="paginator-item" onClick={() => setPage(page - 1)}>
				{page - 1}
			</button>
		);
	}
	elements.push(
		<button class="paginator-item paginator-item-active">{page}</button>
	);
	if (page + 1 <= numPages) {
		elements.push(
			<button class="paginator-item" onClick={() => setPage(page + 1)}>
				{page + 1}
			</button>
		);
	}
	if (page + 2 <= numPages) {
		elements.push(
			<button class="paginator-item" onClick={() => setPage(page + 2)}>
				{page + 2}
			</button>
		);
	}

	if (numPages - page >= 3) {
		if (numPages - page > 5) {
			elements.push(<button class="paginator-item">...</button>);
		}
		for (let i = numPages - 2; i <= numPages; i++) {
			if (i - page <= 2) {
				continue;
			}
			elements.push(
				<button class="paginator-item" onClick={() => setPage(i)}>
					{i}
				</button>
			);
		}
	}

	//elements.push(<button class="paginator-item" onClick={() => setPage(page+1)}><i class="fas fa-angle-right"></i></button>);
	//elements.push(<button class="paginator-item" onClick={() => setPage(numPages)}><i class="fas fa-angle-double-right"></i></button>);
	return <div class="paginator">{elements}</div>;
}

import { useCallback, useEffect, useState } from "preact/hooks";

export function PaginateTester() {
	let [pg, setPg] = useState(1);
	let [maxPg, setMaxPg] = useState(5);
	return (
		<div>
			<input
				type="number"
				class="form-input"
				value={pg}
				onChange={(e) => {
					setPg(
						Number.parseInt((e.target as HTMLInputElement).value)
					);
				}}
			/>
			<br />
			<input
				type="number"
				class="form-input"
				value={maxPg}
				onChange={(e) => {
					setMaxPg(
						Number.parseInt((e.target as HTMLInputElement).value)
					);
				}}
			/>
			<br />
			<br />
			<Paginator page={pg} numPages={maxPg} setPage={(pg) => setPg(pg)} />
		</div>
	);
}

export function BigSpinner() {
	return (
		<div class="text-4xl mx-auto w-full my-10 text-center">
			<div>
				<i class="fas fa-spinner animate-spin"></i> {getText("loading")}
			</div>
		</div>
	);
}

export function SmallSpinner() {
	return (
		<div class="mx-auto my-auto w-full text-center">
			<div>
				<i class="fas fa-spinner animate-spin"></i> {getText("loading")}
			</div>
		</div>
	);
}

export function InlineSpinner() {
	return (
		<div class="mx-auto w-full text-center">
			<div>
				<i class="fas fa-spinner animate-spin"></i> {getText("loading")}
			</div>
		</div>
	);
}

export function Segment(props) {
	return <div className="segment-container">{props.children}</div>;
}

export function Button(props) {
	return <button className="btn btn-blue">{props.children}</button>;
}

export function RedButton(props) {
	return <button className="btn btn-red">{props.children}</button>;
}

export function ProblemAttachment({ attname = "" }) {
	let pname = window.location.pathname;
	if (pname.endsWith("/")) {
		console.log("cf");
		pname = pname.substr(0, pname.lastIndexOf("/"));
	}
	return <img src={`${pname}/attachments/${attname}`} />;
}

interface OlderSubsParams {
	userid: number;
	problemid: number;
}

type shortSub = {
	sub: {
		id: number;
		created_at: string;
		status: string;
		score: number;
	};
};

type getSubsResult = {
	count: number;
	subs: shortSub[];
};

async function getSubmissions(
	user_id: number,
	problem_id: number,
	limit: number
): Promise<getSubsResult> {
	const result = await getCall("/submissions/get", {
		limit,
		problem_id,
		user_id,
	});
	if (result.status !== "success") {
		throw new Error(result.data);
	}
	return result.data as Promise<getSubsResult>;
}

const SUB_VIEW_LIMIT = 5;

export function OlderSubmissions({ userid, problemid }: OlderSubsParams) {
	let [subs, setSubs] = useState<shortSub[]>([]);
	let [loading, setLoading] = useState(true);
	let [numHidden, setNumHidden] = useState(0);

	const loader = useCallback(async () => {
		const data = await getSubmissions(userid, problemid, SUB_VIEW_LIMIT);
		setSubs(data.subs);
		setNumHidden(Math.max(data.count - SUB_VIEW_LIMIT, 0));
		setLoading(false);
	}, []);

	useEffect(() => {
		loader().catch(console.error);
	}, [userid, problemid]);

	useEffect(() => {
		const poll = async (e) => loader();
		document.addEventListener("kn-poll", poll);
		return () => {
			document.removeEventListener("kn-poll", poll);
		};
	}, []);

	return (
		<>
			<h2 class=" mb-2 px-2">{getText("oldSubs")}</h2>
			{loading ? (
				<InlineSpinner />
			) : (
				<>
					{subs.length > 0 ? (
						<div>
							{subs.map((sub) => (
								<a
									href={`/submissions/${sub.sub.id}`}
									class="black-anchor flex justify-between items-center rounded py-1 px-2 hoverable"
									key={sub.sub.id}
								>
									<span>
										{`#${sub.sub.id}: ${dayjs(
											sub.sub.created_at
										).format("DD/MM/YYYY HH:mm")}`}
									</span>
									<span class="rounded-md px-2 py-1 bg-teal-700 text-white text-sm">
										{{
											finished: <>{sub.sub.score}</>,
											working: (
												<i class="fas fa-cog animate-spin"></i>
											),
										}[sub.sub.status] || (
											<i class="fas fa-clock"></i>
										)}
									</span>
								</a>
							))}
						</div>
					) : (
						<p class="px-2">{getText("noSub")}</p>
					)}
					{numHidden > 0 && (
						<a
							class="px-2"
							href={`/submissions/?problem_id=${problemid}&user_id=${userid}`}
						>
							{getText(
								numHidden == 1
									? "seeOne"
									: numHidden < 20
									? "seeU20"
									: "seeMany",
								numHidden
							)}
						</a>
					)}
				</>
			)}
		</>
	);
}

register(OlderSubmissions, "older-subs", ["userid", "problemid"]);
register(ProblemAttachment, "problem-attachment", ["attname"]);

function ProgressChecker({ id }: { id: number }) {
	var [computable, setComputable] = useState<boolean>(false);
	var [loaded, setLoaded] = useState<number>(0);
	var [total, setTotal] = useState<number>(0);

	useEffect(() => {
		const upd = (e: CustomEvent<ProgressEventData>) => {
			if (e.detail.id == id) {
				setLoaded(e.detail.cntLoaded);
				setTotal(e.detail.cntTotal);
				setComputable(e.detail.computable);
			}
		};
		document.addEventListener("kn-upload-update", upd);
		return () => {
			document.removeEventListener("kn-upload-update", upd);
		};
	}, []);

	return (
		<>
			<div class="block">
				<progress value={computable ? loaded / total : undefined} />
			</div>

			{computable && <span>{Math.floor((loaded / total) * 100)}%</span>}
		</>
	);
}

register(ProgressChecker, "upload-progress", ["id"]);
