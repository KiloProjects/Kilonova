        function refreshRoles() {
            return htmx.ajax("GET", location.href, {target: "#role_lists", select: "#role_lists", swap: "morph"})
        }

        async function setAdmin(id, set) {
            let res = await bundled.postCall("/admin/setAdmin", {id, set})
            bundled.apiToast(res)
            if (res.status == "error") {
                return
            }
            await refreshRoles();
        }

        async function setProposer(id, set) {
            let res = await bundled.bodyCall("/admin/setProposer", {id, set})
            bundled.apiToast(res)
            if (res.status == "error") {
                return
            }
            await refreshRoles();
        }

        async function setFormAdmin(toSet) {
            let user = document.getElementById("userToChange").value;
            let res = await bundled.getCall(`/user/byName/${user}`, {})
            if (res.status == "error") {
                bundled.apiToast(res)
                return
            }
            await setAdmin(res.data.id, toSet)
        }

        async function setFormProposer(toSet) {
            let user = document.getElementById("userToChange").value;
            let res = await bundled.getCall(`/user/byName/${user}`, {})
            if (res.status == "error") {
                bundled.apiToast(res)
                return
            }
            await setProposer(res.data.id, toSet)
        }
