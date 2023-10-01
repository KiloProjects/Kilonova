import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import { Reducer, useEffect, useMemo, useReducer, useState } from "preact/hooks";
import { dayjs, getGradient } from "../util";
import getText from "../translation";
import { sprintf } from "sprintf-js";
import { fromBase64 } from "js-base64";
import { answerQuestion, getAllQuestions, getUserQuestions, getAnnouncements, updateAnnouncement, deleteAnnouncement } from "../api/contest";
import type { Question, Announcement } from "../api/contest";
import { UserBrief, getUser } from "../api/submissions";
import { apiToast, createToast } from "../toast";
import { BigSpinner, Paginator } from "./common";
import { getCall, postCall } from "../api/net";
import { buildScoreBreakdownModal } from "./maxscore_breakdown";
import { confirm } from "./modal";

export function contestToNetworkDate(timestamp: string): string {
	const djs = dayjs(timestamp, "YYYY-MM-DDTHH:mm", true);
	if (!djs.isValid()) {
		throw new Error("Invalid timestamp");
	}
	return djs.format("YYYY-MM-DDTHH:mm:ss.SSSZ");
}

function formatMinutes(mins: number): string {
	const minutes = mins % 60;
	mins = (mins - minutes) / 60;
	const hours = mins;
	if (hours >= 24) {
		return `${Math.floor(hours / 24)}:${Math.floor(hours % 24)}:${minutes}`;
	}
	return sprintf("%02d:%02d", hours, minutes);
}

function remainingTimeStr(time: dayjs.Dayjs): string {
	let diff = time.diff(dayjs(), "s");
	if (diff < 0) {
		return getText("time_expired");
	}
	const seconds = diff % 60;
	diff = (diff - seconds) / 60;
	const minutes = diff % 60;
	diff = (diff - minutes) / 60;
	const hours = diff;
	if (hours >= 48) {
		// >2 days
		return getText("days", Math.floor(diff / 24));
	}

	return sprintf("%02d:%02d:%02d", hours, minutes, seconds);
}

export function ContestRemainingTime({ target_time, reload }: { target_time: dayjs.Dayjs; reload: boolean }) {
	let [text, setText] = useState<string>(remainingTimeStr(target_time));

	function updateTime() {
		setText(remainingTimeStr(target_time));
		if (reload) {
			let diff = target_time.diff(dayjs(), "s");
			if (diff < 0) {
				console.log("Reloading webpage...");
				window.location.reload();
			}
		}
	}

	useEffect(() => {
		updateTime();
		const interval = setInterval(() => {
			updateTime();
		}, 500);
		return () => clearInterval(interval);
	}, []);

	return <span>{text}</span>;
}

export function ContestCountdown({ target_time, type }: { target_time: string; type: string }) {
	let timestamp = parseInt(target_time);
	if (isNaN(timestamp)) {
		console.error("unix timestamp is somehow NaN", target_time);
		return <>Invalid Timestamp</>;
	}
	const endTime = dayjs(timestamp);
	return (
		<>
			{endTime.diff(dayjs()) <= 0 ? (
				<span>{{ running: getText("contest_ended"), before_start: getText("contest_running") }[type]}</span>
			) : (
				<ContestRemainingTime target_time={endTime} reload={true} />
			)}
		</>
	);
}

type LeaderboardResponse = {
	problem_ordering: number[];
	problem_names: Record<number, string>;
	entries: {
		user: UserBrief;
		scores: Record<number, number>;
		total: number;

		num_solved: number;
		penalty: number;
		last_time: string | null;
		freeze_time: string | null;
		last_times: Record<number, number>;
		attempts: Record<number, number>; // TODO: check if will still be null once finished
	}[];

	freeze_time?: string;
	type: "classic" | "acm-icpc";
};

