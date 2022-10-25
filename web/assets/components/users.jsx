import {h, Fragment, Component, render} from 'preact';
import register from 'preact-custom-element';
import {getText} from '../translation';
import {BigSpinner, Paginator} from './common';
import {getCall} from '../net';

function UserTable({users}) {
	console.log(users)
	return (
		<div class="list-group">
			<div class="list-group-head font-bold">
				User
			</div>
			{users.map(user => (
				<a href={`/profile/${user.name}`} class="list-group-item inline-flex align-middle items-center">
					<img class="flex-none mr-2 rounded" src={`/api/user/getGravatar?name=${user.name}&s=32`}/> #{user.id}: {user.name}
				</a>
			))}
		</div>
	);
}

class UserList extends Component {
	state = {
		users: [] 
	};
	constructor() {
		super();
		this.lister = UserTable
		if(this.props && ('lister' in this.props)) {
			this.lister = this.props.lister;
		}

		this.page = 1
		this.numPages = -1
	}

	async componentDidMount() {
		await this.poll()
	}

	setPage(pg) {
		this.page = pg
		this.poll()
	}

	async poll() {
		let res = await getCall("/admin/getAllUsers", {offset: (this.page-1)* 50})
		this.numPages = res.data.length / 50 + (res.data.length % 50 > 0)
		this.setState(() => ({users: res.data}))
	}

	render({lister}, state) {
		if(lister === undefined) {
			lister = UserTable
		}
		return (
			<div>
				<Paginator page={this.page} numPages={this.numPages} setPage={this.setPage}/>
				<this.lister users={state.users}/>
			</div>
		)
	}
}

register(UserList, 'kn-user-list', [])
