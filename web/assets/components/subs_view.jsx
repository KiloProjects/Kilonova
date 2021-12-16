import {h, Fragment, Component} from 'preact';
import register from 'preact-custom-element';

import dayjs from 'dayjs';

import throttle from 'underscore/modules/throttle.js';
import {getText} from '../translation';
import {createToast, apiToast} from '../toast';
import {getGradient} from '../util';

import {BigSpinner} from './common';
import {getCall} from '../net';

class SubmissionsViewer extends Component {
	state = {
		submissions: [],
		subCount: -1,
		page: 1,
		filters: {user_id: null, problem_id: null, status: "", lang: "", visible: "", quality: "", compile_error: "", offset: 0, ordering: "id", ascending: false},
		loading: true,
		problem_filtered: 0,
		hiddenBar: false,
	}

	constructor() {
		super();

		this.polling = false;

		this.load = throttle(this.load, 200)
		
		// load initial state from url query
		this.fromQuery();
		
	}

	async componentDidMount() {
		// do initial loading
		await this.load()
	}

	updater(filter, inTop=false) {
		return ((e) => {
			console.log("Updating", JSON.stringify(filter), e.target.valueAsNumber)
			const val = e.target.valueAsNumber || e.target.value;
			this.setState(oldState => {
				if(inTop) {
					oldState[filter] = val;
				} else {
					oldState["filters"][filter] = val;
				}
				return oldState
			}, () => {
				if(filter == "page") {
					this.load()
				}
			})
		}).bind(this)
	}
	
	updateCheckbox(filter, inTop=false) {
		return ((e) => {
			console.log("Updating checkbox", JSON.stringify(filter), !e.target.checked)
			this.setState(oldState => {
				if(inTop) {
					const val = !oldState[filter];
					oldState[filter] = val;
				} else {
					const val = !oldState["filters"][filter];
					oldState["filters"][filter] = val;
				}
				return oldState
			})
		}).bind(this)
	}

	async load() {
		if(this.polling === true) {
			return
		}
		this.polling = true;
		//this.setState(() => ({loading: true}))
		console.log(this.getFilters())
		let res = await getCall("/submissions/get", this.getFilters())
		if(res.status !== "success") {
			apiToast(res)
			this.setState(() => ({loading: false}))
			this.polling = false;
			return
		}
		console.log(res.data)
		if(res.data == null) {
			this.setState(() => ({submissions: [], loading: false}))
		} else {
			this.setState(() => ({submissions: res.data.subs, subCount: res.data.count, problem_filtered: this.state.filters.problem_id, loading: false}))
		}
		this.polling = false;
	}

	getFilters() {
		const old = this.state.filters;
		var res = {};
		res.ordering = old.ordering
		res.ascending = old.ascending
		if(old.user_id > 0) {
			res.user_id = old.user_id
		}
		if(old.problem_id > 0) {
			res.problem_id = old.problem_id
		}
		if(old.status != "") {
			res.status = old.status
		}
		if(old.score >= 0) {
			res.score = old.score
		}
		if(old.lang !== "" ) {
			res.lang = old.lang
		}
		if(old.visible == "true" || old.visible == "false") {
			res.visible = old.visible == "true";
		}
		if(old.quality == "true" || old.quality == "false") {
			res.quality = old.quality == "true";
		}
		if(old.compile_error == "true" || old.compile_error == "false") {
			res.compile_error = old.compile_error == "true";
		}
		res.offset = (this.state.page - 1) * 50;
		return res;
	}

	fromQuery() {
		const params = new URLSearchParams(window.location.search);
		const user_id = params.get("user_id");
		const problem_id = params.get("problem_id");
		const status = params.get("status");
		const score = params.get("score");
		const lang = params.get("lang");
		const visible = params.get("visible");
		const quality = params.get("quality");
		const compile_error = params.get("compile_error");
	
		const ordering = params.get("ordering");

		if(ordering != "" && ordering != null) {
			this.state.filters.ordering = ordering
		}
		if(params.get("ascending") == "true") {
			this.state.filters.ascending = true
		}
		if(params.get("hiddenBar") == "true") {
			this.state.hiddenBar = true
		}

		let page = Number(params.get("page"));
		if(page == 0 || isNaN(page)) {
			page = 1;
		}
		this.state.page = page;

		if(user_id !== null && !isNaN(Number(user_id))) {
			this.state.filters.user_id = Number(user_id);
		}
		
		if(problem_id !== null && !isNaN(Number(problem_id))) {
			this.state.filters.problem_id = Number(problem_id);
		}

		if(status == "working" || status == "finished" || status == "waiting") {
			this.state.filters.status = status;
		}

		if(score !== null && !isNaN(Number(score))) {
			this.state.filters.score = Number(score);
		}

		if(lang !== null) {
			this.state.filters.lang = lang;
		}

		if(visible == "true" || visible == "false") {
			this.state.filters.visible = (visible == "true").toString();
		}

		if(quality == "true" || quality == "false") {
			this.state.filters.quality = (quality == "true").toString();
		}

		if(compile_error == "true" || compile_error == "false") {
			this.state.filters.compile_error = (compile_error == "true").toString();
		}
	}