export function ContestLeaderboard({ contestID, editor }: { contestID: number; editor: boolean }) {
	let [loading, setLoading] = useState(true);
	let [problems, setProblems] = useState<{ id: number; name: string }[]>([]);
	let [leaderboard, setLeaderboard] = useState<LeaderboardResponse | null>(null);
	let [lastUpdated, setLastUpdated] = useState<string | null>(null);

	async function loadLeaderboard() {
		setLoading(true);
		const res = await getCall<LeaderboardResponse>(`/contest/${contestID}/leaderboard`, {});
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		setLastUpdated(
			res.data.entries.reduce((ant: string | null, prez): string | null => {
				if (prez.last_time == null) {
					return ant;
				}
				if (ant == null) {
					return prez.last_time;
				}
				return dayjs(ant).isBefore(prez.last_time) ? prez.last_time : ant;
			}, null)
		);
		console.log(res.data);
		setLeaderboard(res.data);
		setProblems(res.data.problem_ordering.map((val) => ({ id: val, name: res.data.problem_names[val] })));
		setLoading(false);
	}

	useEffect(() => {
		loadLeaderboard().catch(console.error);
	}, [contestID]);

	if (loading || leaderboard == null) {
		return (
			<>
				<button class="btn btn-blue mb-2" onClick={() => loadLeaderboard()}>
					{getText("reload")}
				</button>
				<BigSpinner />
			</>
		);
	}

	return (
		<>
			<button class="btn btn-blue mb-2" onClick={() => loadLeaderboard()}>
				{getText("reload")}
			</button>
			<div class="mb-2">
				<p>
					{getText("last_updated_at")}: {lastUpdated ? dayjs(lastUpdated).format("DD/MM/YYYY HH:mm") : "-"}
				</p>
				{leaderboard.freeze_time && dayjs().isAfter(leaderboard.freeze_time) && (
					<p>
						{getText("freeze_time")}: {dayjs(leaderboard.freeze_time).format("DD/MM/YYYY HH:mm")}
					</p>
				)}
			</div>
			<table class="kn-table table-fixed">
				<thead>
					<tr>
						<th class="kn-table-cell w-1/12" style={{ wordBreak: "break-all" }} scope="col">
							{getText("position")}
						</th>
						<th class="kn-table-cell w-1/5" style={{ wordBreak: "break-all" }} scope="col">
							{getText("name")}
						</th>
						{leaderboard.type == "acm-icpc" && (
							<>
								<th class="kn-table-cell w-1/12" style={{ wordBreak: "break-all" }} scope="col">
									{getText("icpc_num_solved")}
								</th>
								<th class="kn-table-cell w-1/12" style={{ wordBreak: "break-all" }} scope="col">
									{getText("penalty")}
								</th>
							</>
						)}
						{problems.map((pb) => (
							<th class="kn-table-cell" style={{ wordBreak: "break-all" }} scope="col" key={pb.id}>
								<a href={`/contests/${contestID}/problems/${pb.id}`}>{pb.name}</a>
							</th>
						))}
						{leaderboard.type == "classic" && (
							<th class="kn-table-cell w-1/6" style={{ wordBreak: "break-all" }} scope="col">
								{getText("total")}
							</th>
						)}
					</tr>
				</thead>
				<tbody>
					{leaderboard.entries.map((entry, idx) => (
						<tr class="kn-table-row" key={entry.user.id}>
							<td class="kn-table-cell">{idx + 1}.</td>
							<td class="kn-table-cell">
								<a href={`/profile/${entry.user.name}`}>{entry.user.name}</a>
							</td>
							{leaderboard?.type == "acm-icpc" && (
								<>
									<td class="kn-table-cell">{entry.num_solved}</td>
									<td class="kn-table-cell">{entry.penalty}</td>
								</>
							)}
							{problems.map((pb) =>
								leaderboard?.type == "classic" ? (
									<td
										class={"kn-table-cell" + (editor ? " cursor-pointer" : "")}
										scope="col"
										key={entry.user.name + pb.id + (editor ? "-e" : "-ne")}
										onClick={() => editor && buildScoreBreakdownModal(pb.id, contestID, entry.user.id)}
									>
										{pb.id in entry.scores && entry.scores[pb.id] >= 0 ? entry.scores[pb.id] : "-"}
									</td>
								) : entry.scores[pb.id] >= 0 ? (
									<td
										class="kn-table-cell"
										style={{
											color: entry.scores[pb.id] >= 100 ? "black" : undefined,
											backgroundColor: getGradient(entry.scores[pb.id] >= 100 ? 1 : 0, 1),
										}}
									>
										{entry.scores[pb.id] >= 0 ? (
											<>
												<span class="block font-bold text-lg">
													{entry.scores[pb.id] >= 100 ? "+" : "-"} {entry.attempts[pb.id] > 0 && entry.attempts[pb.id]}
												</span>
												{entry.scores[pb.id] >= 100 && <span class="block">{formatMinutes(entry.last_times[pb.id] ?? 0)}</span>}
											</>
										) : (
											"-"
										)}
									</td>
								) : (
									<td class="kn-table-cell">-</td>
								)
							)}
							{leaderboard?.type == "classic" && <td class="kn-table-cell">{entry.total}</td>}
						</tr>
					))}
					{leaderboard.entries.length == 0 && (
						<tr class="kn-table-row">
							<td class="kn-table-cell" colSpan={99}>
								<h1>{getText("no_users")}</h1>
							</td>
						</tr>
					)}
				</tbody>
			</table>
		</>
	);
}

