    const problem_id = __PBID__;

    async function addEditor() {
        const username = document.getElementById('userToAdd').value;
        let res = await bundled.postCall(`/problem/${problem_id}/update/addEditor`, { username: username })
        bundled.apiToast(res)
        await loadView();
    }

    async function addViewer() {
        const username = document.getElementById('userToAdd').value;
        let res = await bundled.postCall(`/problem/${problem_id}/update/addViewer`, { username: username })
        bundled.apiToast(res)
        await loadView();
    }

    async function stripAccess(id) {
        let res = await bundled.postCall(`/problem/${problem_id}/update/stripAccess`, { user_id: id })
        bundled.apiToast(res)
        await loadView();
    }

    async function loadProblem() {
        const res = await bundled.getCall(`/problem/${problem_id}/get/accessControl`, {});
        if (res.status === "error") {
            bundled.apiToast(res)
            return
        }
        return res.data
    }

    async function loadUsers(users, id) {
        console.log(id, document.getElementById(id))
        //document.getElementById(id).innerHTML = `<div class="list-group-head">${bundled.getText("loading")}</div>`
        let outhtml = ""
        if (users.length == 0) {
            outhtml += `<div class="list-group-head">No users</div>`
        }
        for (let user of users) {
            outhtml += `
            <div class="list-group-item flex justify-between items-center">
                <a href="/profile/${user.name}">
                    <img class="rounded-sm inline-block mr-2" src="/profile/${user.name}/avatar?s=32" width="32" height="32" alt="Avatar"/>
                    ${user.name}
                </a>
                ${user.id != window.platform_info.user_id ? `<span style="cursor: pointer;" onclick="stripAccess(${user.id})">X</span>` : ''}
            </div>`
        }
        document.getElementById(id).innerHTML = outhtml
    }

    async function loadView() {
        const data = await loadProblem();
        loadUsers(data.editors, 'editors_list')
        loadUsers(data.viewers, 'viewers_list')
    }

    document.addEventListener("DOMContentLoaded", () => loadView());

