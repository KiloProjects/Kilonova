import {h, Fragment, Component, render} from 'preact';
import register from 'preact-custom-element';
import {getText} from '../translation';
import {createToast, apiToast} from '../toast';
import {postCall} from '../net';

class ProblemEditor extends Component {
	state = {
		problem: {},
	}
	constructor(props) {
		super();
		this.state.problem = props.problem;
		//this.org_type = props.problem.type; 
	}

	async update(e) {
		e.preventDefault();
		const {problem} = this.state;
		if(problem.name === "") {
			createToast({status: "error", description: getText('emptyTitle')});
			return
		}
		let data = {
			name: problem.name,
			default_points: problem.default_points,

			source_credits: problem.source_credits,
			author_credits: problem.author_credits,
			
			type: problem.type,
			console_input: problem.console_input,
			test_name: problem.test_name,
	
			memory_limit: problem.memory_limit,
			time_limit: problem.time_limit,

		};
		if(window.platform_info.admin) {
			data.visible = problem.visible;
		}
		let res = await postCall(`/problem/${problem.id}/update/`, data)
		/*
		if(res.status === "success" && this.org_type != problem.type) {
			window.location.reload();
			return;
		}
		*/
		apiToast(res);

	}
	
	async delete(e) {
		e.preventDefault();
		if(!confirm(getText("confirmProblemDelete"))) {
			return
		}
		let res = await postCall(`/problem/${this.problem.id}/delete`, {})
		if(res.status === "success") {
			window.location.assign("/");
			return
		}
		apiToast(res)
	}

	updater(att) {
		return ((e) => {
			console.log("Updating", JSON.stringify(att), e.target.valueAsNumber)
			const val = e.target.valueAsNumber || e.target.value;
			this.setState(oldState => {
				oldState["problem"][att] = val;
				return oldState
			})
		}).bind(this)
	}

	render({}, {problem}) {
		return (
			<div class="segment-container">
				<form onsubmit={this.update.bind(this)}>
					<div class="block my-2">
						<label>
							<span class="form-label">{getText("title")}:</span>
							<input class="form-input" type="text" value={problem.name} onchange={this.updater('name')} />
						</label>
					</div>
					<div class="block my-2">
						<label>
							<span class="form-label">{getText("author")}:</span>
							<input class="form-input" type="text" size="50" value={problem.author_credits} onchange={this.updater('author_credits')} />
						</label>
					</div>
					<div class="block my-2">
						<label>
							<span class="form-label">{getText("source")}:</span>
							<input class="form-input" type="text" size="50" value={problem.source_credits} onchange={this.updater('source_credits')} />
						</label>
					</div>
					<div class="block my-2">
						<label>
							<span class="form-label">{getText("pbType")}:</span>
							<select class="form-select" value={problem.type} onchange={this.updater('type')}>
								<option value="classic">{getText("classic")}</option>
								<option value="custom_checker">{getText("checker")}</option>
							</select>
						</label>
					</div>
					{window.platform_info.admin ? (
						<div class="block my-2">
							<label>
								<input class="form-checkbox" type="checkbox" value={problem.visible} onchange={this.updater('visible')}/>
								<span class="form-label ml-2">{getText("visiblePb")}</span>
							</label>
						</div>
					) : <></>}
					<div class="block my-2">
						<label>
							<input class="form-checkbox" type="checkbox" value={problem.console_input} onchange={this.updater('console_input')}/>
							<span class="form-label ml-2">{getText("consoleInput")}</span>
						</label>
					</div>
					<div class="block my-2" v-if="!problem.console_input">
						<label>
							<span class="mr-2 text-xl">{getText("testName")}:</span>
							<input class="form-input" type="text" value={problem.test_name} onchange={this.updater('test_name')} />
						</label>
					</div>
					<div class="block my-2">
						<label>
							<span class="form-label">{getText("memoryLimit")}:</span>
							<input type="number" class="form-input" id="memoryTotal" placeholder="Limită de memorie (total)" min="0" step="128" max="131072" value={problem.memory_limit} onchange={this.updater('memory_limit')}/>
							<span class="ml-1 text-xl">KB</span>
						</label>
					</div>
					<div class="block my-2">
						<label>
							<span class="form-label">{getText("timeLimit")}:</span>
							<input type="number" class="form-input" id="timeLimit" placeholder="Limită de timp..." min="0" step="0.01" value={problem.time_limit} onchange={this.updater('time_limit')}/>
							<span class="ml-1 text-xl">{getText("seconds")}</span>
						</label>
					</div>
					<div class="block my-2">
						<label>
							<span class="form-label">{getText("defaultPoints")}:</span>
							<input class="form-input" type="number" min="0" max="100" step="1" pattern="\d*" value={problem.default_points} onchange={this.updater('default_points')} />
						</label>
					</div>
					<button type="submit" class="btn btn-blue">{getText("updateProblem")}</button>

				</form>
			</div>
		);
	}

}

export function createProblemEditor(domid, problem) {
	render(<ProblemEditor problem={problem}/>, document.getElementById(domid))
}
