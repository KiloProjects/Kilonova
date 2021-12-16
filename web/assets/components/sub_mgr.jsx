import {h, Fragment, render, Component} from 'preact';
import {useReducer} from 'preact/hooks';
import register from 'preact-custom-element';

import {getText} from '../translation';
import {createToast, apiToast} from '../toast';
const slugify = str => str.toLowerCase().trim().replace(/[^\w\s-]/g, '').replace(/[\s_-]+/g, '-').replace(/^-+|-+$/g, '');

import {BigSpinner} from './common';

import {downloadBlob, parseTime, sizeFormatter, getGradient} from '../util';

import {getCall, postCall} from '../net';

// TODO: Test all buttons


function downloadCode(sub) {
	var file = new Blob([sub.code], {type: 'text/plain'})
	var filename = `${slugify(sub.problem.name)}-${sub.id}.${sub.language}`
	downloadBlob(file, filename);
}

async function copyCode(sub) {
	await navigator.clipboard.writeText(sub.code).then(() => {
		createToast({status: "success", description: getText("copied")})
	}, (err) => {
		createToast({status: "error", description: getText("notCopied")})
		console.error(err)
	})
}

function Summary({sub}) {
	return (
		<>
			<h2>{getText("sub")} {sub.id}</h2>
			<p>{getText("author")}: <a href={`/profile/${sub.author.name}`}>{sub.author.name}</a></p>
			<p>{getText("problemSingle")}: <a href={`/problems/${sub.problem.id}`}>{sub.problem.name}</a></p>
			<p>{getText("uploadDate")}: {parseTime(sub.created_at)}</p>
			<p>{getText("status")}: {sub.status}</p>
			<p>{getText("language")}: {sub.language}</p>
			{sub.quality ? <p><i class="fas fa-star text-yellow-300"></i> {getText("qualitySub")}</p> : <></>}
			{sub.code ? <p>{getText("size")}: {sizeFormatter(sub.code.length)}</p> : <></>}
			{sub.problem.default_points > 0 ? <p>{getText("defaultPoints")}: {sub.problem.default_points}</p> : <></>}
			{sub.status === 'finished' ? (<>
				<p>{getText("score")}: {sub.score}</p>
				<p>{getText("maxTime")}: {Math.floor(sub.max_time * 1000)} ms</p>
				<p>{getText("maxMemory")}: {sizeFormatter(sub.max_memory*1024, 1, true)}</p>
			</>) : <></>}
			{sub.compile_error.Bool ? (<>
				<h4>{getText("compileErr")}</h4>
				<h5>{getText("compileMsg")}:</h5>
				<pre>{sub.compile_message.String}</pre>
			</>) : <></>}
		</>
	)
}

function SubCode({sub, dispatcher}) {

	async function toggleVisible() {
		let res = await postCall("/submissions/setVisible", {visible: !sub.visible, id: sub.id});
		apiToast(res)
		dispatcher('toggleVisible')
	}

	async function toggleQuality() {
		let res = await postCall("/submissions/setQuality", {quality: !sub.quality, id: sub.id});
		apiToast(res)
		dispatcher('toggleQuality')
	}

	return (
		<>
			<h3>{getText("sourceCode")}</h3>
			<pre>
				<code class="hljs" dangerouslySetInnerHTML={{__html: hljs.highlight(sub.code, {language: sub.language}).value}}>
					Rendering...		
				</code>
			</pre>
			<div class="block my-2">
				<button class="btn btn-blue mr-2 text-semibold text-lg" onclick={() => copyCode(sub)}>
					{getText("copy")}
				</button>
				<button class="btn btn-blue text-semibold text-lg" onclick={() => downloadCode(sub)}>
					{getText("download")}
				</button>
			</div>
			{sub.editor ? (
				<button class="btn btn-blue block my-2 text-semibold text-lg" onclick={toggleVisible}>
					<i class="fas fa-share-square mr-2"></i>{getText("makeCode")} {sub.visible ? getText("invisible") : getText("visible")}
				</button>
			) : <></>}
			{sub.problemEditor ? (
				<button class="btn btn-blue block my-2 text-semibold text-lg" onclick={toggleQuality}>
					<i class="fas fa-star mr-2"></i>{sub.quality ? getText("dropQuality") : getText("makeQuality")}
				</button>
			) : <></>}
		</>
	)
}


function TestTable({sub}) {
	function testSubTasks(test_id) {
		let stks = [];
		for(let st of sub.subTasks) {
			if(st.tests.includes(test_id)) {
				stks.push(st.visible_id);
			}
		}
		return stks;
	}

	return (
		<table class="kn-table">
			<thead>
				<tr>
					<th class="py-1" scope="col">{getText("id")}</th>
					<th scope="col">{getText("time")}</th>
					<th scope="col">{getText("memory")}</th>
					<th scope="col">{getText("verdict")}</th>
					<th scope="col">{getText("score")}</th>
					{sub.problemEditor ? <th scope='col'>{getText("output")}</th> : <></>}
					{sub.subTasks.length > 0 ? <th scope='col'>{getText("subTasks")}</th> : <></>}
				</tr>
			</thead>
			<tbody>
				{sub.subTests.map(test => (
					<tr class="kn-table-row" key={"kn_test"+test.subtest.id}>
						<th class="py-2" scope="row">
							{test.pb_test.visible_id}
						</th>
						{test.subtest.done ? (<>
							<td>
								{Math.floor(test.subtest.time * 1000)} ms
							</td>
							<td>
								{sizeFormatter(test.subtest.memory*1024, 1, true)}
							</td>
							<td>
								{test.subtest.verdict}
							</td>
							<td class="text-black" style={'background-color: ' + getGradient(sub.subTasks.length > 0 ? test.subtest.score : test.pb_test.score, 100)}>
								{sub.subTasks.length > 0 ? (<>
									{test.subtest.score}% {getText("correct")}
								</>) : (<>
									{Math.round(test.pb_test.score * test.subtest.score / 100.0)} / {test.pb_test.score}
								</>) }
							</td>
						</>) : (<>
							<td>
							</td>
							<td>
							</td>
							<td>
								<div class='fas fa-spinner animate-spin' role='status'></div> {getText("waiting")}
							</td>
							<td>
								-
							</td>
						</>)}
						{sub.problemEditor ? <td><a href={"/proposer/get/subtest_output/"+test.subtest.id}>{getText("output")}</a></td> : <></>}
						{sub.subTasks.length > 0 ? <td>{testSubTasks(test.pb_test.id).join(', ')}</td> : <></>}
					</tr>
				))}
			</tbody>
		</table>
	)
}

