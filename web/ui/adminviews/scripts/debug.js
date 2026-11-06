        async function resetSubs() {
            bundled.apiToast(await bundled.postCall("/admin/maintenance/resetWaitingSubs", {}))
        }

        async function mdCacheWarmup() {
            bundled.apiToast({status: "info", data: bundled.getText("warmingUp")})
            let res = await bundled.postCall(`/admin/maintenance/mdCacheWarmup`, {})
            if(res.status !== "success") {
                bundled.apiToast(res)
                return
            }
            window.location.reload()
        }
        async function evictBucket(name) {
            let res = await bundled.postCall(`/admin/maintenance/bucket/${name}/evictObjects`, {})
            if(res.status !== "success") {
                bundled.apiToast(res)
                return
            }
            window.location.reload()
        }
        async function cleanBucket(name) {
            let res = await bundled.postCall(`/admin/maintenance/bucket/${name}/cleanCache`, {})
            if(res.status !== "success") {
                bundled.apiToast(res)
                return
            }
            window.location.reload()
        }
        async function refreshBucket(name) {
            let res = await bundled.postCall(`/admin/maintenance/bucket/${name}/stats`, {refresh: true})
            if(res.status !== "success") {
                bundled.apiToast(res)
                return
            }
            window.location.reload()
        }
