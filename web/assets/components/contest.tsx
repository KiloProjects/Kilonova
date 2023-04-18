import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import { Reducer, useEffect, useMemo, useReducer, useState } from "preact/hooks";
import { dayjs } from "../util";
import getText from "../translation";
import { sprintf } from "sprintf-js";
import { fromBase64 } from "js-base64";
import { answerQuestion, getAllQuestions, getUserQuestions, getAnnouncements, updateAnnouncement, deleteAnnouncement } from "../contest";
import type { Question, Announcement } from "../contest";
import { UserBrief, getUser } from "../api/submissions";
import { apiToast, createToast } from "../toast";
import { isEqual } from "underscore";
import { BigSpinner, Paginator } from "./common";
import { getCall } from "../net";

export const RFC1123Z = "ddd, DD MMM YYYY HH:mm:ss ZZ";

export function contestToNetworkDate(timestamp: string): string {
	const djs = dayjs(timestamp, "YYYY-MM-DD HH:mm ZZ", true);
	if (!djs.isValid()) {
		throw new Error("Invalid timestamp");
	}
	return djs.format(RFC1123Z);
}

export function ContestRemainingTime({ target_time, reload }: { target_time: dayjs.Dayjs; reload: boolean }) {
	let [text, setText] = useState<string>("");

	function updateTime() {
		let diff = target_time.diff(dayjs(), "s");
		if (diff < 0) {
			if (reload) {
				console.log("Reloading webpage...");
				window.location.reload();
			}
			setText(getText("time_expired"));
			return;
		}
		const seconds = diff % 60;
		diff = (diff - seconds) / 60;
		const minutes = diff % 60;
		diff = (diff - minutes) / 60;
		const hours = diff;

		if (hours >= 48) {
			// >2 days
			setText(getText("days", Math.floor(diff / 24)));
			return;
		}

		setText(sprintf("%02d:%02d:%02d", hours, minutes, seconds));
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
	}[];
};

export function ContestLeaderboard({ contestID }: { contestID: number }) {
	let [loading, setLoading] = useState(true);
	let [problems, setProblems] = useState<{ id: number; name: string }[]>([]);
	let [leaderboard, setLeaderboard] = useState<LeaderboardResponse | null>(null);

	async function loadLeaderboard() {
		setLoading(true);
		const res = await getCall<LeaderboardResponse>(`/contest/${contestID}/leaderboard`, {});
		if (res.status === "error") {
			apiToast(res);
			return;
		}
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
			<table class="kn-table">
				<thead>
					<tr>
						<th class="kn-table-cell w-1/12" scope="col">
							{getText("position")}
						</th>
						<th class="kn-table-cell" scope="col">
							{getText("name")}
						</th>
						{problems.map((pb) => (
							<th class="kn-table-cell" scope="col" key={pb.id}>
								<a href={`/contests/${contestID}/problems/${pb.id}`}>{pb.name}</a>
							</th>
						))}
						<th class="kn-table-cell" scope="col">
							{getText("total")}
						</th>
					</tr>
				</thead>
				<tbody>
					{leaderboard.entries.map((entry, idx) => (
						<tr class="kn-table-row" key={entry.user.id}>
							<td class="kn-table-cell">{idx + 1}.</td>
							<td class="kn-table-cell">
								<a href={`/profile/${entry.user.name}`}>{entry.user.name}</a>
							</td>
							{problems.map((pb) => (
								<td class="kn-table-cell" scope="col" key={entry.user.name + pb.id}>
									{pb.id in entry.scores && entry.scores[pb.id] >= 0 ? entry.scores[pb.id] : "-"}
								</td>
							))}
							<td class="kn-table-cell">{entry.total}</td>
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

function RegistrationTable({ users, usacoMode }: { users: ContestRegRez[]; usacoMode: boolean }) {
	if (users.length == 0) {
		return (
			<div class="list-group">
				<div class="list-group-head font-bold">User</div>
				<div class="list-group-item">{getText("no_users")}</div>
			</div>
		);
	}
	return (
		<div class="list-group">
			<div class="list-group-head font-bold">User</div>
			{users.map((user) => (
				<a href={`/profile/${user.user.name}`} class="list-group-item inline-flex align-middle items-center" key={user.user.id}>
					<img class="flex-none mr-2 rounded" src={`/api/user/getGravatar?name=${user.user.name}&s=32`} /> {user.user.name} (#{user.user.id}){" "}
					{usacoMode && (
						<span class="badge badge-lite ml-2">
							{user.registration.individual_start === null ? (
								<>{getText("not_started")}</>
							) : (
								<ContestRemainingTime target_time={dayjs(user.registration.individual_end)} reload={false} />
							)}
						</span>
					)}
				</a>
			))}
		</div>
	);
}

function ContestRegistrations({ contestid, usacomode }: { contestid: string; usacomode: string }) {
	let [users, setUsers] = useState<ContestRegRez[]>([]);
	let [page, setPage] = useState<number>(1);
	let [numPages, setNumPages] = useState<number>(1);
	let [cnt, setCnt] = useState<number>(-1);

	async function poll() {
		let res = await getCall(`/contest/${contestid}/registrations`, { offset: 50 * (page - 1), limit: 50 });
		if (res.status !== "success") {
			apiToast(res);
			throw new Error("Couldn't fetch users");
		}
		setCnt(res.data.total_count);
		setUsers(res.data.registrations);
		setNumPages(Math.floor(res.data.total_count / 50) + (res.data.total_count % 50 != 0 ? 1 : 0));
	}

	useEffect(() => {
		poll().catch(console.error);
	}, [page]);

	return (
		<div class="my-4">
			<button
				class={"btn btn-blue block mb-2"}
				onClick={() => {
					setUsers([]);
					setPage(1);
					setCnt(-1);
					poll();
				}}
			>
				{getText("reload")}
			</button>
			{cnt >= 0 && getText("num_registrations", cnt)}
			<Paginator numpages={numPages} page={page} setPage={setPage} showArrows={true} />
			<RegistrationTable users={users} usacoMode={usacomode == "true"} />
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

function ContestLeaderboardDOM({ contestid }: { contestid: string }) {
	const contestID = parseInt(contestid);
	if (isNaN(contestID)) {
		throw new Error("Invalid contest ID");
	}
	return <ContestLeaderboard contestID={contestID} />;
}

register(QuestionManagerDOM, "kn-question-mgr", ["encoded", "contestid"]);
register(QuestionListDOM, "kn-questions", ["encoded", "contestid"]);
register(AnnouncementListDOM, "kn-announcements", ["encoded", "contestid", "canedit"]);
register(ContestCountdown, "kn-contest-countdown", ["target_time", "type"]);
register(CommunicationAnnouncerDOM, "kn-comm-announcer", ["contestid", "contesteditor"]);
register(ContestLeaderboardDOM, "kn-leaderboard", ["contestid"]);
register(ContestRegistrations, "kn-contest-registrations", ["contestid", "usacomode"]);
