import cookie from "js-cookie";
import qs from "query-string";

export type ResultSubmission = {
	sub: Submission;
	author: UserBrief;
	problem?: Problem;
};

export type SubmissionQuery = {
	user_id?: number;
	problem_id?: number;
	contest_id?: number;
	score?: number;
	status?: string;
	lang?: string;

	page: number;

	compile_error?: boolean;
	ordering?: string;
	ascending?: boolean;

	limit?: number;
};

type RequestParams = {
	url: string;
	method: "GET" | "POST" | "PUT" | "DELETE";
	contentType?: string;
	queryParams?: any;
	body?: BodyInit;
	headers?: HeadersInit;
};

export type Submissions = {
	submissions: Submission[];
	count: number;
	users: Record<string, UserBrief>;
	problems: Record<string, Problem>;
};

export type Response<T> = { status: "error"; data: string } | { status: "success"; data: T };

export class KNClient {
	session: string;

	constructor(session: string, base: string) {
		this.session = session;
	}

	setSession(newSession: string) {
		this.session = newSession;
	}

	async request<T = any>(params: RequestParams): Promise<Response<T>> {
		if (params.url.startsWith("/")) {
			params.url = params.url.substring(1);
		}
		try {
			let resp = await fetch(`/api/${params.url}?${qs.stringify(params.queryParams)}`, {
				method: params.method,
				headers: {
					Accept: "application/json",
					Authorization: this.session,
					"Content-Type": params.contentType || "text/plain; charset=UTF-8",
					...params.headers,
				},
				body: params.body || null,
			});
			return (await resp.json()) as Response<T>;
		} catch (e: any) {
			return { status: "error", data: e.toString() };
		}
	}

	async getSubmissions(q: SubmissionQuery) {
		let res = await this.request<Submissions>({
			url: "/submissions/get",
			method: "GET",
			queryParams: {
				ordering: typeof q.ordering !== "undefined" ? q.ordering : "id",
				ascending: (typeof q.ordering !== "undefined" && q.ascending) || false,
				user_id: typeof q.user_id !== "undefined" && q.user_id > 0 ? q.user_id : undefined,
				problem_id: typeof q.problem_id !== "undefined" && q.problem_id > 0 ? q.problem_id : undefined,
				contest_id: typeof q.contest_id !== "undefined" && q.contest_id > 0 ? q.contest_id : undefined,
				status: q.status !== "" ? q.status : undefined,
				score: typeof q.score !== "undefined" && q.score >= 0 ? q.score : undefined,
				lang: typeof q.lang !== "undefined" && q.lang !== "" ? q.lang : undefined,
				compile_error: q.compile_error,
				offset: (q.page - 1) * 50,
				limit: typeof q.limit !== "undefined" && q.limit > 0 ? q.limit : 50,
			},
		});
		if (res.status === "error") {
			throw new Error(res.data);
		}
		return res.data;
	}

	async getSubmission(id: number) {
		let res = await this.request<FullSubmission>({
			url: "/submissions/getByID",
			method: "GET",
			queryParams: { id: id },
		});
		if (res.status === "error") {
			throw new Error(res.data);
		}
		return res.data;
	}

	async getUser(id: number) {
		let res = await this.request<UserBrief>({
			url: "/user/get",
			method: "GET",
			queryParams: { id: id },
		});
		if (res.status === "error") {
			throw new Error(res.data);
		}
		return res.data;
	}
}

export const defaultClient = new KNClient(cookie.get("kn-sessionid") || "guest", "/api");

export async function getCall<T = any>(call: string, params: any): Promise<Response<T>> {
	return defaultClient.request<T>({
		url: call,
		method: "GET",
		queryParams: params,
	});
}

export async function postCall<T = any>(call: string, params: any): Promise<Response<T>> {
	return defaultClient.request<T>({
		url: call,
		method: "POST",
		contentType: "application/x-www-form-urlencoded",
		body: qs.stringify(params),
	});
}

export async function bodyCall<T = any>(call: string, body: any): Promise<Response<T>> {
	return defaultClient.request<T>({
		url: call,
		method: "POST",
		contentType: "application/json",
		body: JSON.stringify(body),
	});
}

export async function multipartCall<T = any>(call: string, formdata: FormData): Promise<Response<T>> {
	return defaultClient.request<T>({
		url: call,
		method: "POST",
		contentType: "multipart/form-data",
		body: formdata,
	});
}
