import _ from 'underscore';

var SubmissionsApp = {
	data: () => {
		return {
			submissions: [],
			filters: {user_id: null, problem_id: null, status: "", lang: "", visible: "", quality: "", compile_error: "", offset: 0, ordering: "id", ascending: false},
			page: 1,
			loading: true,
			hiddenBar: false,
		}
	},
	methods: {
		poll: async function() {
			this.loading = true
			let res = await bundled.getCall("/submissions/get", this.getFilters(this.filters))
			if(res.status !== "success") {
				bundled.apiToast(res)
				this.loading = false
				return
			}
			if(res.data == null) {
				this.submissions = []
			} else {
				this.submissions = res.data.subs
				this.subCount = res.data.count
			}
			this.loading = false
		},
		formatStatus: function(sub) {
			if(sub.status === "finished") {
				return "Evaluat: " + sub.score + " puncte"
			} else if(sub.status === "working") {
				return "În evaluare..."
			}
			return "În așteptare..."
		},
		getRezStr: function() {
			let l = this.subCount;
			if(l == 1) {
				return "Un rezultat."
			}
			if(l < 20) {
				return l + " rezultate."
			}
			return l + " de rezultate."
		},
		getFilters: function(old) {
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
			res.offset = (this.page - 1) * 50;
			return res;
		},
		getGradient: function(score, maxscore) {
			return bundled.getGradient(score, maxscore)
		},
		timeStr: function(d) {
			return bundled.dayjs(d).format('DD/MM/YYYY HH:mm')
		},
		getQuery: async function() {
			var p = new URLSearchParams();
			if(this.filters.user_id > 0) {
				p.append("user_id", this.filters.user_id);
			}
			if(this.filters.problem_id > 0) {
				p.append("problem_id", this.filters.problem_id);
			}
			if(this.filters.status !== "") {
				p.append("status", this.filters.status);
			}
			if(this.filters.score >= 0) {
				p.append("score", this.filters.score);
			}
			if(this.filters.lang !== "") {
				p.append("lang", this.filters.lang);
			}
			if(this.filters.visible !== "") {
				p.append("visible", this.filters.visible);
			}
			if(this.filters.quality !== "") {
				p.append("quality", this.filters.quality);
			}
			if(this.filters.compile_error !== "") {
				p.append("compile_error", this.filters.compile_error);
			}
			if(this.filters.ordering !== "id") {
				p.append("ordering", this.filters.ordering);
			}
			if(this.filters.ascending == true) {
				p.append("ascending", true);
			}
			if(this.hiddenBar == true) {
				p.append("hiddenBar", true);
			}
			p.append("page", this.page);
			let url = window.location.origin + window.location.pathname + "?" + p.toString();
			try {
				await navigator.clipboard.writeText(url)
				bundled.createToast({status: "success", title: "Copied URL to clipboard"});
			} catch(e) {
				console.error(e);
				bundled.createToast({status: "error", title: "Couldn't copy to clipboard"})
			}
		},
		sizeFormatter: (sz) => bundled.sizeFormatter(sz),
	},
	watch: {
		filters: {
			handler: _.throttle(async function() {
				console.log(":fundita:")
				await this.poll()
			}, 200),
			problem_id: function(val) {
				if(val == 0) {
					this.filters.problem_id = null
				}
			},
			user_id: function(val) {
				if(val == 0) {
					this.filters.user_id = null
				}
			},
			deep: true
		},
		page: async function() {
			await this.poll()
		}
	},
	created: function() {
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
			this.filters.ordering = ordering
		}
		if(params.get("ascending") == "true") {
			this.filters.ascending = true
		}
		if(params.get("hiddenBar") == "true") {
			this.hiddenBar = true
		}

		let page = Number(params.get("page"));
		if(page == 0 || !isNaN(page)) {
			page = 1;
		}
		this.page = page;

		if(user_id !== null && !isNaN(Number(user_id))) {
			this.filters.user_id = Number(user_id);
		}
		
		if(problem_id !== null && !isNaN(Number(problem_id))) {
			this.filters.problem_id = Number(problem_id);
		}

		if(status == "working" || status == "finished" || status == "waiting") {
			this.filters.status = status;
		}

		if(score !== null && !isNaN(Number(score))) {
			this.filters.score = Number(score);
		}

		if(lang !== null) {
			this.filters.lang = lang;
		}

		if(visible == "true" || visible == "false") {
			this.filters.visible = (visible == "true").toString();
		}

		if(quality == "true" || quality == "false") {
			this.filters.quality = (quality == "true").toString();
		}

		if(compile_error == "true" || compile_error == "false") {
			this.filters.compile_error = (compile_error == "true").toString();
		}

		this.poll()
	},
	delimiters: ['${', '}']	
}

export {SubmissionsApp};