function formatJSONTime(t: string, format_key: string): string {
	return dayjs(t).format(getText(format_key));
}

export function AnnouncementView({ ann, canEditAnnouncement }: { ann: Announcement; canEditAnnouncement: boolean }) {
	let [text, setText] = useState(ann.text);
	let [expandAnnouncement, setExpandAnnouncement] = useState<boolean>(false);

	async function editAnnouncement() {
		await updateAnnouncement(ann, text);
		setExpandAnnouncement(false);
	}

	if (expandAnnouncement) {
		return (
			<div class="segment-panel">
				<a href="#" onClick={(e) => (e.preventDefault(), setExpandAnnouncement(!expandAnnouncement))}>
					[{getText("button.cancel")}]
				</a>
				<label class="block my-2">
					<span class="form-label">{getText("text")}:</span>
					<textarea class="block form-textarea" value={text} onInput={(e) => setText(e.currentTarget.value)} />
				</label>
				<button class="btn btn-blue" onClick={editAnnouncement}>
					{getText("button.update")}
				</button>
			</div>
		);
	}

	return (
		<div class="segment-panel">
			<pre class="mt-2 mb-1">{text}</pre>
			<p class="text-sm">{formatJSONTime(ann.created_at, "contest_timestamp_posted_format")}</p>
			{canEditAnnouncement && (
				<>
					<div class="mt-2"></div>
					<button class="btn btn-blue mr-2" onClick={() => setExpandAnnouncement(!expandAnnouncement)}>
						{getText("button.update")}
					</button>
					<button class="btn btn-red" onClick={() => deleteAnnouncement(ann)}>
						{getText("button.delete")}
					</button>
				</>
			)}
		</div>
	);
}

