import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import { useEffect, useState } from "preact/hooks";
import { getCall } from "../api/net";
import { apiToast } from "../toast";
import { Paginator } from "./common";
import getText from "../translation";
import { UserBrief } from "../api/submissions";

function UserTable({ users }: { users: UserBrief[] }) {
	if (users.length == 0) {
		return (
			<div class="list-group">
				<div class="list-group-head font-bold">User</div>
				<div class="list-group-item">{getText("no_users")}</div>
			</div>
		);
	}
	return (
		<div class="list-group">
			<div class="list-group-head font-bold">User</div>
			{users.map((user) => (
				<a href={`/profile/${user.name}`} class="list-group-item inline-flex align-middle items-center" key={user.id}>
					<img class="flex-none mr-2 rounded" src={`/api/user/getGravatar?name=${user.name}&s=32`} /> #{user.id}: {user.name}
				</a>
			))}
		</div>
	);
}

function UserList() {
	let [users, setUsers] = useState<UserBrief[]>([]);
	let [page, setPage] = useState<number>(1);
	let [numPages, setNumPages] = useState<number>(1);

	async function poll() {
		let res = await getCall("/admin/getAllUsers", { offset: 50 * (page - 1), limit: 50 });
		if (res.status !== "success") {
			apiToast(res);
			throw new Error("Couldn't fetch users");
		}
		setUsers(res.data.users);
		setNumPages(Math.floor(res.data.total_count / 50) + (res.data.total_count % 50 != 0 ? 1 : 0));
	}

	useEffect(() => {
		poll().catch(console.error);
	}, [page]);

	return (
		<div class="my-4">
			<Paginator numpages={numPages} page={page} setPage={setPage} showArrows={true} />
			<UserTable users={users} />
		</div>
	);
}

register(UserList, "kn-user-list", []);