	async getQuery() {
		var p = new URLSearchParams();
		const {filters, hiddenBar, page} = this.state;
		if(filters.user_id > 0) {
			p.append("user_id", filters.user_id);
		}
		if(filters.problem_id > 0) {
			p.append("problem_id", filters.problem_id);
		}
		if(filters.status !== "") {
			p.append("status", filters.status);
		}
		if(filters.score >= 0) {
			p.append("score", filters.score);
		}
		if(filters.lang !== "") {
			p.append("lang", filters.lang);
		}
		if(filters.visible !== "") {
			p.append("visible", filters.visible);
		}
		if(filters.quality !== "") {
			p.append("quality", filters.quality);
		}
		if(filters.compile_error !== "") {
			p.append("compile_error", filters.compile_error);
		}
		if(filters.ordering !== "id") {
			p.append("ordering", filters.ordering);
		}
		if(filters.ascending == true) {
			p.append("ascending", true);
		}
		if(hiddenBar == true) {
			p.append("hiddenBar", true);
		}
		if(page != 1) {
			p.append("page", page);
		}

		let url = window.location.origin + window.location.pathname + "?" + p.toString();
		try {
			await navigator.clipboard.writeText(url)
			createToast({status: "success", title: getText("copied")});
		} catch(e) {
			console.error(e);
			createToast({status: "error", title: getText("notCopied")})
		}
	}

	getRezStr() {
		const lang = window.platform_info?.language || 'en';
		return {
			'en': (n) => {
				if(n == 1) { return "One result" }
				return `${n} results`
			},
			'ro': (n) => {
				if(n == 1) { return "Un rezultat" }
				if(n < 20) { return `${n} rezultate` }
				return `${n} de rezultate`
			},
		}[lang](this.state.subCount)
	}

	formatStatus(sub) {
		if(sub.status === "finished") {
			return `${getText("evaluated")}: ${sub.score} ${getText("points")}`
		} else if(sub.status === "working") {
			return getText("evaluating")
		}
		return getText("waiting")
	}
	