export function QuestionView({ q, canEditAnswer, userLoadable }: { q: Question; canEditAnswer: boolean; userLoadable: boolean }) {
	let [response, setResponse] = useState<string>(q.response ?? "");
	let [expandAnswer, setExpandAnswer] = useState<boolean>(false);
	let [user, setUser] = useState<UserBrief | null>(null);

	async function doAnswer() {
		await answerQuestion(q, response);
		setExpandAnswer(false);
	}

	useEffect(() => {
		if (userLoadable) {
			getUser(q.author_id)
				.then((d) => setUser(d))
				.catch(console.error);
		}
	}, [q, userLoadable]);

	let responseComponent = <></>;
	if (q.response != null && !canEditAnswer) {
		// View answer
		responseComponent = (
			<>
				<h3>{getText("question_response")}</h3>
				<pre class="mt-2 mb-1">{q.response}</pre>
				<p class="text-sm">{formatJSONTime(q.responded_at!, "contest_timestamp_responded_format")}</p>
			</>
		);
	} else if (q.response == null && canEditAnswer) {
		// Send answer
		if (expandAnswer) {
			responseComponent = (
				<>
					<h3>
						{getText("respond_to_answer")}{" "}
						<a href="#" onClick={(e) => (e.preventDefault(), setExpandAnswer(!expandAnswer))}>
							[{getText("hide")}]
						</a>
					</h3>
					<label class="block my-2">
						<textarea class="form-textarea" value={response} onInput={(e) => setResponse(e.currentTarget.value)} />
					</label>
					<button class="btn btn-blue" onClick={doAnswer}>
						{getText("button.answer")}
					</button>
				</>
			);
		} else {
			responseComponent = (
				<button class="btn btn-blue mt-2" onClick={() => setExpandAnswer(!expandAnswer)}>
					{getText("button.respond")}
				</button>
			);
		}
	} else if (q.response != null && canEditAnswer) {
		// Edit answer
		responseComponent = (
			<>
				{!expandAnswer ? (
					<>
						<h3>{getText("question_response")}</h3>
						<pre class="mt-2 mb-1">{q.response}</pre>
						<p class="text-sm">{formatJSONTime(q.responded_at!, "contest_timestamp_responded_format")}</p>
						<button class="btn btn-blue mt-2" onClick={() => setExpandAnswer(!expandAnswer)}>
							{getText("edit_answer")}
						</button>
					</>
				) : (
					<>
						<h3>
							{getText("question_response")}{" "}
							<a href="#" onClick={(e) => (e.preventDefault(), setExpandAnswer(!expandAnswer))}>
								[{getText("button.cancel")}]
							</a>
						</h3>
						<label class="block my-2">
							<textarea class="form-textarea" value={response} onInput={(e) => setResponse(e.currentTarget.value)} />
						</label>
						<button class="btn btn-blue" onClick={doAnswer}>
							{getText("button.update")}
						</button>
					</>
				)}
			</>
		);
	} else {
		// Not answered yet and cannot do anything about that
		responseComponent = <>{getText("not_answered")}</>;
	}

	return (
		<div class="segment-panel">
			<pre class="mt-2 mb-1">{q.text}</pre>
			<p class="text-sm">{formatJSONTime(q.asked_at, "contest_timestamp_asked_format")}</p>
			{userLoadable && (
				<p>
					{getText("author")}: {user == null ? getText("loading") : <a href={`/profile/${user.name}`}>{user.name}</a>}
				</p>
			)}
			{responseComponent}
		</div>
	);
}

export function QuestionManager({ initialQuestions, contestID }: { initialQuestions: Question[]; contestID: number }) {
	let [questions, setQuestions] = useState(initialQuestions);

	const answeredQuestions = useMemo(
		() =>
			questions.filter((q): boolean => {
				return typeof q.response === "string";
			}),
		[questions]
	);
	const unansweredQuestions = useMemo(
		() =>
			questions.filter((q): boolean => {
				return q.response == null || typeof q.response === "undefined";
			}),
		[questions]
	);

	async function onQuestionReload() {
		const qs = await getAllQuestions(contestID);
		setQuestions(qs);
	}

	useEffect(() => {
		document.addEventListener("kn-contest-question-reload", onQuestionReload);
		return () => document.removeEventListener("kn-contest-question-reload", onQuestionReload);
	}, []);

	return (
		<div>
			{questions.length == 0 && <p>{getText("noQuestions")}</p>}
			{unansweredQuestions.length > 0 && <h3>{getText("unanswered_questions")}:</h3>}
			{unansweredQuestions.map((q) => (
				<QuestionView q={q} canEditAnswer={true} userLoadable={true} key={q.id} />
			))}
			{answeredQuestions.length > 0 && (
				<details>
					<summary>{getText("answered_questions")}</summary>
					{answeredQuestions.map((q) => (
						<QuestionView q={q} canEditAnswer={true} userLoadable={true} key={q.id} />
					))}
				</details>
			)}
		</div>
	);
}

