import {h, Fragment, Component, render} from 'preact';
import register from 'preact-custom-element';
import {getText} from '../translation';

class Contests extends Component {
	state = {
		currentContests: [],
		futureContests: [],
	}
	constructor() {
	}

	getContests() {
		
	}

	render() {
		return (
			<></>
		)
	}
}

register(Contests, 'kn-contests', [])
