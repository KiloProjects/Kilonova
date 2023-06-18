import { h, Fragment, render } from "preact";
import { useCallback, useEffect, useRef, useState } from "preact/hooks";
import register from "preact-custom-element";
import getText from "../translation";
import { apiToast } from "../toast";
import { bodyCall, getCall, postCall } from "../api/net";
import { fromBase64 } from "js-base64";
import { KNModal } from "./modal";
import { BigSpinner } from "./common";

export type TagType = "author" | "contest" | "method" | "other";

export type Tag = {
	id: number;
	name: string;
	type: TagType;
};

const tagTypes: Record<TagType, string> = {
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
	const classes = `${wide ? "mx-1" : ""} ${typeof onClick !== "undefined" ? "cursor-pointer" : ""} ${tagClasses(tag)} ${extraClasses || ""}`;
	if (!link) {
		return (
			<span class={classes} onClick={onClick}>
				{tag.name}
			</span>
		);
	}
	return (
		<a onClick={onClick} href={`/tags/${tag.id}`}>
			<span class={classes}>{tag.name}</span>
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
		if (text == "" || text == getText("tag_name")) {
			apiToast({ status: "error", data: "Name must not be empty" });
			return;
		}
		let res = await postCall<number>("/tags/create", { name: text, type });
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		cb(res.data);
		setCreateMode(false);
	}

	if (!createMode) {
		return (
			<a
				href="#"
				class="mx-1"
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
		<span class={"mx-1 " + tagClasses({ type })}>
			<span
				ref={ref2}
				class="tag-editable"
				contentEditable={true}
				onKeyDown={(e) => {
					if (e.key === "Enter" || e.keyCode == 13) {
						e.preventDefault();
					}
				}}
				onInput={(e) => {
					setText(e.currentTarget.innerText);
				}}
				data-bf={getText("tag_name")}
			></span>
			{text !== "" && text !== getText("tag_name") && <i class="ml-2 cursor-pointer fas fa-check" onClick={createTag}></i>}
			<i
				class="ml-2 cursor-pointer fas fa-xmark"
				onClick={() => {
					setCreateMode(false);
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
	let [newTags, setTags] = useState(tags);

	async function loadProblemTags() {
		let res = await getCall<Tag[]>(`/problem/${problemID}/tags`, {});
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		setTags(res.data);
	}

	async function updateTags(tags: Tag[]) {
		let res = await bodyCall(`/problem/${problemID}/update/tags`, { tags: tags.map((t) => t.id) });
		if (res.status === "error") {
			apiToast(res);
			return;
		}
		apiToast(res);
		await loadProblemTags();
	}

	return (
		<>
			{newTags.map((tag) => (
				<TagView tag={tag} key={tag.id}></TagView>
			))}
			<a
				class="mx-1"
				href="#"
				onClickCapture={(e) => {
					e.preventDefault();
					selectTags(newTags, true, "update_tags").then((rez) => {
						if (rez.updated) updateTags(rez.tags);
					});
				}}
			>
				<i class="fas fa-pen-to-square"></i> {newTags.length === 0 ? getText("add_tags") : getText("update_tags")}
			</a>
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

export function selectTags(initialSelected: Tag[], canCreate: boolean = false, titleKey: string = "select_tags"): Promise<{ tags: Tag[]; updated: boolean }> {
	return new Promise((resolve) => {
		const par = document.createElement("div");
		document.getElementById("modals")!.append(par);

		function callback(val: Tag[], cancel: boolean) {
			par.parentElement?.removeChild(par);
			if (cancel) {
				resolve({ tags: initialSelected, updated: false });
				return;
			}
			resolve({ tags: val, updated: true });
		}

		function Selector({ initialTags }: { initialTags: Tag[] }) {
			let [tags, setTags] = useState<Tag[]>(initialTags);
			let [tagList, updateTagList] = useState<Tag[]>([]);
			let [loading, setLoading] = useState<boolean>(true);
			let [searchVal, setSearchVal] = useState<string>("");

			function searchStrFilter(t: Tag): boolean {
				return cleanupTagSearchTerm(t.name).includes(cleanupTagSearchTerm(searchVal));
			}

			function updTags(newTags: Tag[]) {
				newTags.sort((a, b) => {
					if (a.type > b.type) {
						return 1;
					}
					if (a.type < b.type) {
						return -1;
					}
					if (a.name > b.name) {
						return 1;
					}
					if (a.name < b.name) {
						return -1;
					}
					return 0;
				});
				setTags(newTags);
			}

			async function load(insertFromID?: number) {
				let res = await getCall<Tag[]>(`/tags/`, {});
				if (res.status === "error") {
					apiToast(res);
					return;
				}
				updateTagList(res.data);
				if (typeof insertFromID !== "undefined") {
					updTags([...tags, ...res.data.filter((t) => t.id == insertFromID)]);
				}
				setLoading(false);
			}

			useEffect(() => {
				load().catch(console.error);
			}, []);

			return (
				<KNModal
					title={getText(titleKey)}
					open={true}
					closeCallback={() => callback(tags, true)}
					footer={
						<>
							<button class="btn my-2 mr-2" onClick={() => callback(tags, true)}>
								{getText("button.cancel")}
							</button>
							<button class="btn btn-blue my-2" onClick={() => callback(tags, false)}>
								{getText("button.select")}
							</button>
						</>
					}
				>
					{loading ? (
						<BigSpinner />
					) : (
						<div class="overflow-hidden">
							<div class="segment-panel">
								{tags.length > 0 ? (
									<span>
										{getText("selected_tags")}:{" "}
										{tags.map((tag) => (
											<TagView
												tag={tag}
												key={tag.id}
												link={false}
												onClick={() => {
													updTags(tags.filter((val) => val.id != tag.id));
												}}
											/>
										))}
									</span>
								) : (
									<span>{getText("no_selected_tags")}</span>
								)}
							</div>
							<input
								type="input"
								class="form-input block my-2"
								placeholder={getText("search_tag")}
								onInput={(e) => {
									setSearchVal(e.currentTarget.value);
								}}
								value={searchVal}
							></input>
							<div class="overflow-y-auto">
								{(Object.keys(tagTypes) as TagType[]).map((tp) => {
									const typeTags = tagList
										.filter((t) => t.type == tp && typeof tags.find((val) => val.id == t.id) === "undefined")
										.filter(searchStrFilter);
									return (
										<details key={tp} class="block my-2" open>
											<summary>
												<h3 class="inline-block mb-2">{getText(`tag_names.${tp}`)}</h3>
											</summary>
											{typeTags.map((tag) => (
												<TagView
													tag={tag}
													link={false}
													onClick={() => {
														updTags([...tags, tag]);
														setSearchVal("");
													}}
													key={tag.id}
												/>
											))}
											{typeTags.length === 0 && <span>{getText("no_tags")}</span>}
											{canCreate && searchVal.length == 0 && (
												<TagQuickAddView type={tp} cb={(val: number) => load(val).catch(console.error)} />
											)}
										</details>
									);
								})}
							</div>
						</div>
					)}
				</KNModal>
			);
		}

		render(<Selector initialTags={initialSelected}></Selector>, par);
	});
}
