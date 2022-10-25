import {h, Fragment, Component, render} from 'preact';
import register from 'preact-custom-element';
import {getText} from '../translation';

class ReplaceMe extends Component {

	render() {
		return (
			<></>
		)
	}
}

register(ReplaceMe, 'kn-replace-me', [])
