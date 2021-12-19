import {h, Fragment, Component} from 'preact';
import register from 'preact-custom-element';
import {getText} from '../translation.js';

interface PaginatorParams {
	page: number,
	numPages: number,
	setPage: (num: number) => void,
}

export function Paginator({page, numPages, setPage}: PaginatorParams) {
	// props format: {page: number, setPage(): handler for updating, numPages: number}
	let elements: preact.JSX.Element[] = [];
	if(page != 1) {
		elements.push(<button class="btn-gray" onClick={() => setPage(1)}><i class="fas fa-angle-double-left"></i></button>);
		elements.push(<button class="btn-gray" onClick={() => setPage(page-1)}><i class="fas fa-angle-left"></i></button>);
	}
	if(page != numPages) {
		elements.push(<button class="btn-gray" onClick={() => setPage(numPages)}><i class="fas fa-angle-double-right"></i></button>);
		elements.push(<button class="btn-gray" onClick={() => setPage(page+1)}><i class="fas fa-angle-right"></i></button>);
	}
	if(page > 3) {
		for(let i of [1, 2, 3]) {
			elements.push(<button class="btn-gray" onClick={() => setPage(i)}>{i}</button>);
		}
		elements.push(<button class="btn-gray">...</button>);
	}

	elements.push(<button class="btn-gray" onClick={() => setPage(page-2)}>{page-2}</button>);
	elements.push(<button class="btn-gray" onClick={() => setPage(page-1)}>{page-1}</button>);
	elements.push(<button class="btn-gray">{page}</button>);
	elements.push(<button class="btn-gray" onClick={() => setPage(page+1)}>{page+1}</button>);
	elements.push(<button class="btn-gray" onClick={() => setPage(page+2)}>{page+2}</button>);

	if(numPages - page > 3) {
		elements.push(<button class="btn-gray">...</button>);
		for(let i = numPages - 3; i <= numPages; i++) {
			elements.push(<button class="btn-gray" onClick={() => setPage(i)}>{i}</button>);
		}
	}
	return <div class="mt-2 mb-2 inline-flex">{elements}</div>;
}

import {useState} from 'preact/hooks';

export function PaginateTester() {
	let [pg, setPg] = useState(0);
	let [maxPg, setMaxPg] = useState(0);
	return (
		<div>
			<input type="number" value={pg} onChange={(e) => {setPg(Number.parseInt((e.target as HTMLInputElement).value))}}/>
		</div>
	);
}

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
