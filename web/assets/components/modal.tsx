import { h, Fragment, render } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";

export function KNModal({
	open,
	title,
	children,
	closeCallback,
	large = true,
}: {
	open: boolean;
	title: any;
	children?: preact.ComponentChildren;
	closeCallback?: () => void;
	large?: boolean;
}) {
	let [lastState, setLastState] = useState<boolean | null>(null);
	let ref = useRef<HTMLDialogElement>(null);

	ref.current?.addEventListener;

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
		<dialog onClose={closeCallback} onCancel={closeCallback} ref={ref} class={`modal-container ${large && "modal-container-large"}`} id="max_score_dialog">
			<div class="modal-header">
				<h1>{title}</h1>
				<form method="dialog" onSubmit={closeCallback}>
					<button type="submit">
						<i class="modal-close"></i>
					</button>
				</form>
			</div>
			<div class="modal-content" id="max_score_content">
				{children}
			</div>
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
			<KNModal title={getText("confirm_header")} open={true} closeCallback={() => callback(false)} large={false}>
				<p class="my-2">{message}</p>
				<button onClick={() => callback(false)} class="btn btn-blue mr-2">
					{getText("button.cancel")}
				</button>
				<button onClick={() => callback(true)} class="btn btn-red">
					OK
				</button>
			</KNModal>,
			par
		);
	});
}
