import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import { StateUpdater, useEffect, useMemo, useState } from "preact/hooks";
import { dayjs } from "../util";
import getText from "../translation";
import { sprintf } from "sprintf-js";
import { fromBase64 } from "js-base64";
import { answerQuestion, getAllQuestions, getUserQuestions, getAnnouncements, updateAnnouncement, deleteAnnouncement } from "../contest";
import type { Question, Announcement } from "../contest";
import { UserBrief, getUser } from "../api/submissions";

export const RFC1123Z = "ddd, DD MMM YYYY HH:mm:ss ZZ";
export const JSONTimestamp = "YYYY-MM-DDTHH:mm:ss.SSSSSSZ";

export function contestToNetworkDate(timestamp: string): string {
	const djs = dayjs(timestamp, "YYYY-MM-DD HH:mm ZZ", true);
	if (!djs.isValid()) {
		throw new Error("Invalid timestamp");
	}
	return djs.format(RFC1123Z);
}

export function ContestRemainingTime({ target_time }: { target_time: dayjs.Dayjs }) {
	let [text, setText] = useState<string>("");

	function updateTime() {
		let diff = target_time.diff(dayjs(), "s");
		if (diff < 0) {
			console.log("Reloading webpage...");
			window.location.reload();
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
				<ContestRemainingTime target_time={endTime} />
			)}
		</>
	);
}

function formatJSONTime(t: string, format_key: string): string {
	return dayjs(t, JSONTimestamp).format(getText(format_key));
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
			<div class="segment-container">
				<a href="#" onClick={(e) => (e.preventDefault(), setExpandAnnouncement(!expandAnnouncement))}>
					[{getText("button.cancel")}]
				</a>
				<label class="block my-2">
					<span class="form-label">{getText("text")}:</span>
					<textarea class="block form-textarea" value={text} onChange={(e) => setText(e.currentTarget.value)} />
				</label>
				<button class="btn btn-blue" onClick={editAnnouncement}>
					{getText("button.update")}
				</button>
			</div>
		);
	}

	return (
		<div class="segment-container">
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
						<textarea class="form-textarea" value={response} onChange={(e) => setResponse(e.currentTarget.value)} />
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
							<textarea class="form-textarea" value={response} onChange={(e) => setResponse(e.currentTarget.value)} />
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
		<div class="segment-container">
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
		<div>
			{questions.length == 0 && <p>{getText("noQuestions")}</p>}
			{unansweredQuestions.map((q) => (
				<QuestionView q={q} canEditAnswer={false} userLoadable={false} key={q.id} />
			))}
			{unansweredQuestions.length > 0 && answeredQuestions.length > 0 && <div class="page-sidebar-divider mb-2" />}
			{answeredQuestions.map((q) => (
				<QuestionView q={q} canEditAnswer={false} userLoadable={false} key={q.id} />
			))}
		</div>
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

register(QuestionManagerDOM, "kn-question-mgr", ["encoded", "contestid"]);
register(QuestionListDOM, "kn-questions", ["encoded", "contestid"]);
register(AnnouncementListDOM, "kn-announcements", ["encoded", "contestid", "canedit"]);
register(ContestCountdown, "kn-contest-countdown", ["target_time", "type"]);
