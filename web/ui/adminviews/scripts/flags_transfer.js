        document.getElementById("exportFlags").addEventListener("click", async () => {
            const secrets = document.getElementById("exportSecrets").checked
            const res = await bundled.getCallV2("/admin/v2/flags/export", {secrets})
            if (res.status !== "success") {
                bundled.apiToast(res)
                return
            }
            // The v2 API authenticates by header, so the file is built here instead of linked.
            const blob = new Blob([JSON.stringify(res.data, null, 2)], {type: "application/json"})
            const url = URL.createObjectURL(blob)
            const a = document.createElement("a")
            a.href = url
            a.download = `kilonova-flags-${location.hostname}${secrets ? "-with-secrets" : ""}.json`
            a.click()
            URL.revokeObjectURL(url)
        })

        document.getElementById("importFlags").addEventListener("click", async () => {
            const file = document.getElementById("importFile").files[0]
            if (!file) {
                bundled.createToast({status: "error", description: "Choose an exported flags file first"})
                return
            }
            let body
            try {
                body = JSON.parse(await file.text())
            } catch (err) {
                bundled.createToast({status: "error", description: "That file is not valid JSON"})
                return
            }
            const res = await bundled.postCallV2("/admin/v2/flags/import", body)
            if (res.status !== "success") {
                bundled.apiToast(res)
                return
            }
            const {applied, applied_secrets, skipped, errors} = res.data
            const out = document.getElementById("importResult")
            out.replaceChildren()
            const line = (text, cls) => {
                const p = document.createElement("p")
                p.textContent = text
                if (cls) p.className = cls
                out.append(p)
            }
            line(`Applied ${applied.length} flag(s)${applied.length ? ": " + applied.join(", ") : ""}`)
            if (applied_secrets.length) line(`Including secrets: ${applied_secrets.join(", ")}`, "text-red-600")
            for (const [key, reason] of Object.entries(skipped)) line(`Skipped ${key}: ${reason}`, "text-muted")
            for (const err of errors) line(`Not saved: ${err}`, "text-red-600")
            bundled.createToast({
                status: errors.length ? "error" : "success",
                description: errors.length ? "Some flags could not be saved" : "Imported flags. Reload to see them in the editor",
            })
        })
