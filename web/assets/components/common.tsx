import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import getText from "../translation";
import { dayjs, fromBase64 } from "../util";
import { useEffect, useState } from "preact/hooks";
import { Submissions, defaultClient } from "../api/client";

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

export function OlderSubmissions({
	userID,
	problemID,
	contestID,
	limit = 5,
	initialData,
}: {
	userID: number;
	problemID: number;
	contestID?: number;
	limit?: number;
	initialData?: Submissions;
}) {
	let [subs, setSubs] = useState<Submissions | undefined>(initialData);
	let [numHidden, setNumHidden] = useState(initialData?.count ? initialData?.count - limit : 0);

	async function load() {
		var data = await defaultClient.getSubmissions({ user_id: userID, problem_id: problemID, contest_id: contestID, limit, page: 1 });
		setSubs(data);
		setNumHidden(Math.max(data.count - limit, 0));
	}

	useEffect(() => {
		// TODO: Test
		if (typeof initialData === "undefined") {
			load().catch(console.error);
		}
	}, [userID, problemID, contestID, limit]);

	useEffect(() => {
		const poll = async (e) => load();
		document.addEventListener("kn-poll", poll);
		return () => document.removeEventListener("kn-poll", poll);
	}, []);

	return (
		<details open>
			<summary>
				<h2 class="inline-block mb-2">{getText("oldSubs")}</h2>
			</summary>
			{typeof subs == "undefined" ? (
				<InlineSpinner />
			) : (
				<>
					{subs?.submissions.length > 0 ? (
						<div>
							{subs.submissions.map((sub) => (
								<a
									href={`/submissions/${sub.id}`}
									class="black-anchor flex justify-between items-center rounded py-1 px-2 hoverable"
									key={sub.id}
								>
									<span>{`#${sub.id}: ${dayjs(sub.created_at).format("DD/MM/YYYY HH:mm")}`}</span>
									<span class="badge-lite text-sm">
										{{
											finished: <>{sub.score.toFixed(sub.score_precision)}</>,
											working: <i class="fas fa-cog animate-spin"></i>,
										}[sub.status] || <i class="fas fa-clock"></i>}
									</span>
								</a>
							))}
						</div>
					) : (
						<p class="px-2">{getText("noSub")}</p>
					)}
					{numHidden > 0 && (
						<a class="px-2" href={`${contestID ? `/contests/${contestID}` : ""}/problems/${problemID}/submissions/?user_id=${userID}`}>
							{getText(numHidden == 1 ? "seeOne" : numHidden < 20 ? "seeU20" : "seeMany", numHidden)}
						</a>
					)}
				</>
			)}
		</details>
	);
}

function OlderSubmissionsDOM({ userid, problemid, contestid, enc }: { userid: string; problemid: string; contestid: string; enc: string }) {
	const userID = parseInt(userid);
	if (isNaN(userID)) {
		throw new Error("Invalid user ID");
	}
	const problemID = parseInt(problemid);
	if (isNaN(problemID)) {
		throw new Error("Invalid problem ID");
	}
	let contestID: number | undefined = undefined;
	if (contestid !== "" && typeof contestid !== "undefined") {
		contestID = parseInt(contestid);
		if (isNaN(contestID)) {
			console.warn("Invalid Contest ID");
			contestID = undefined;
		}
	}

	let initialData: Submissions | undefined = JSON.parse(fromBase64(enc));
	return <OlderSubmissions userID={userID} problemID={problemID} contestID={contestID} initialData={initialData}></OlderSubmissions>;
}

register(OlderSubmissionsDOM, "older-subs", ["userid", "problemid", "contestid", "enc"]);

function ProgressChecker({ id }: { id: number }) {
	var [computable, setComputable] = useState<boolean>(false);
	var [loaded, setLoaded] = useState<number>(0);
	var [total, setTotal] = useState<number>(0);
	var [processing, setProcessing] = useState<boolean>(false);

	useEffect(() => {
		const upd = (e: CustomEvent<ProgressEventData>) => {
			if (e.detail.id == id) {
				setLoaded(e.detail.cntLoaded);
				setTotal(e.detail.cntTotal);
				setComputable(e.detail.computable);
				setProcessing(e.detail.processing);
			}
		};
		document.addEventListener("kn-upload-update", upd);
		return () => {
			document.removeEventListener("kn-upload-update", upd);
		};
	}, [id]);

	if (processing) {
		return <span>{getText("upload_processing")}</span>;
	}

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
