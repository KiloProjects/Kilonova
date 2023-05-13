import { h, Fragment, render } from "preact";
import { useEffect, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";
import { apiToast } from "../toast";
import { BigSpinner } from "./common";
import type { Problem, UserBrief, Submission } from "../api/submissions";
import { getCall } from "../api/net";
import { fromBase64 } from "js-base64";
import { KNModal } from "./maxscore_breakdown";
import { sizeFormatter } from "../util";

type SubList = {
	users: Record<number, UserBrief>;
	problems: Problem[];
	submissions: Submission[];
};

type ProblemStats = {
	num_solved: number;
	num_attempted: number;
	size_leaderboard: SubList;
	memory_leaderboard: SubList;
	time_leaderboard: SubList;
};

export function ProblemStatistics({ problem }: { problem: Problem }) {
	const [stats, setStats] = useState<ProblemStats | null>(null);

	async function load() {
		let res = await getCall<ProblemStats>(`/problem/${problem.id}/statistics`, {});
		if (res.status === "error") {
			apiToast(res);
			return;
		}

		setStats(res.data);
	}

	function canShowSizeLeaderboard(): boolean {
		if (stats == null) return false;
		if (stats.size_leaderboard.submissions.length === 0) return false;
		return stats.size_leaderboard.submissions[0].code_size > 0;
	}

	useEffect(() => {
		load().catch(console.error);
	}, [problem]);

	if (stats == null) {
		return <BigSpinner />;
	}

	console.log(stats);

	return (
		<>
			<h2>{getText("generalStats")}</h2>
			<p>{getText("numUsersSolved", stats.num_solved)}</p>
			<p>{getText("numUsersAttempted", stats.num_attempted)}</p>
			{stats.time_leaderboard.submissions.length > 0 && (
				<>
					<h2>{getText("timeLeaderboard")}</h2>
					<table class="kn-table kn-table-slim">
						<thead>
							<tr>
								<th class="kn-table-cell" scope="col">
									{getText("position")}
								</th>
								<th class="kn-table-cell" scope="col">
									{getText("name")}
								</th>
								<th class="kn-table-cell" scope="col">
									{getText("time")}
								</th>
							</tr>
						</thead>
						<tbody>
							{stats.time_leaderboard.submissions.map((sub, idx) => {
								const author: UserBrief | undefined = stats.time_leaderboard.users[sub.user_id];
								return (
									<tr class="kn-table-row" key={sub.id}>
										<td class="kn-table-cell">{idx + 1}.</td>
										<td class="kn-table-cell">
											<a href={`/profile/${author?.name}`}>{author?.name ?? "???"}</a>
										</td>
										<td class="kn-table-cell">
											<a href={`/submissions/${sub.id}`}>{sub.max_time == -1 ? "-" : Math.floor(sub.max_time * 1000) + "ms"}</a>
										</td>
									</tr>
								);
							})}
						</tbody>
					</table>
				</>
			)}
			{stats.memory_leaderboard.submissions.length > 0 && (
				<>
					<h2>{getText("memoryLeaderboard")}</h2>
					<table class="kn-table kn-table-slim">
						<thead>
							<tr>
								<th class="kn-table-cell" scope="col">
									{getText("position")}
								</th>
								<th class="kn-table-cell" scope="col">
									{getText("name")}
								</th>
								<th class="kn-table-cell" scope="col">
									{getText("memory")}
								</th>
							</tr>
						</thead>
						<tbody>
							{stats.memory_leaderboard.submissions.map((sub, idx) => {
								const author: UserBrief | undefined = stats.memory_leaderboard.users[sub.user_id];
								return (
									<tr class="kn-table-row" key={sub.id}>
										<td class="kn-table-cell">{idx + 1}.</td>
										<td class="kn-table-cell">
											<a href={`/profile/${author?.name}`}>{author?.name ?? "???"}</a>
										</td>
										<td class="kn-table-cell">
											<a href={`/submissions/${sub.id}`}>{sub.max_memory == -1 ? "-" : sizeFormatter(sub.max_memory * 1024)}</a>
										</td>
									</tr>
								);
							})}
						</tbody>
					</table>
				</>
			)}
			{canShowSizeLeaderboard() && (
				<>
					<h2>{getText("sizeLeaderboard")}</h2>
					<table class="kn-table kn-table-slim">
						<thead>
							<tr>
								<th class="kn-table-cell" scope="col">
									{getText("position")}
								</th>
								<th class="kn-table-cell" scope="col">
									{getText("name")}
								</th>
								<th class="kn-table-cell" scope="col">
									{getText("codeSize")}
								</th>
							</tr>
						</thead>
						<tbody>
							{stats.size_leaderboard.submissions.map((sub, idx) => {
								const author: UserBrief | undefined = stats.size_leaderboard.users[sub.user_id];
								return (
									<tr class="kn-table-row" key={sub.id}>
										<td class="kn-table-cell">{idx + 1}.</td>
										<td class="kn-table-cell">
											<a href={`/profile/${author?.name}`}>{author?.name ?? "???"}</a>
										</td>
										<td class="kn-table-cell">
											<a href={`/submissions/${sub.id}`}>{sub.code_size > 0 ? sizeFormatter(sub.code_size) : "-"}</a>
										</td>
									</tr>
								);
							})}
						</tbody>
					</table>
				</>
			)}
		</>
	);
}

function ProblemStatisticsDOM({ pbenc }: { pbenc: string }) {
	const pb: Problem = JSON.parse(pbenc);
	return (
		<KNModal open={true} title={<span dangerouslySetInnerHTML={{ __html: getText("problemStatsTitle", pb.name) }}></span>}>
			<ProblemStatistics problem={pb} />
		</KNModal>
	);
}

register(ProblemStatisticsDOM, "kn-problem-stats", ["pbenc"]);

export function buildProblemStatsModal(problem: Problem) {
	const val = document.getElementById("problem_stats_preact");
	if (val != null) {
		document.getElementById("modals")!.removeChild(val);
	}
	const newVal = document.createElement("kn-problem-stats");
	newVal.id = "problem_stats_preact";
	newVal.setAttribute("pbenc", JSON.stringify(problem));

	document.getElementById("modals")!.appendChild(newVal);
}

document.addEventListener("DOMContentLoaded", () => {
	Array.from(document.getElementsByClassName("problem_statistics")).forEach((val) => {
		val.addEventListener("click", (e) => {
			e.preventDefault();
			if (e.currentTarget == null) {
				return;
			}

			try {
				var problem = JSON.parse(fromBase64((e.currentTarget as HTMLElement).dataset.problem ?? ""));
				if (Object.keys(problem).length == 0) {
					apiToast({ status: "error", data: "No data provided for statistics modal" });
				}
			} catch (e) {
				console.error(e);
				apiToast({ status: "error", data: "Invalid problem base64 for statistics modal" });
				return;
			}
			buildProblemStatsModal(problem);
		});
	});
});
