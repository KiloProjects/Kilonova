        async function loadUsers(q, id = "admin-group", cb) {
            document.getElementById(id).innerHTML = `<div class="list-group-head">${bundled.getText("loading")}</div>`
            let res = await bundled.getCall("/admin/getAllUsers", q)
            if (res.status == "error") {
                bundled.apiToast(res)
                return
            }
            let outhtml = ""
            for (let user of res.data.users) {
                outhtml += `<div class="list-group-item flex justify-between items-center"><a href="/profile/${user.name}"><img class="rounded-sm inline-block mr-2" src="/profile/${user.name}/avatar?s=32" width="32" height="32" alt="Avatar"/>${user.name}</a>`
                if (user.id != 1 && AUTHED_USER_ID == 1) {
                    outhtml += `<span style="cursor: pointer;" onclick="${cb}(${user.id}, false)">X</span>`
                }
                outhtml += `</div>`
            }
            document.getElementById(id).innerHTML = outhtml
        }

        loadUsers({admin: true}, "admin-group", "setAdmin");
        loadUsers({proposer: true}, "proposer-group", "setProposer");
