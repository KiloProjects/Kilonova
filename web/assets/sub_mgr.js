import slugify from 'slugify';

export class SubmissionManager {
	constructor(id, replace_id) {
		this.id = id
		this.replace_id = replace_id
		this.poll_mu = false
		this.subAuthor = {}
		this.subProblem = {}
		this.problemEditor = false
		this.sub = {}
		this.subTests = []
		this.poller = null
		this.finished = false

		this.subTasks = []
		this.subTestIDs = {}
	}

	async startPoller() {
		console.log("Started poller")
		await this.poll()
		if(!this.finished) {
			this.poller = setInterval(async () => this.poll(), 2000)
		}
	}

	stopPoller() {
		if(this.poller == null) {
			return
		}
		console.log("Stopped poller")
		clearInterval(this.poller)
		this.poller = null
	}

	downloadCode() {
		var file = new Blob([this.sub.code], {type: 'text/plain'})
		var filename = `${slugify(this.subProblem.name, {lower: true})}-${this.id}.${this.sub.language}`
		bundled.downloadBlob(file, filename);
	}

	async copyCode() {
		await navigator.clipboard.writeText(this.sub.code)
		bundled.createToast({status: "success", description: "Cod copiat"})
	}

	async poll() {
		if(this.poll_mu === false) this.poll_mu = true
		else return
		console.log("Poll", this.id)
		let res = await bundled.getCall("/submissions/getByID", {id: this.id, expanded: true})
		if(res.status !== "success") {
			bundled.apiToast(res)
			console.error(res)
			this.poll_mu = false
			return
		}

		console.log(res)

		res = res.data
		if(res.subtests) {
			this.subTests = res.subtests
			this.subTestIDs = {}
			for(let subtest of res.subtests) {
				this.subTestIDs[subtest.pb_test.id] = subtest;
			}
		}
		
		this.sub = res.sub
		this.subEditor = res.sub_editor
		this.problemEditor = res.problem_editor
		this.subAuthor = res.author
		this.subProblem = res.problem
		if(res.subtasks) {
			this.subTasks = res.subtasks
		}

		if(this.sub.status === "finished") {
			this.stopPoller()
			this.finished = true
		}

		this.render()
		this.poll_mu = false
	}

	async toggleVisible() {
		let res = await bundled.postCall("/submissions/setVisible", {visible: !this.sub.visible, id: this.id});
		bundled.createToast({
			status: res.status,
			title: (res.status == "success" ? (this.sub.visible ? "Made visible" : "Made invisible") : "Error changing visibility"),
			description: res.data
		});
		this.sub.visible = !this.sub.visible
		this.render();
	}

	async toggleQuality() {
		let res = await bundled.postCall("/submissions/setQuality", {quality: !this.sub.quality, id: this.id});
		this.sub.quality = !this.sub.quality
		bundled.createToast({
			status: res.status,
			title: (res.status == "success" ? (this.sub.quality ? "Made quality" : "Stopped being quality") : "Error changing visibility"),
			description: res.data
		});
		this.render();
	}

	summaryNode() {
		let rez = document.createElement('div')
		let html = ""
		html += `<h2>Submission ${this.sub.id}</h2>
			<p>Autor: <a href="/profile/${this.subAuthor.name}">${this.subAuthor.name}</a></p>
			<p>Data încărcării: ${bundled.parseTime(this.sub.created_at)}</p>
			<p>Status: ${this.sub.status}</p>`;
		if(this.sub.quality) {
			html += `<p><i class="fas fa-star text-yellow-300"></i> Submisie evidențiată</p>`
		}
		if(this.sub.code) {
			html += `<p>Dimensiune: ${bundled.sizeFormatter(this.sub.code.length)}</p>`
		}
		html += `<p>Limbaj: ${this.sub.language}</p><p>Problemă: <a href="/problems/${this.subProblem.id}">${this.subProblem.name}</a></p>`
		if(this.subProblem.default_points > 0) {
			html += `<p>Puncte din oficiu: ${this.subProblem.default_points}</p>`
		}
		if(this.sub.status === 'finished') {
			html += `<p>Scor: ${this.sub.score}</p>`
		}
		if(this.sub.compile_error.Bool) {
			html += `<h4>Eroare de compilare</h4><h5>Mesaj Evaluare:</h5><pre>${this.sub.compile_message.String}</pre>`
		}
		rez.innerHTML = html
		return rez;
	}

	tableColGen(text) {
		let td = document.createElement('td')
		td.innerHTML = text
		return td
	}

	subTasksNode() {
		let rezz = document.createElement('div');
		let rez = document.createElement('div')
		rez.classList.add('list-group', 'my-2')
		for(let subtask of this.subTasks) {
			let row = document.createElement('details')
			row.classList.add('list-group-item')
			
			let sum = document.createElement('summary')
			sum.classList.add('flex', 'justify-between')
			
			let stk_score = 100;
			let subtests = document.createElement('div')
			subtests.classList.add('list-group', 'm-1')
			for(let testID of subtask.tests) {
				let roww = document.createElement('div')
				roww.classList.add('list-group-item', 'flex', 'justify-between')

				let actualTest = this.subTestIDs[testID];
				if(actualTest.subtest.score < stk_score) {
					stk_score = actualTest.subtest.score;
				}
				roww.innerHTML = `<span>Testul #${actualTest.pb_test.visible_id}</span><span class="rounded-full py-1 px-2 text-base text-white font-semibold" style="background-color: ${bundled.getGradient(actualTest.subtest.score, 100)}">${Math.round(subtask.score * actualTest.subtest.score / 100.0)} / ${subtask.score}</span>`
				
				subtests.appendChild(roww)
			}

			sum.innerHTML = `<span>SubTask-ul #${subtask.visible_id}</span><span class="rounded-full py-1 px-2 text-base text-white font-semibold" style="background-color: ${bundled.getGradient(stk_score, 100)}">${Math.round(subtask.score * stk_score / 100.0)} / ${subtask.score}</span>`
			
			row.appendChild(sum)
			row.appendChild(subtests)
			rez.appendChild(row)
		}
		rezz.appendChild(rez)

		if(this.subEditor) {
			let tmp = document.createElement('details');
			let tmp1 = document.createElement('summary');
			tmp1.innerText = "(temporar) Vizualizare teste";
			tmp.appendChild(tmp1);
			tmp.appendChild(this.tableNode());
			rezz.appendChild(tmp);
		}

		return rezz
	}

