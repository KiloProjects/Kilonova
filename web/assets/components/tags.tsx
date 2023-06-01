import { h, Fragment, render } from "preact";
import { useCallback, useEffect, useRef, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";
import { apiToast } from "../toast";
import { bodyCall, getCall, postCall } from "../api/net";
import { fromBase64 } from "js-base64";
import { KNModal } from "./modal";
import { BigSpinner } from "./common";
import { throttle } from "lodash-es";

export type TagType = "author" | "contest" | "method" | "other";

export type Tag = {
	id: number;
	name: string;
	type: TagType;
};

const tagTypes = {
	author: "tag-author",
	contest: "tag-contest",
	method: "tag-method",
	other: "tag-other",
};

function tagClasses(tag: { type: TagType }): string {
	return `tag ${tagTypes[tag.type] ?? "bg-teal-700"}`;
}

export function TagView({
	tag,
	link = true,
	onClick,
	wide = true,
	extraClasses,
}: {
	tag: Tag;
	link?: boolean;
	onClick?: any;
	wide?: boolean;
	extraClasses?: string;
}) {
	if (!link) {
		return <span class={(wide ? "mx-1 " : "") + tagClasses(tag) + " " + extraClasses}>{tag.name}</span>;
	}
	return (
		<a onClick={onClick} href={`/tags/${tag.id}`}>
			<span class={(wide ? "mx-1 " : "") + tagClasses(tag) + " " + extraClasses}>{tag.name}</span>
		</a>
	);
}

function TagViewDOM({ enc, link, wide, cls }: { enc: string; link: string; wide: string; cls: string }) {
	const tag: Tag = JSON.parse(fromBase64(enc));
	return <TagView tag={tag} link={link !== "false"} wide={wide !== "false"} extraClasses={cls} />;
}

register(TagViewDOM, "kn-tag", ["enc", "link", "wide", "cls"]);

function TagQuickAddView({ type, cb }: { type: TagType; cb: (number) => any }) {
	let [createMode, setCreateMode] = useState(false);
	let [text, setText] = useState("");

	let ref2 = useCallback((node) => {
		if (node !== null) {
			node.focus();
		}
	}, []);

	async function createTag() {
		console.log(text);
		let res = await postCall<number>("/tags/create", { name: text, type });
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		cb(res.data);
		setCreateMode(false);
		setText("");
	}

	if (!createMode) {
		return (
			<a
				href="#"
				onClick={(e) => {
					e.preventDefault();
					setCreateMode(true);
				}}
			>
				<span class={"cursor-pointer " + tagClasses({ type })}>
					<i class="fas fa-plus"></i> {getText("button.add")}
				</span>
			</a>
		);
	}
	return (
		<span class={tagClasses({ type })}>
			<span
				ref={ref2}
				class="tag-editable"
				contentEditable={true}
				onKeyDown={(e) => {
					if (e.key === "Enter" || e.keyCode == 13) {
						e.preventDefault();
					} else {
						setText(e.currentTarget.innerText);
					}
				}}
				data-bf={getText("tag_name")}
			></span>
			{text !== "" && <i class="ml-2 cursor-pointer fas fa-check" onClick={createTag}></i>}
			<i
				class="ml-2 cursor-pointer fas fa-xmark"
				onClick={() => {
					setCreateMode(false);
					setText("");
				}}
			></i>
		</span>
	);
}

export function cleanupTagSearchTerm(val: string): string {
	return val
		.normalize("NFD")
		.replace(/[\u0300-\u036f]/g, "")
		.toLowerCase();
}

function ProblemTagEdit({ tags, problemID }: { tags: Tag[]; problemID: number }) {
	let [open, setOpen] = useState(false);
	let [newTags, setTags] = useState(tags);
	let [allTags, setAllTags] = useState<Tag[] | null>(null);
	let [checkedTags, setCheckedTags] = useState<number[]>(tags.map((t) => t.id));
	let [searchVal, setSearchVal] = useState<string>("");

	async function loadProblemTags() {
		let res = await getCall<Tag[]>(`/problem/${problemID}/tags`, {});
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		setTags(res.data);
	}

	async function loadTags() {
		let res = await getCall<Tag[]>(`/tags/`, "");
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		setAllTags(res.data);
	}

	function searchStrFilter(t: Tag): boolean {
		return cleanupTagSearchTerm(t.name).includes(cleanupTagSearchTerm(searchVal));
	}

	async function updateTags(e: Event) {
		e.preventDefault();
		let res = await bodyCall(`/problem/${problemID}/update/tags`, { tags: checkedTags });
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		apiToast(res);
		await loadProblemTags();
		setOpen(false);
	}

	useEffect(() => {
		if (open === true) {
			setSearchVal("");
			setCheckedTags(newTags.map((t) => t.id));
			loadTags().catch(console.error);
		} else {
			loadProblemTags().catch(console.error);
		}
	}, [open]);

	return (
		<>
			{newTags.map((tag) => (
				<TagView tag={tag} key={tag.id}></TagView>
			))}
			<a
				class="mx-1"
				href="#"
				onClickCapture={(e) => {
					e.preventDefault(), setOpen(true);
				}}
			>
				<i class="fas fa-pen-to-square"></i> {newTags.length === 0 ? getText("add_tags") : getText("update_tags")}
			</a>
			<KNModal title={getText("update_tags")} open={open} closeCallback={() => setOpen(false)}>
				{allTags == null ? (
					<BigSpinner />
				) : (
					<div>
						<input
							type="input"
							class={"form-input"}
							placeholder={getText("search_tag")}
							onInput={(e) => {
								setSearchVal(e.currentTarget.value);
							}}
							value={searchVal}
						></input>
						{(["author", "contest", "method", "other"] as TagType[]).map((tp) => {
							const tags = allTags!.filter((t) => t.type == tp).filter(searchStrFilter);
							return (
								<details key={tp} class="block my-2" open>
									<summary>
										<h3 class="inline-block mb-2">{getText(`tag_names.${tp}`)}</h3>
									</summary>
									{tags.map((tag) => (
										<label class="mx-1" key={tag.id}>
											<input
												type="checkbox"
												class="form-checkbox"
												checked={checkedTags.includes(tag.id)}
												onChange={() => {
													if (checkedTags.includes(tag.id)) {
														setCheckedTags(checkedTags.filter((t) => t != tag.id));
													} else {
														setCheckedTags([...checkedTags, tag.id]);
													}
												}}
											/>
											<TagView tag={tag} link={false} onClick={() => console.log(tag)} />
										</label>
									))}
									{tags.length === 0 && <span>{getText("no_tags")}</span>}
									{open && searchVal.length == 0 && (
										<TagQuickAddView
											type={tp}
											cb={(val) => {
												setCheckedTags([...checkedTags, val]);
												loadTags().catch(console.error);
											}}
										/>
									)}
								</details>
							);
						})}
						<button class="btn btn-blue my-2" onClick={updateTags}>
							{getText("button.update")}
						</button>
					</div>
				)}
			</KNModal>
		</>
	);
}

function ProblemTagEditDOM({ enc, pbid }: { enc: string; pbid: string }) {
	const tags: Tag[] = JSON.parse(fromBase64(enc));
	const problemID = parseInt(pbid);
	if (isNaN(problemID)) {
		throw new Error("Invalid problem ID");
	}
	return <ProblemTagEdit tags={tags} problemID={problemID}></ProblemTagEdit>;
}

register(ProblemTagEditDOM, "kn-pb-tag-edit", ["enc"]);

function ProblemTags({ tags, defaultOpen }: { tags: Tag[]; defaultOpen: boolean }) {
	let [open, setOpen] = useState(defaultOpen);

	if (!open) {
		return (
			<a
				href="#"
				onClick={(e) => {
					e.preventDefault();
					setOpen(true);
				}}
			>
				{getText("view")}
			</a>
		);
	}

	return (
		<>
			{tags.map((tag) => (
				<>
					<TagView tag={tag} key={tag.id} wide={false} extraClasses="text-sm" />{" "}
				</>
			))}
			<a
				href="#"
				onClick={(e) => {
					e.preventDefault();
					setOpen(false);
				}}
			>
				{getText("hide")}
			</a>
		</>
	);
}

function ProblemTagsDOM({ enc, open }: { enc: string; open: string }) {
	const tags: Tag[] = JSON.parse(fromBase64(enc));
	return <ProblemTags tags={tags} defaultOpen={open !== "false"}></ProblemTags>;
}

register(ProblemTagsDOM, "kn-pb-tags", ["enc", "open"]);