export function QuestionList({ initialQuestions, contestID }: { initialQuestions: Question[]; contestID: number }) {
	let [questions, setQuestions] = useState(initialQuestions);

	const answeredQuestions = useMemo(
		() =>
			questions.filter((q): boolean => {
				return typeof q.response === "string";
			}),
		[questions]
	);
	const unansweredQuestions = useMemo(
		() =>
			questions.filter((q): boolean => {
				return q.response == null || typeof q.response === "undefined";
			}),
		[questions]
	);

	async function onQuestionReload() {
		const qs = await getUserQuestions(contestID);
		setQuestions(qs);
	}

	useEffect(() => {
		document.addEventListener("kn-contest-question-reload", onQuestionReload);
		return () => document.removeEventListener("kn-contest-question-reload", onQuestionReload);
	}, []);

	return (
		<>
			{questions.length == 0 && (
				<div class="segment-panel">
					<h2>{getText("questions")}</h2>
					<p>{getText("noQuestions")}</p>
				</div>
			)}
			{unansweredQuestions.length > 0 && (
				<div class="segment-panel">
					<h2>{getText("unanswered_questions")}</h2>
					{unansweredQuestions.map((q) => (
						<QuestionView q={q} canEditAnswer={false} userLoadable={false} key={q.id} />
					))}
				</div>
			)}
			{answeredQuestions.length > 0 && (
				<div class="segment-panel">
					<h2>{getText("answered_questions")}</h2>
					{answeredQuestions.map((q) => (
						<QuestionView q={q} canEditAnswer={false} userLoadable={false} key={q.id} />
					))}
				</div>
			)}
		</>
	);
}

function AnnouncementList({ initialAnnouncements, contestID, canEdit }: { initialAnnouncements: Announcement[]; contestID: number; canEdit: boolean }) {
	let [announcements, setAnnouncements] = useState(initialAnnouncements);

	async function onAnnouncementReload() {
		const anns = await getAnnouncements(contestID);
		setAnnouncements(anns);
	}

	useEffect(() => {
		document.addEventListener("kn-contest-announcement-reload", onAnnouncementReload);
		return () => document.removeEventListener("kn-contest-announcement-reload", onAnnouncementReload);
	}, []);

	return (
		<>
			<h2>{getText("announcements")}</h2>
			{announcements.length == 0 && <p>{getText("noAnnouncements")}</p>}
			{announcements.map((ann) => (
				<AnnouncementView ann={ann} canEditAnnouncement={canEdit} key={ann.id} />
			))}
		</>
	);
}

function createUpdateToast(contestID: number, title: string) {
	createToast({
		status: "info",
		title: title,
		description: `<a href="/contests/${contestID}/communication">${getText("go_to_communication")}</a>`,
	});
}

function genReducer(contestID: number, toast_text: string, setSthNew: (_: boolean) => void): Reducer<number, number> {
	return (val, newVal) => {
		if (newVal > val && val != -1) {
			createUpdateToast(contestID, getText(toast_text));
			setSthNew(true);
		}
		return newVal;
	};
}