function SubTasks({sub}) {
		function stkScore(subtask) {
			let stk_score = 100;
			for(let testID of subtask.tests) {
				let actualTest = sub.subTestIDs[testID];
				if(actualTest.subtest.score < stk_score) {
					stk_score = actualTest.subtest.score;
				}
			}
			return stk_score;			
		}
		return (
			<div class="my-2">
				<div class="list-group my-1 list-group-mini">
					{sub.subTasks.map(subtask => (
						<details class="list-group-item" key={"stk_"+subtask.id}>
							<summary class="flex justify-between">
								<span>{getText("subTask")} #{subtask.visible_id}</span>
								<span class="rounded-full py-1 px-2 text-base text-white font-semibold" style={`background-color: ${getGradient(stkScore(subtask), 100)}`}>{Math.round(subtask.score * stkScore(subtask) / 100.0)} / {subtask.score}</span>
							</summary>	
							<div class="list-group m-1 list-group-mini">
									{subtask.tests.map(testID => {
										let actualTest = sub.subTestIDs[testID];
										return (
											<div class="list-group-item flex justify-between">
												<span>{getText("test")} #{actualTest.pb_test.visible_id}</span>
												<span class="rounded-full py-1 px-2 text-base text-white font-semibold" style={`background-color: ${getGradient(actualTest.subtest.score, 100)}`}> 
													{Math.round(subtask.score * actualTest.subtest.score / 100.0)} / {subtask.score}
												</span>
											</div>
										)
									})}
								</div>
						</details>
					))}
				</div>
				<details>
					<summary>{getText("seeTests")}</summary>
					<TestTable sub={sub}/>
				</details>
			</div>
		)
	}


export class SubmissionManager extends Component {
	state = {
		sub: {},
	}
	constructor() {
		super();
		this.poll_mu = false
		this.finished = false

		this.poller = null

	}

	doAction(action) {
		switch(action) {
			case 'toggleVisible':
				this.setState((oldState) => {
					oldState.sub.visible = !oldState.sub.visible;
					return oldState
				})
				break
			case 'toggleQuality':
				this.setState((oldState) => {
					oldState.sub.quality = !oldState.sub.quality;
					return oldState
				})
				break
			default:
				console.error("wtf")
		}
	}

	async componentDidMount() {
		await this.poll()
		if(!this.finished) {
			console.info("Started poller")
			this.poller = setInterval(async () => await this.poll(), 2000)
		}
	}

	stopPoller() {
		if(this.poller == null) {
			return
		}
		console.info("Stopped poller")
		clearInterval(this.poller)
		this.poller = null
	}


	async poll() {
		if(this.poll_mu === false) this.poll_mu = true
		else return
		console.log("Poll submission #", this.props.id)
		let res = await getCall("/submissions/getByID", {id: this.props.id, expanded: true})
		if(res.status !== "success") {
			apiToast(res)
			console.error(res)
			this.poll_mu = false
			return
		}

		res = res.data
		let newState = {}
		
		newState.sub = res.sub
		newState.sub.editor = res.sub_editor
		newState.sub.problemEditor = res.problem_editor
		newState.sub.author = res.author
		newState.sub.problem = res.problem
		if(res.subtasks) {
			newState.sub.subTasks = res.subtasks
		}

		if(res.subtests) {
			newState.sub.subTests = res.subtests
			newState.sub.subTestIDs = {}
			for(let subtest of res.subtests) {
				newState.sub.subTestIDs[subtest.pb_test.id] = subtest;
			}
		}

		this.setState(() => newState)

		if(res.sub.status === "finished") {
			this.stopPoller()
			this.finished = true;
			this.forceUpdate();
		}

		this.poll_mu = false
	}

	render() {
		let {sub} = this.state;
		if(sub && Object.keys(sub).length == 0) {
			return <BigSpinner/>
		}
		let nodes = [<Summary sub={sub}/>]
		if(sub.subTests.length > 0 && !sub.compile_error.Bool) {
			if(sub.subTasks.length > 0) {
				nodes.push(<SubTasks sub={sub}/>)
			} else {
				nodes.push(<TestTable sub={sub}/>)
			}
		}
		if(sub.code != null) {
			nodes.push(<SubCode sub={sub} dispatcher={(val) => this.doAction(val)}/>)
		}
		return nodes
	}
}

register(SubmissionManager, 'kn-sub-mgr', ['id'])
