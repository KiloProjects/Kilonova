package contests

// Inline scripts of the contest edit page, kept verbatim from the old template.

// editInviteScript: invite handling; {{ID}} is replaced with the contest ID.
const editInviteScript = `
    // invite handling

    async function copyInvitationURL(id) {
        try {
            await navigator.clipboard.writeText(window.location.origin + "/contests/invite/" + id)
            bundled.apiToast({status: "success", data: bundled.getText("copied")})
        } catch(e) {
            console.error(e)
            bundled.apiToast({status: "error", data: bundled.getText("notCopied")})
        }
    }

    async function invalidateInvite(id) {
        let res = await bundled.postCall("/contest/updateInvitation", {invite_id: id, expired: true})
        if(res.status === "error") {
            bundled.apiToast(res)
            return
        }
        window.location.reload()
    }

    async function createInvite(numUses) {
        let res = await bundled.postCall("/contest/{{ID}}/createInvitation", {max_uses: numUses})
        if(res.status === "error") {
            bundled.apiToast(res)
            return
        }
        await copyInvitationURL(res.data)
        window.location.reload()
    }

    async function createMOSS() {
        bundled.apiToast({status: "info", data: "Submitting MOSS request. Will take a while, page will reload on finish."})
        let res = await bundled.postCall("/contest/{{ID}}/runMOSS", {})
        if(res.status === "error") {
            bundled.apiToast(res)
            return
        }
        window.location.reload()
    }
`

// editDatetimeScript: datetime helpers; the calls are appended by editScripts.
const editDatetimeScript = `
// datetime handling
function setDatetime(inputID, backendValue) {
    document.getElementById(inputID).value = bundled.dayjs(backendValue).format("YYYY-MM-DD[T]HH:mm")
}
`

// editToggleScript: shows/hides dependent leaderboard options.
const editToggleScript = `
// contest option hiding handling
document.getElementById("c_freeze_enabled")?.addEventListener("change", (e) => {
    document.getElementById("freezeLabel").classList.toggle("hidden", !e.currentTarget.checked)
})
document.getElementById("leaderboard_style")?.addEventListener("change", (e) => {
    document.getElementById("icpcPenaltyLabel").classList.toggle("hidden", e.currentTarget.value !== "acm-icpc")
})
`

// editUpdateScript: follows the contest_id declaration.
const editUpdateScript = `
async function updateContest(e) {
    e.preventDefault();
    try {
        var fd = new FormData(e.currentTarget)

        let leaderboardFreeze = document.getElementById("c_freeze_enabled").checked;

        var data = {
            name: fd.get("name"),
            max_subs: fd.get("max_subs"),
            public_join: document.getElementById("c_public_join").checked,
            visible: document.getElementById("c_visible").checked,
            start_time: bundled.formatISO8601(fd.get("start_time")),
            end_time: bundled.formatISO8601(fd.get("end_time")),

            public_leaderboard: document.getElementById("c_public_leaderboard").checked,
            leaderboard_advanced_filter: document.getElementById("c_advanced_filters").checked,
            leaderboard_style: fd.get("leaderboard_style"),
            change_leaderboard_freeze: true,
            leaderboard_freeze: leaderboardFreeze ? bundled.formatISO8601(fd.get("freeze_time")) : undefined,
            icpc_submission_penalty: fd.get("icpc_submission_penalty"),

            question_cooldown: parseInt(fd.get("question_cooldown"))*1000,
            submission_cooldown: parseInt(fd.get("submission_cooldown"))*1000,
            
            per_user_time: fd.get("per_user_time"),
            register_during_contest: document.getElementById("c_reg").checked,

            ip_management_enabled: document.getElementById("c_ip_mgmt").checked,
            whitelist_enabled: document.getElementById("c_whitelist").checked,
        }

        if(!document.getElementById("contest_type").disabled) {
            data.type = document.getElementById("contest_type").value
        }
        console.log(fd, data)
    } catch(e) {
        bundled.apiToast({status: "error", data: e.toString()});
        return
    }

    const res = await bundled.postCall(` + "`" + `/contest/${contest_id}/update` + "`" + `, data)
    if(res.status === "success") {
        window.location.reload();
        return
    }
    bundled.apiToast(res)
}

async function createPblistFromContest(e) {
    e.preventDefault();
    try {
        var data = {
            title: document.getElementById("contest_name_upd").value,
            ids: bundled.stringIntToNumber(document.getElementById("contest_list").value.split(',')),
        }
        const parentID = parseInt(document.getElementById("cr_pblist_parent_id").value);
        if(!isNaN(parentID)) {
            data.parent_id = parentID
        }
    } catch(e) {
        bundled.apiToast({status: "error", data: e.toString()});
        return
    }
	let res = await bundled.bodyCall("/problemList/create", data)
	if(res.status === "success") {
		window.location.assign(` + "`" + `/problem_lists/${res.data.id}` + "`" + `);
		return
	}
	bundled.apiToast(res)
}

let descCMChanged = false;
window.addEventListener("beforeunload", e => {
    if(descCMChanged) {
        e.preventDefault();
        e.returnValue = true
        return true
    }
})
var desc_cm = new vendored.KNEditor({
	textArea: document.getElementById("contestDesc"),
	language: "md",
	dynamicSize: true,
})
desc_cm.onChange(() => descCMChanged = true)

async function updateContestDesc(e) {
    e.preventDefault()
    const rez = await bundled.postCall(` + "`" + `/contest/${contest_id}/update` + "`" + `, {description: desc_cm.getText()})
    bundled.apiToast(rez)
    if(rez.status === "success") {
        descCMChanged = false
    }
}

document.getElementById("contest_update_form")?.addEventListener("submit", updateContest)
document.getElementById("contest_description_form")?.addEventListener("submit", updateContestDesc)
document.getElementById("contest_pblist_form")?.addEventListener("submit", createPblistFromContest)
`