function CommunicationAnnouncer({ contestID, contestEditor }: { contestID: number; contestEditor: boolean }) {
	let [sthNew, setSthNew] = useState<boolean>(false);
	let [numEditorQuestions, dispatchNumEditorQs] = useReducer(genReducer(contestID, "new_question", setSthNew), -1);
	let [numAnnouncements, dispatchNumAnns] = useReducer(genReducer(contestID, "new_announcement", setSthNew), -1);
	let [numAnswers, dispatchNumAnswers] = useReducer(genReducer(contestID, "new_response", setSthNew), -1);

	async function onQuestionReload() {
		const userQs = (await getUserQuestions(contestID)).filter((val) => typeof val.response === "string");
		dispatchNumAnswers(userQs.length);

		if (contestEditor) {
			const allQs = await getAllQuestions(contestID);
			dispatchNumEditorQs(allQs.length);
		}
	}

	async function onAnnouncementReload() {
		const anns = await getAnnouncements(contestID);
		dispatchNumAnns(anns.length);
	}

	useEffect(() => {
		onQuestionReload().catch(console.error);
		onAnnouncementReload().catch(console.error);
		document.addEventListener("kn-contest-question-reload", onQuestionReload);
		document.addEventListener("kn-contest-announcement-reload", onAnnouncementReload);
		return () => {
			document.removeEventListener("kn-contest-question-reload", onQuestionReload);
			document.removeEventListener("kn-contest-announcement-reload", onAnnouncementReload);
		};
	}, []);

	if (sthNew) {
		return <div class="badge-lite text-sm">{getText("new")}</div>;
	}

	return <></>;
}

type ContestRegistration = {
	created_at: string;
	contest_id: number;
	user_id: number;
	individual_start?: string;
	individual_end?: string;
};

type ContestRegRez = {
	user: UserBrief;
	registration: ContestRegistration;
};

function ContestRegistrations(params: { contestid: string; usacomode: string }) {
	const contestID = parseInt(params.contestid);
	if (isNaN(contestID)) {
		throw new Error("Invalid contest ID");
	}
	const usacoMode = params.usacomode == "true";

	let [users, setUsers] = useState<ContestRegRez[]>([]);
	let [page, setPage] = useState<number>(1);
	let [numPages, setNumPages] = useState<number>(1);
	let [cnt, setCnt] = useState<number>(-1);
	let [name, setName] = useState<string>("");

	async function poll() {
		let res = await getCall(`/contest/${contestID}/registrations`, { offset: 50 * (page - 1), limit: 50, name_fuzzy: name.length > 0 ? name : undefined });
		if (res.status !== "success") {
			apiToast(res);
			throw new Error("Couldn't fetch users");
		}
		setCnt(res.data.total_count);
		setUsers(res.data.registrations);
		setNumPages(Math.floor(res.data.total_count / 50) + (res.data.total_count % 50 != 0 ? 1 : 0));
	}

	function updateName(newName: string) {
		setPage(1);
		setName(newName);
	}

	useEffect(() => {
		poll().catch(console.error);
	}, [page, name]);

	return (
		<div class="my-4">
			<label class="block my-2">
				<span class="form-label">{getText("nameFilter")}: </span>
				<input
					class="form-input"
					type="text"
					onInput={(e) => {
						updateName(e.currentTarget.value);
					}}
					value={name}
				/>
			</label>
			{cnt >= 0 && <span class="block text-lg my-2"> {getText("num_registrations", cnt)}</span>}
			<button
				class={"btn btn-blue block"}
				onClick={() => {
					setUsers([]);
					setPage(1);
					setCnt(-1);
					poll();
				}}
			>
				{getText("reload")}
			</button>
			<Paginator numpages={numPages} page={page} setPage={setPage} showArrows={true} />
			{users.length === 0 ? (
				<div class="text-4xl mx-auto my-auto w-full mt-10 mb-10 text-center">{getText("noUserFound")}</div>
			) : (
				<table class="kn-table">
					<thead>
						<tr>
							<th class="kn-table-cell" scope="col">
								{getText("username")}
							</th>
							{usacoMode && (
								<th class="kn-table-cell" scope="col">
									{getText("started_at")}
								</th>
							)}
							<th class="kn-table-cell" scope="col">
								{getText("action")}
							</th>
						</tr>
					</thead>
					<tbody>
						{users.map((user) => (
							<tr class="kn-table-row" key={user.user.id}>
								<td class="kn-table-cell">
									<a href={`/profile/${user.user.name}`}>
										<img
											width={32}
											height={32}
											class="inline-block mr-2 rounded align-middle"
											src={`/api/user/getGravatar?name=${user.user.name}&s=32`}
										/>{" "}
										<span class="align-middle">{user.user.name}</span>
									</a>
								</td>
								{usacoMode && (
									<td class="kn-table-cell">
										{user.registration.individual_start === null ? (
											<>{getText("not_started")}</>
										) : (
											<ContestRemainingTime target_time={dayjs(user.registration.individual_end)} reload={false} />
										)}
									</td>
								)}
								<td class="kn-table-cell">
									<button
										class="btn btn-red"
										onClick={async () => {
											if (!(await confirm(getText("confirmUserKick")))) {
												return;
											}
											let res = await postCall(`/contest/${contestID}/kickUser`, { name: user.user.name });
											apiToast(res);
											if (res.status === "success") {
												await poll();
											}
										}}
									>
										{getText("kick_user")}
									</button>
								</td>
							</tr>
						))}
					</tbody>
				</table>
			)}
		</div>
	);
}

