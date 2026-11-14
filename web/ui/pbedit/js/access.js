    const problem_id = __PBID__;

    function refreshAccess() {
        return htmx.ajax("GET", location.href, {target: "#access_lists", select: "#access_lists", swap: "morph"})
    }

    async function addEditor() {
        const username = document.getElementById('userToAdd').value;
        let res = await bundled.postCall(`/problem/${problem_id}/update/addEditor`, { username: username })
        bundled.apiToast(res)
        await refreshAccess();
    }

    async function addViewer() {
        const username = document.getElementById('userToAdd').value;
        let res = await bundled.postCall(`/problem/${problem_id}/update/addViewer`, { username: username })
        bundled.apiToast(res)
        await refreshAccess();
    }

    async function stripAccess(id) {
        let res = await bundled.postCall(`/problem/${problem_id}/update/stripAccess`, { user_id: id })
        bundled.apiToast(res)
        await refreshAccess();
    }
