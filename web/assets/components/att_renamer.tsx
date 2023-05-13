import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import getText from "../translation";
import { useState } from "preact/hooks";
import { bodyCall } from "../api/net";
import { apiToast } from "../toast";

export function AttachmentRenamer({ pbid, attid, orgname }: { pbid: string; attid: string; orgname: string }) {
	let [editing, setEditing] = useState<boolean>(false);
	let [name, setName] = useState<string>(orgname);
	let [preEditName, setPreEditName] = useState<string>(orgname);

	async function updateName() {
		const rez = await bodyCall<string>(`/problem/${pbid}/update/bulkUpdateAttachmentInfo`, {
			[attid]: { name },
		});
		apiToast(rez);
		if (rez.status === "success") {
			setEditing(false);
		}
	}

	if (!editing) {
		return (
			<>
				<a href={`/problems/${pbid}/attachments/${name}`}>{name}</a>{" "}
				<span
					onClick={() => {
						setEditing(true);
						setPreEditName(name);
					}}
				>
					<i class="fas fa-pencil"></i> {getText("edit")}
				</span>
			</>
		);
	}

	return (
		<>
			<button
				class="btn btn-blue"
				onClick={() => {
					setName(preEditName);
					setEditing(false);
				}}
			>
				X
			</button>
			<input class="form-input mx-2" type="text" value={name} onChange={(e) => setName(e.currentTarget.value)} />
			<button class="btn btn-blue" onClick={() => updateName().catch(console.error)}>
				{getText("button.update")}
			</button>
		</>
	);
}

register(AttachmentRenamer, "kn-att-name", ["pbid", "attid", "orgname"]);