function AnnouncementListDOM({ encoded, contestid, canedit }: { encoded: string; contestid: string; canedit: string }) {
	const q: Announcement[] = JSON.parse(fromBase64(encoded));
	const contestID = parseInt(contestid);
	if (isNaN(contestID)) {
		throw new Error("Invalid contest ID");
	}
	return <AnnouncementList initialAnnouncements={q} canEdit={canedit == "true"} contestID={contestID} />;
}

function QuestionListDOM({ encoded, contestid }: { encoded: string; contestid: string }) {
	const q: Question[] = JSON.parse(fromBase64(encoded));
	const contestID = parseInt(contestid);
	if (isNaN(contestID)) {
		throw new Error("Invalid contest ID");
	}
	return <QuestionList initialQuestions={q} contestID={contestID} />;
}

function QuestionManagerDOM({ encoded, contestid }: { encoded: string; contestid: string }) {
	const q: Question[] = JSON.parse(fromBase64(encoded));
	const contestID = parseInt(contestid);
	if (isNaN(contestID)) {
		throw new Error("Invalid contest ID");
	}
	return <QuestionManager initialQuestions={q} contestID={contestID} />;
}

function CommunicationAnnouncerDOM({ contestid, contesteditor }: { contestid: string; contesteditor: string }) {
	const contestID = parseInt(contestid);
	if (isNaN(contestID)) {
		throw new Error("Invalid contest ID");
	}
	return <CommunicationAnnouncer contestID={contestID} contestEditor={contesteditor == "true"} />;
}

function ContestLeaderboardDOM({ contestid, editor }: { contestid: string; editor: string }) {
	const contestID = parseInt(contestid);
	if (isNaN(contestID)) {
		throw new Error("Invalid contest ID");
	}
	return <ContestLeaderboard contestID={contestID} editor={editor === "true"} />;
}

register(QuestionManagerDOM, "kn-question-mgr", ["encoded", "contestid"]);
register(QuestionListDOM, "kn-questions", ["encoded", "contestid"]);
register(AnnouncementListDOM, "kn-announcements", ["encoded", "contestid", "canedit"]);
register(ContestCountdown, "kn-contest-countdown", ["target_time", "type"]);
register(CommunicationAnnouncerDOM, "kn-comm-announcer", ["contestid", "contesteditor"]);
register(ContestLeaderboardDOM, "kn-leaderboard", ["contestid", "editor"]);
register(ContestRegistrations, "kn-contest-registrations", ["contestid", "usacomode"]);
