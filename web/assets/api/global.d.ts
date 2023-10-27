export {};

declare global {
	// Base types

	type UserBrief = {
		id: number;
		name: string;
		admin: boolean;
		proposer: boolean;

		display_name: string;
	};

	type Problem = {
		id: number;
		created_at: string;
		name: string;
		test_name: string;
		default_points: number;
		visible: boolean;
		time_limit: number;
		memory_limit: number;
		source_credits: string;
		console_input: boolean;
		scoring_strategy: "sum_subtasks" | "max_submission" | "acm_icpc";
		score_precision: number;
	};

	type Submission = {
		id: number;
		created_at: string;
		user_id: number;
		problem_id: number;
		language: string;
		code?: string;
		code_size: number;
		compile_error: boolean;
		compile_message?: string;
		contest_id: number | null;
		max_time: number;
		max_memory: number;
		score: number;
		status: string;
		score_precision: number;

		submission_type: "classic" | "acm-icpc";
		icpc_verdict: string | null;
	};
	export type SubTest = {
		id: number;
		created_at: string;
		done: boolean;
		verdict: string;
		time: number;
		memory: number;
		percentage: number;
		test_id?: number;
		user_id: number;
		submission_id: number;

		visible_id: number;
		score: number;
	};

	export type SubmissionSubTask = {
		id: number;
		created_at: string;

		submission_id: number;
		user_id: number;
		subtask_id?: number;

		problem_id: number;
		visible_id: number;
		score: number;
		final_percentage?: number;

		subtests: number[];
	};

	// Derived types

	type FullSubmission = Submission & {
		author: UserBrief;
		problem: Problem;
		subtests: SubTest[];
		subtasks: SubmissionSubTask[];

		problem_editor: boolean;
		truly_visible: boolean;
	};

	// Contest types
	export type Question = {
		id: number;
		asked_at: string;
		responded_at?: string;
		text: string;
		response?: string;
		author_id: number;
		contest_id: number;
	};

	export type Announcement = {
		id: number;
		created_at: string;
		contest_id: number;
		text: string;
	};
}
