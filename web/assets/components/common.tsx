import {h, Fragment, Component} from 'preact';
import register from 'preact-custom-element';
import {getText} from '../translation.js';

export function BigSpinner() {
	return (
		<div class="text-4xl mx-auto my-auto w-full my-10 text-center">
			<div><i class="fas fa-spinner animate-spin"></i> {getText("loading")}</div>
		</div>
	)
}

export function SmallSpinner() {
	return (
		<div class="mx-auto my-auto w-full text-center">
			<div><i class="fas fa-spinner animate-spin"></i> {getText("loading")}</div>
		</div>
	)
}

export function Segment(props) {
	return (
		<div className="segment-container">
			{props.children}
		</div>
	);
}

export function Button(props) {
	return (
		<button className="btn btn-blue">
			{props.children}
		</button>
	);
}

export function RedButton(props) {
	return (
		<button className="btn btn-red">
			{props.children}
		</button>
	);
}

export function ProblemAttachment({attname = ""}) {
	let pname = window.location.pathname;
	if(pname.endsWith('/')) {
		console.log("cf")
		pname = pname.substr(0, pname.lastIndexOf('/'));
	}
	return <img src={`${pname}/attachments/${attname}`}/>;
}

register(ProblemAttachment, 'problem-attachment', ['attname'])
