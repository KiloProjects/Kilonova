import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import getText from "../translation.js";
import { getCall } from "../net";
import { dayjs } from "../util";
import { useCallback, useEffect, useState } from "preact/hooks";
import { getSubmissions, ResultSubmission } from "../api/submissions.js";

interface PaginatorParams {
	page: number;
	numpages: number;
	setPage: (num: number) => void;
	ctxSize?: number;
	className?: string;
	showArrows?: boolean;
}

export function Paginator({ page, numpages, setPage, ctxSize, className, showArrows }: PaginatorParams) {
	if (page < 1) {
		page = 1;
	}
	if (ctxSize === undefined) {
		ctxSize = 2;
	}
	if (className === undefined) {
		className = "";
	}
	if (numpages < 1) {
		numpages = 1;
	}
	if (page > numpages) {
		page = numpages;
	}
	let elements: preact.JSX.Element[] = [];
	const old_sp = setPage;
	setPage = (pg) => {
		if (pg < 1) {
			pg = 1;
		}
		if (pg > numpages) {
			pg = numpages;
		}
		if (typeof old_sp == "function") {
			old_sp(pg);
		}
	};

	if (showArrows) {
		elements.push(
			<button class="paginator-item" onClick={() => setPage(1)} key={`jump_first`}>
				<i class="fas fa-angle-double-left"></i>
			</button>
		);
		elements.push(
			<button class="paginator-item" onClick={() => setPage(page - 1)} key={`jump_before`}>
				<i class="fas fa-angle-left"></i>
			</button>
		);
	}
	if (page > ctxSize + 1) {
		for (let i = 1; i <= 1 + ctxSize && page - i >= 1 + ctxSize; i++) {
			elements.push(
				<button class="paginator-item" onClick={() => setPage(i)} key={`inactive_${i}`}>
					{i}
				</button>
			);
		}
		if (page > 2 * (ctxSize + 1)) {
			elements.push(
				<span class="paginator-item" key="first_greater">
					...
				</span>
			);
		}
	}

	for (let i = page - ctxSize; i < page; i++) {
		if (i < 1) {
			continue;
		}
		elements.push(
			<button class="paginator-item" onClick={() => setPage(i)} key={`inactive_${i}`}>
				{i}
			</button>
		);
	}
	elements.push(
		<button class="paginator-item paginator-item-active" key={`active_${page}`}>
			{page}
		</button>
	);
	for (let i = page + 1; i <= page + ctxSize; i++) {
		if (i > numpages) {
			continue;
		}
		elements.push(
			<button class="paginator-item" onClick={() => setPage(i)} key={`inactive_${i}`}>
				{i}
			</button>
		);
	}

	if (numpages - page >= ctxSize + 1) {
		if (numpages - page > 2 * ctxSize + 1) {
			elements.push(
				<span class="paginator-item" key="last_greater">
					...
				</span>
			);
		}
		for (let i = numpages - ctxSize; i <= numpages; i++) {
			if (i - page <= ctxSize) {
				continue;
			}
			elements.push(
				<button class="paginator-item" onClick={() => setPage(i)} key={`inactive_${i}`}>
					{i}
				</button>
			);
		}
	}

	if (showArrows) {
		elements.push(
			<button class="paginator-item" onClick={() => setPage(page + 1)} key={`jump_after`}>
				<i class="fas fa-angle-right"></i>
			</button>
		);
		elements.push(
			<button class="paginator-item" onClick={() => setPage(numpages)} key={`jump_last`}>
				<i class="fas fa-angle-double-right"></i>
			</button>
		);
	}
	return <div class={"paginator " + className}>{elements}</div>;
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

async function getSubmissssions(user_id: number, problem_id: number, limit: number): Promise<getSubsResult> {
	const result = await getCall<getSubsResult>("/submissions/get", {
		limit,
		problem_id,
		user_id,
	});
	if (result.status !== "success") {
		throw new Error(result.data);
	}
	return result.data;
}

const SUB_VIEW_LIMIT = 5;

export function OlderSubmissions({ userid, problemid }: OlderSubsParams) {
	let [subs, setSubs] = useState<ResultSubmission[]>([]);
	let [loading, setLoading] = useState(true);
	let [numHidden, setNumHidden] = useState(0);

	async function load() {
		var data = await getSubmissions({ user_id: userid, problem_id: problemid, limit: SUB_VIEW_LIMIT, page: 1 });
		// const data = await getSubmissions(userid, problemid, SUB_VIEW_LIMIT);
		setSubs(data.subs);
		setNumHidden(Math.max(data.count - SUB_VIEW_LIMIT, 0));
		setLoading(false);
	}

	useEffect(() => {
		load().catch(console.error);
	}, [userid, problemid]);

	useEffect(() => {
		const poll = async (e) => load();
		document.addEventListener("kn-poll", poll);
		return () => document.removeEventListener("kn-poll", poll);
	}, []);

	return (
		<>
			<h2 class="mb-2">{getText("oldSubs")}</h2>
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
									<span>{`#${sub.sub.id}: ${dayjs(sub.sub.created_at).format("DD/MM/YYYY HH:mm")}`}</span>
									<span class="rounded-md px-2 py-1 bg-teal-700 text-white text-sm">
										{{
											finished: <>{sub.sub.score}</>,
											working: <i class="fas fa-cog animate-spin"></i>,
										}[sub.sub.status] || <i class="fas fa-clock"></i>}
									</span>
								</a>
							))}
						</div>
					) : (
						<p class="px-2">{getText("noSub")}</p>
					)}
					{numHidden > 0 && (
						<a class="px-2" href={`/submissions/?problem_id=${problemid}&user_id=${userid}`}>
							{getText(numHidden == 1 ? "seeOne" : numHidden < 20 ? "seeU20" : "seeMany", numHidden)}
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
