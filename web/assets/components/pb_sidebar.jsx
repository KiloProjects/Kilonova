import {h, Fragment, Component} from 'preact';
import register from 'preact-custom-element';

import {getText} from '../translation';
import dayjs from 'dayjs';

import {SmallSpinner} from './common';
import {getCall} from '../net';

const subLimit = 5;

// ProblemSidebar handles the rendering and updating of submissions
class ProblemSidebar extends Component {
	state = {
		subs: [],
		total: -1,
		hidden: -1,
		limited: false,
		loading: true,
	};
	constructor() {
		super();
		this.polling = false
		document.addEventListener('kn-poll', this.handleEvent.bind(this))
	}

	async componentDidMount() {
		await this.poll()
	}

	async poll() {
		if(this.polling == true) {
			return
		}
		this.polling = true;
		console.info("Polling...");
		let res = await getCall("/submissions/get", {user_id: window.platform_info.user_id, problem_id: this.props.pbid, limit: subLimit})
		if(res.status !== "success") {
			console.error(res);
			this.polling = false;
			return
		}
		if(res.data != null) {
			let limited = false;
			let hidden = -1;
			if(res.data.count != res.data.subs.length) { // it was sliced
				limited = true;
				hidden = res.data.count - res.data.subs.length;
			}
			this.setState(() => ({
				subs: res.data.subs,
				total: res.data.count,
				hidden,
				limited,
				loading: false,
			}))
		}
		this.polling = false;
	}

	async handleEvent(e) {
		e.preventDefault();
		await this.poll();
	}

	seeMore(hidden) {
		switch(window.platform_info.language) {
		case 'ro':
			return `Vezi încă ${hidden} ${hidden>10 ? 'de' : ''} submisii`
		default:
			en = `See ${hidden} more submission${hidden==1?'':'s'}`
		}
	}

	render({pbid, uid}, state) {
		let content = <SmallSpinner/>;
		if(!state.loading) {
			if(state.subs.length > 0) {
				content = (
					<div>
						{state.subs.map(sub => (
							<a class="black-anchor flex justify-between items-center rounded py-1 px-2 hoverable" key={sub.sub.id} href={`/submissions/${sub.sub.id}`}>
								<span>{dayjs(sub.sub.created_at).format('DD/MM/YYYY HH:mm')}</span>
								<span class="rounded-md px-2 py-1 bg-teal-700 text-white text-sm">{sub.sub.status == 'finished' ? sub.sub.score : (sub.sub.status == 'working' ? <i class="fas fa-cog animate-spin"></i> : <i class="fas fa-clock"></i>)}</span>
							</a>
						))}
					</div>
				);
			} else {
				content = <p class="px-2">{getText("noSub")}</p>;
			}
		}
		return (
			<>
				<div class="h-0 w-full border-t border-gray-200"></div>
				<div class="mt-2 lg:pl-2 lg:pb-2">
					{getText("oldSubs")}
					{content}
					{state.limited && state.hidden > 0 ? (
						<a href={`/submissions/?problem_id=${pbid}&user_id=${uid}`}>
							{this.seeMore(state.hidden)}
						</a>
					) : <>{JSON.stringify(state)}</>}
				</div>
			</>	
		)
	}
}

register(ProblemSidebar, 'kn-pb-sidebar', ['pbid'])

