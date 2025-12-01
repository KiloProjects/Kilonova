import {Fragment, h, render} from "preact";
import {useEffect, useRef, useState} from "preact/hooks";
import getText from "../translation";

export function KNModal({
	                        open,
	                        title,
	                        children,
	                        footer,
	                        closeCallback,
	                        large = true,
                        }: {
	open: boolean;
	title: any;
	children?: preact.ComponentChildren;
	footer?: preact.ComponentChildren;
	closeCallback?: () => void;
	large?: boolean;
}) {
	let [lastState, setLastState] = useState<boolean | null>(null);
	let ref = useRef<HTMLDialogElement>(null);

	useEffect(() => {
		if (open == lastState) {
			return;
		}

		if (open) {
			ref.current?.showModal();
		} else {
			ref.current?.close();
		}

		setLastState(open);
		return () => {
			ref.current?.close();
		};
	}, [open]);

	return (
		<dialog onClose={closeCallback} onCancel={closeCallback} ref={ref}
		        class={`modal-container ${large && "modal-container-large"}`}>
			<div class="modal-header">
				<h1>{title}</h1>
				<form method="dialog" onSubmit={closeCallback}>
					<button type="submit">
						<i class="modal-close fas fa-xmark"></i>
					</button>
				</form>
			</div>
			<div class="modal-content">{children}</div>
			{typeof footer !== "undefined" && <div class="modal-footer">{footer}</div>}
		</dialog>
	);
}

export function confirm(message: string): Promise<boolean> {
	return new Promise((resolve) => {
		const par = document.createElement("div");
		document.getElementById("modals")!.append(par);

		function callback(val: boolean) {
			par.parentElement?.removeChild(par);
			resolve(val);
		}

		render(
			<KNModal
				title={getText("confirm_header")}
				open={true}
				closeCallback={() => callback(false)}
				footer={
					<>
						<button onClick={() => callback(false)} class="btn mr-2">
							{getText("button.cancel")}
						</button>
						<button onClick={() => callback(true)} class="btn btn-blue">
							OK
						</button>
					</>
				}
				large={false}
			>
				<p class="my-2">{message}</p>
			</KNModal>,
			par
		);
	});
}

export function inviteQRCode(inviteID: string) {
	const par = document.createElement("div");
	document.getElementById("modals")!.append(par);

	function callback() {
		par.parentElement?.removeChild(par)
	}

	render(
		<KNModal open={true} title={getText("invite_qr_code")}
		         closeCallback={() => callback()}
		         footer={
			         <button onClick={() => callback()} class="btn btn-blue mr-2">
				         {getText("button.close")}
			         </button>
		         }>
			<img class="object-contain mx-auto max-w-full max-h-full" src={`/contests/invite/${inviteID}/qr`} alt="QR Code for contest invitation"/>
		</KNModal>,
		par);
}


document.addEventListener("htmx:confirm", e => {
	if (!e.target?.hasAttribute("hx-confirm")) return

	e.preventDefault();
	confirm(e.detail.question).then(ok => {
		if (ok) e.detail.issueRequest(true);
	})
})

document.addEventListener("DOMContentLoaded", e => {
	const observer = new MutationObserver(records => {
		for (let record of records) {
			for (let node of record.addedNodes) {
				if (!(node instanceof Element)) {
					continue
				}

				if (node instanceof HTMLDialogElement) {
					node.showModal();
				}

				for (let dNode of node.querySelectorAll("dialog")) {
					dNode.showModal();
				}
			}
		}
	})
	const modals = document.getElementById("modals")
	if (modals != null) {
		observer.observe(modals, {childList: true})
	}
})

