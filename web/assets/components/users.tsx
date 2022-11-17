import { h, Fragment, Component } from "preact";
import register from "preact-custom-element";
import { useCallback, useEffect, useMemo, useState } from "preact/hooks";
import { getCall } from "../net";
import { apiToast } from "../toast";
import { Paginator } from "./common";

type User = {
	name: string;
	id: number;
};

function UserTable({ users }: { users: User[] }) {
	return (
		<div class="list-group list-group-mini">
			<div class="list-group-head font-bold">User</div>
			{users.map((user) => (
				<a href={`/profile/${user.name}`} class="list-group-item inline-flex align-middle items-center">
					<img class="flex-none mr-2 rounded" src={`/api/user/getGravatar?name=${user.name}&s=32`} /> #{user.id}: {user.name}
				</a>
			))}
		</div>
	);
}

function UserList() {
	let [users, setUsers] = useState<User[]>([]);

	let poll = useCallback(async () => {
		let res = await getCall("/admin/getAllUsers", {});
		if (res.status !== "success") {
			apiToast(res);
			throw new Error("Couldn't fetch users");
		}
		setUsers(res.data);
	}, []);

	useEffect(() => {
		poll().catch(console.error);
	}, []);

	return (
		<div class="my-4">
			<UserTable users={users} />
		</div>
	);
}

register(UserList, "kn-user-list", []);
