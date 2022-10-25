import {h, Fragment, Component} from 'preact';
import register from 'preact-custom-element';
import {getText} from '../translation.js';

interface PaginatorParams {
	page: number,
	numPages: number,
	setPage: (num: number) => void,
}

import {apiToast} from '../toast';
export function Paginator({page, numPages, setPage}: PaginatorParams) {
	// props format: {page: number, setPage(): handler for updating, numPages: number}
	if(page < 1 || page > numPages) {
		console.warn("Invalid page number");
		apiToast({status: "error", data: "Invalid page number"});
	}
	let elements: preact.JSX.Element[] = [];
	const old_sp = setPage;
	setPage = (pg) => {
		if(pg < 1) {
			pg = 1;
		}
		if(pg > numPages) {
			pg = numPages;
		}
		old_sp(pg);
	}

	//elements.push(<button class="paginator-item" onClick={() => setPage(1)}><i class="fas fa-angle-double-left"></i></button>);
	//elements.push(<button class="paginator-item" onClick={() => setPage(page-1)}><i class="fas fa-angle-left"></i></button>);
	if(page > 3) {
		for(let i = 1; i <= 3 && page - i >= 3; i++) {
			elements.push(<button class="paginator-item" onClick={() => setPage(i)}>{i}</button>);
		}
		if(page > 6) {
			elements.push(<button class="paginator-item">...</button>);
		}
	}
	
	if(page-2>0) {
		elements.push(<button class="paginator-item" onClick={() => setPage(page-2)}>{page-2}</button>);
	}
	if(page-1>0) {
		elements.push(<button class="paginator-item" onClick={() => setPage(page-1)}>{page-1}</button>);
	}
	elements.push(<button class="paginator-item paginator-item-active">{page}</button>);
	if(page+1<=numPages) {
		elements.push(<button class="paginator-item" onClick={() => setPage(page+1)}>{page+1}</button>);
	}
	if(page+2<=numPages) {
		elements.push(<button class="paginator-item" onClick={() => setPage(page+2)}>{page+2}</button>);
	}

	if(numPages - page >= 3) {
		if(numPages - page > 5) {
			elements.push(<button class="paginator-item">...</button>);
		}
		for(let i = numPages - 2; i <= numPages; i++) {
			if(i - page <= 2) {
				continue;
			}
			elements.push(<button class="paginator-item" onClick={() => setPage(i)}>{i}</button>);
		}
	}

	//elements.push(<button class="paginator-item" onClick={() => setPage(page+1)}><i class="fas fa-angle-right"></i></button>);
	//elements.push(<button class="paginator-item" onClick={() => setPage(numPages)}><i class="fas fa-angle-double-right"></i></button>);
	return <div class="paginator">{elements}</div>;
}

import {useState} from 'preact/hooks';

export function PaginateTester() {
	let [pg, setPg] = useState(1);
	let [maxPg, setMaxPg] = useState(5);
	return (
		<div>
			<input type="number" class="form-input" value={pg} onChange={(e) => {setPg(Number.parseInt((e.target as HTMLInputElement).value))}}/><br/>
			<input type="number" class="form-input" value={maxPg} onChange={(e) => {setMaxPg(Number.parseInt((e.target as HTMLInputElement).value))}}/><br/><br/>
			<Paginator page={pg} numPages={maxPg} setPage={(pg) => setPg(pg)}/>
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