	tableNode() {
		let rez = document.createElement('table')
		rez.classList.add('kn-table')
		let head = document.createElement('thead')
		head.innerHTML = `<tr><th class="py-2" scope="col">ID</th><th scope="col">Timp</th><th scope="col">Memorie</th><th scope="col">Verdict</th><th scope="col">Scor</th>${this.problemEditor ? "<th scope='col'>Output</th>" : ""}</tr>`
		let body = document.createElement('tbody')
		for(let test of this.subTests) {
			let row = document.createElement('tr')
			row.classList.add('kn-table-row')
			
			let vid = document.createElement('th')
			vid.innerText = test.pb_test.visible_id
			vid.classList.add('py-3')
			vid.scope = "row"
			row.appendChild(vid)
			
			let time = this.tableColGen("")
			let mem = this.tableColGen("")
			let verdict = this.tableColGen("<div class='fas fa-spinner animate-spin' role='status'></div> În așteptare...")
			let score = this.tableColGen(`${Math.round(test.pb_test.score * test.subtest.score / 100.0)} / ${test.pb_test.score}`)
			if(test.subtest.done) {
				verdict.innerHTML = test.subtest.verdict
				
				time.innerHTML = Math.floor(test.subtest.time * 1000) + " ms";
				mem.innerHTML = bundled.sizeFormatter(test.subtest.memory*1024, 1, true)

				score.classList.add("text-black")
				score.style = "background-color:" + bundled.getGradient(test.subtest.score, 100) + ";"
			}

			row.appendChild(time)
			row.appendChild(mem)
			row.appendChild(verdict)
			row.appendChild(score)
			if(this.problemEditor) {
				let out = this.tableColGen("")
				if(test.subtest.done) {
					out.innerHTML = `<a href="/proposer/get/subtest_output/${test.subtest.id}">Output</a>`
				}
				row.appendChild(out)
			}

			body.appendChild(row);
		}
		rez.appendChild(head)
		rez.appendChild(body)
		return rez;
	}

	codeNode() {
		let rez = document.createElement('div')
		
		// header
		let header = document.createElement('h3')
		header.innerText = "Codul Sursă:"
		rez.appendChild(header)

		// code
		let code = document.createElement('pre')
		let c = document.createElement('code')
		c.classList.add('hljs', this.sub.language)
		c.innerHTML = hljs.highlight(this.sub.language, this.sub.code).value
		code.appendChild(c)
		rez.appendChild(code)

		let dv = document.createElement('div')
		dv.classList.add('block', 'my-2')

		let btn = document.createElement('button')
		btn.classList.add('btn', 'btn-blue', 'mr-2', 'text-semibold', 'text-lg')
		btn.innerText = "Copiere"
		btn.onclick = async () => await this.copyCode()
		dv.appendChild(btn)

		let btn1 = document.createElement('button')
		btn1.classList.add('btn', 'btn-blue', 'text-semibold', 'text-lg')
		btn1.innerText = "Descărcare"
		btn1.onclick = () => this.downloadCode()
		dv.appendChild(btn1)
		rez.appendChild(dv)

		if(this.subEditor) {
			let btn = document.createElement('button');
			btn.classList.add('btn', 'btn-blue', 'block', 'my-2', 'text-semibold', 'text-lg');
			btn.innerHTML = `<i class="fas fa-share-square mr-2"></i>Fă codul ${this.sub.visible ? "invizibil" : "vizibil"}</button>`;
			btn.onclick = () => this.toggleVisible();
			rez.appendChild(btn);
		}
		if(this.problemEditor) {
			let btn = document.createElement('button');
			btn.classList.add('btn', 'btn-blue', 'block', 'my-2', 'text-semibold', 'text-lg');
			btn.innerHTML = `<i class="fas fa-star mr-2"></i>${this.sub.quality ? "nu e evidențiată" : "trebuie evidențiată"}</button>`;
			btn.onclick = () => this.toggleQuality();
			rez.appendChild(btn);
		}

		return rez;
	}

	viewNode() {
		let rez = document.createElement('div')
		rez.appendChild(this.summaryNode())
		if(this.subTests.length > 0 && !this.sub.compile_error.bool) {
			if(this.subTasks.length > 0) {
				rez.appendChild(this.subTasksNode())
			} else {
				rez.appendChild(this.tableNode())
			}
		}
		if(this.sub.code != null) {
			rez.appendChild(this.codeNode())
		}
		return rez;
	}

	render() {
		let node = this.viewNode()
		node.id = this.replace_id

		let target = document.getElementById(this.replace_id);
		target.parentNode.replaceChild(node, target);
	}
}