	render({}, state) {
		let sideBar = <></>;
		if(!this.state.hiddenBar) {
			sideBar = (
				<div class="w-full lg:w-3/12 lg:pr-2 lg:border-r lg:border-gray-200">
					<div class="mt-6"/>
					<label class="block mb-2">
						<span class="form-label">{getText("page")}:</span>
						<input class="form-input" type="number" min="1" value={state.page} onchange={this.updater('page', true)}/>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("userID")} (0 = ignore):</span>
						<input class="form-input" type="number" min="0" value={state.filters.user_id} onchange={this.updater('user_id')}/>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("problemID")} (0 = ignore):</span>
						<input class="form-input" type="number" min="0" value={state.filters.problem_id} onchange={this.updater('problem_id')}/>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("status")}:</span>
						<select class="form-select" value={state.filters.status} onchange={this.updater('status')}>
							<option value="">-</option>
							<option value="finished">{getText("finished")}</option>
							<option value="working">{getText("working")}</option>
							<option value="waiting">{getText("waiting")}</option>
						</select>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("language")}:</span>
						<select class="form-select" value={state.filters.language} onchange={this.updater('language')}>
							<option value="">-</option>
							<option value="cpp">C++</option>
						</select>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("score")} (-1 = ignore):</span>
						<input class="form-input" type="number" min="-1" value={state.filters.score} onchange={this.updater('score')}/>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("visible")}:</span>
						<select class="form-select" value={state.filters.visible} onchange={this.updater('visible')}>
							<option value="">-</option>
							<option value="true">{getText("yes")}</option>
							<option value="false">{getText("no")}</option>
						</select>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("special")}:</span>
						<select class="form-select" value={state.filters.quality} onchange={this.updater('quality')}>
							<option value="">-</option>
							<option value="true">{getText("yes")}</option>
							<option value="false">{getText("no")}</option>
						</select>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("compileErr")}:</span>
						<select class="form-select" value={state.filters.compile_error} onchange={this.updater('compile_error')}>
							<option value="">-</option>
							<option value="true">{getText("yes")}</option>
							<option value="false">{getText("no")}</option>
						</select>
					</label>
					<label class="block mb-2">
						<span class="form-label">{getText("sorting")}:</span>
						<select class="form-select" value={state.filters.ordering} onchange={this.updater('ordering')}>
							<option value="id" default>{getText("id")}</option>
							<option value="score">{getText("score")}</option>
							<option value="max_time">{getText("maxTime")}</option>
							<option value="max_mem">{getText("maxMemory")}</option>
						</select>
					</label>
					<label class="block mb-2">
						<input type="checkbox" class="form-checkbox mr-2" checked={state.filters.ascending} onclick={this.updateCheckbox('ascending')}/>
						<span class="form-label">{getText("ascending")}</span>
					</label>
					<button class="btn btn-blue mb-4 mr-2" onclick={() => this.load()}>{getText("fetch")}</button>
				</div>
			);
		}
		let header = <h1>{this.getRezStr()}</h1>;
		let submissions = <div class="text-4xl mx-auto my-auto w-full mt-10 mb-10 text-center">{getText("noSubFound")}</div>;
		if(this.state.submissions.length > 0) {
			submissions = (<>
				{Number(this.state.problem_filtered) ? <p>{getText("problemSingle")} <a href={`/problems/${this.state.submissions[0].problem.id}`}>{this.state.submissions[0].problem.name}</a></p> : <></>}
				<table class="kn-table">
					<thead>
						<tr>
							<th scope="col" class="w-12 text-center px-4 py-2">
								{getText("id")}	
							</th>
							<th scope="col">
								{getText("author")}	
							</th>
							<th scope="col">
								{getText("uploadDate")}	
							</th>
							<th scope="col" v-if="Number(filters.problem_id) == 0">
								{getText("problemSingle")}	
							</th>
							<th scope="col">
								{getText("time")}	
							</th>
							<th scope="col" class="w-1/6">
								{getText("status")}
							</th>
						</tr>
					</thead>
					<tbody>
						{this.state.submissions.map(sub => (
							<tr key={sub.sub.id} class={sub.sub.quality ? 'quality-highlight' : 'kn-table-row'}>
								<th scope="row" class="text-center px-2 py-1">
									{sub.sub.id}
								</th>
								<td class="px-2 py-1">
									<a href={`/profile/${sub.author.name}`}>{sub.author.name}</a>
								</td>
								<td class="text-center px-2 py-1">
									{dayjs(sub.sub.created_at).format('DD/MM/YYYY HH:mm')}
								</td>
								<td class="text-center px-2 py-1">
									<a href={`/problems/${sub.problem.id}`}>{sub.problem.name}</a>
								</td>
								<td class="text-center px-2 py-1">
									{sub.sub.max_time == -1 ? "-" : Math.floor(sub.sub.max_time * 1000) + "ms"}
								</td>
								<td class={sub.sub.status === 'finished' ? 'text-black' : '' + ' text-center px-4 py-2'} style={sub.sub.status == 'finished' ? 'background-color: ' + getGradient(sub.sub.score, 100) : ''}>
									<a href={`/submissions/${sub.sub.id}`}>{this.formatStatus(sub.sub)}</a>
								</td>

							</tr>
						))}
					</tbody>
				</table>
			</>);
		}

		return (
			<div class="flex flex-wrap lg:border-t lg:border-gray-200">
				{sideBar}
				<div class="mw-full mt-2 lg:flex-1 lg:px-2 lg:py-2">
					{state.loading ? <BigSpinner/> : <>{header}{submissions}</>}

					<button class="btn btn-left mr-2" onclick={() => this.setState((oldState) => ({hiddenBar: !oldState.hiddenBar}))}>
						{state.hiddenBar ? getText("show") : getText("hide")} {getText("filterBar")}
					</button>
					<button class="btn mb-4" onclick={() => this.getQuery()}>{getText("filterLink")}</button>

				</div>
			</div>
		)
	}
}

register(SubmissionsViewer, 'kn-sub-viewer', [])

