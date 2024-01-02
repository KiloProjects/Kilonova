import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import getText from "../translation";
import { useState } from "preact/hooks";
import { bodyCall } from "../api/client";
import { apiToast } from "../toast";

export function AttachmentRenamer({
	pbid,
	postid,
	postslug,
	attid,
	orgname,
	editable,
}: {
	pbid: string;
	postid: string;
	postslug: string;
	attid: string;
	orgname: string;
	editable: string;
}) {
	let [editing, setEditing] = useState<boolean>(false);
	let [name, setName] = useState<string>(orgname);
	let [preEditName, setPreEditName] = useState<string>(orgname);

	let apiPrefix = "",
		urlPrefix = "";
	if (pbid?.length > 0) {
		apiPrefix = `/problem/${pbid}`;
		urlPrefix = `/assets/problem/${pbid}`;
	} else if (postid?.length > 0) {
		apiPrefix = `/blogPosts/${postid}`;
		urlPrefix = `/assets/blogPost/${postslug}`;
	} else {
		throw new Error("problem id or post id not specified");
	}
	console.log(pbid, postid, postslug, apiPrefix, urlPrefix);

	async function updateName() {
		const rez = await bodyCall<string>(apiPrefix + "/update/bulkUpdateAttachmentInfo", {
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
				<a href={urlPrefix + `/attachment/${name}`}>{name}</a>
				{editable == "true" && (
					<>
						{" "}
						<span
							onClick={() => {
								setEditing(true);
								setPreEditName(name);
							}}
						>
							<i class="fas fa-pencil"></i> {getText("edit")}
						</span>
					</>
				)}
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

register(AttachmentRenamer, "kn-att-name", ["pbid", "postid", "postslug", "attid", "orgname", "editable"]);
