        document.getElementById("flashesForm").addEventListener("submit", wrapFunc(async (e) => {
            e.preventDefault()
            const form = e.currentTarget
            // Rebuilt from the registered flashes only, so names no longer in the code are dropped.
            const hidden = [...form.querySelectorAll("input[data-flash]")].filter((c) => c.checked).map((c) => c.dataset.flash)
            reportFlagUpdate(await bundled.patchCallV2("/admin/v2/flags", {str_array_flags: {[form.dataset.flag]: hidden}}))
        }))
