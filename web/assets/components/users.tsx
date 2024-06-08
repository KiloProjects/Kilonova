import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import { useEffect, useState } from "preact/hooks";
import { getCall } from "../api/client";
import { apiToast } from "../toast";
import { Paginator } from "./common";
import getText from "../translation";

function UserTable({ users }: { users: UserBrief[] }) {
	if (users.length == 0) {
		return <div class="text-4xl mx-auto my-auto w-full mt-10 mb-10 text-center">{getText("noUserFound")}</div>;
	}
	return (
		<table class="kn-table">
			<thead>
				<tr>
					<th class="kn-table-cell w-1/12" scope="col">
						{getText("id")}
					</th>
					<th class="kn-table-cell" scope="col">
						User
					</th>
					<th class="kn-table-cell" scope="col">
						{getText("admin")}
					</th>
					<th class="kn-table-cell" scope="col">
						{getText("proposer")}
					</th>
				</tr>
			</thead>
			<tbody>
				{users.map((user) => (
					<tr class="kn-table-row" key={user.id}>
						<td class="kn-table-cell">{user.id}</td>
						<td class="kn-table-cell">
							<a href={`/profile/${user.name}`} class="inline-flex align-middle items-center">
								<img width={32} height={32} class="flex-none mr-2 rounded" src={`/api/user/byName/${user.name}/avatar?s=32`} /> {user.name}
							</a>
						</td>
						<td class="kn-table-cell">{user.admin ? <span class="text-lg">✅</span> : <span class="text-lg">❌</span>}</td>
						<td class="kn-table-cell">{user.proposer ? <span class="text-lg">✅</span> : <span class="text-lg">❌</span>}</td>
					</tr>
				))}
			</tbody>
		</table>
	);
}

function UserList() {
	let [users, setUsers] = useState<UserBrief[]>([]);
	let [page, setPage] = useState<number>(1);
	let [numPages, setNumPages] = useState<number>(1);
	let [name, setName] = useState("");

	async function poll() {
		let res = await getCall("/admin/getAllUsers", { offset: 50 * (page - 1), limit: 50, name_fuzzy: name.length > 0 ? name : undefined });
		if (res.status !== "success") {
			apiToast(res);
			throw new Error("Couldn't fetch users");
		}
		setUsers(res.data.users);
		setNumPages(Math.floor(res.data.total_count / 50) + (res.data.total_count % 50 != 0 ? 1 : 0));
	}

	function updateName(newName: string) {
		setName(newName);
		setPage(1);
	}

	useEffect(() => {
		poll().catch(console.error);
	}, [page, name]);

	return (
		<>
			<h1>{getText("users")}</h1>
			<label class="block my-2">
				<span class="form-label">{getText("name")}: </span>
				<input
					class="form-input"
					type="text"
					onInput={(e) => {
						updateName(e.currentTarget.value);
					}}
					value={name}
				/>
			</label>
			<Paginator numpages={numPages} page={page} setPage={setPage} showArrows={true} />
			<UserTable users={users} />
		</>
	);
}

register(UserList, "kn-user-list", []);