const editCloneScript = `
async function cloneContest(e) {
    e.preventDefault();
    const defaultName = ` + "`" + `${document.getElementById("contest_name_upd").value} (copy)` + "`" + `;
    const res = await bundled.postCall(` + "`" + `/contest/${contest_id}/clone` + "`" + `, {name: defaultName.trim()});
    if (res.status === "success") {
        window.location.assign(` + "`" + `/contests/${res.data}/manage/edit` + "`" + `);
        return;
    }
    bundled.apiToast(res);
}
document.getElementById("cloneContestButton")?.addEventListener("click", cloneContest);
`

const editDeleteScript = `
async function deleteContest(e) {
    e.preventDefault();
    if (!(await bundled.confirm(bundled.getText("confirmContestDelete")))) {
        return
    }
    let res = await bundled.postCall(` + "`" + `/contest/${contest_id}/delete` + "`" + `, {})
    if (res.status === "success") {
        window.location.assign("/contests/");
        return
    }
    bundled.apiToast(res)
}
document.getElementById("deleteContestButton")?.addEventListener("click", deleteContest);
`

const editProblemsScript = `
// Contest problems-specific stuff
async function updateContestProblems(e) {
    e.preventDefault();
    let data = {
        list: bundled.stringIntToNumber(document.getElementById("contest_list").value.split(',')),
    }
    let res = await bundled.bodyCall(` + "`" + `/contest/${contest_id}/update/problems` + "`" + `, data)
    if(res.status === "success") {
        window.location.reload();
        return
    }
    bundled.apiToast(res)
}
document.getElementById("contest_problems_form")?.addEventListener("submit", updateContestProblems)
`

const editAccessScript = `
// Contest access-specific stuff
async function addEditor() {
        const username = document.getElementById('userToAdd').value;
        let res = await bundled.postCall(` + "`" + `/contest/${contest_id}/update/addEditor` + "`" + `, { username: username })
        bundled.apiToast(res)
        await loadView();
    }

    async function addTester() {
        const username = document.getElementById('userToAdd').value;
        let res = await bundled.postCall(` + "`" + `/contest/${contest_id}/update/addTester` + "`" + `, { username: username })
        bundled.apiToast(res)
        await loadView();
    }

    async function stripAccess(id) {
        let res = await bundled.postCall(` + "`" + `/contest/${contest_id}/update/stripAccess` + "`" + `, { user_id: id })
        bundled.apiToast(res)
        await loadView();
    }

    async function loadContest() {
        const res = await bundled.getCall(` + "`" + `/contest/${contest_id}/` + "`" + `, {});
        if (res.status === "error") {
            bundled.apiToast(res)
            return
        }
        return res.data
    }

    async function loadUsers(users, id) {
        //document.getElementById(id).innerHTML = ` + "`" + `<div class="list-group-head">${bundled.getText("loading")}</div>` + "`" + `
        let outhtml = ""
        if (users.length == 0) {
            outhtml += ` + "`" + `<div class="list-group-head">No users</div>` + "`" + `
        }
        for (let user of users) {
            outhtml += ` + "`" + `
            <div class="list-group-item flex justify-between items-center">
                <a href="/profile/${user.name}">
                    <img class="rounded-sm inline-block mr-2" src="/profile/${user.name}/avatar?s=32" width="32" height="32" alt="Avatar"/>
                    ${user.name}
                </a>
                ${user.id != window.platform_info.user_id ? ` + "`" + `<span style='cursor: pointer;' onclick='stripAccess(${user.id})'>X</span>` + "`" + ` : ''}
            </div>` + "`" + `
        }
        document.getElementById(id).innerHTML = outhtml
    }

    async function loadView() {
        const data = await loadContest();
        loadUsers(data.editors, 'editors_list')
        loadUsers(data.testers, 'testers_list')
    }

    document.addEventListener("DOMContentLoaded", () => loadView());
`
