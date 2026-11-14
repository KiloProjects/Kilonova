        function wrapFunc(func) {
            return async (e) => {
                if (!e.submitter) {
                    return func(e)
                }
                e.submitter.disabled = true
                e.submitter.classList.add('btn-loading')
                await func(e)
                e.submitter.classList.remove('btn-loading')
                e.submitter.disabled = false
            }
        }

        // collectFlags builds the PATCH /api/admin/v2/flags body from the changed
        // [data-flag][data-type] inputs in form, wherever they sit in it. It returns
        // null, after a toast, when an input does not parse.
        function collectFlags(form) {
            const data = {bool_flags: {}, string_flags: {}, int_flags: {}, int_array_flags: {}, str_array_flags: {}}
            for (const input of form.querySelectorAll("input[data-flag][data-type]")) {
                const name = input.dataset.flag
                const type = input.dataset.type
                if (type === "bool") {
                    if (input.checked !== (input.dataset.org_value === "true")) {
                        data.bool_flags[name] = input.checked
                    }
                    continue
                }
                if (input.value === input.dataset.org_value) {
                    continue
                }
                if (type === "string") {
                    data.string_flags[name] = input.value
                } else if (type === "int") {
                    const val = Number(input.value)
                    if (input.value.trim() === "" || !Number.isInteger(val)) {
                        bundled.createToast({status: "error", description: `${name}: expected an integer`})
                        return null
                    }
                    data.int_flags[name] = val
                } else if (type === "int_array" || type === "str_array") {
                    const elem = type === "int_array" ? "number" : "string"
                    let val
                    try {
                        val = JSON.parse(input.value)
                    } catch (err) {
                        val = null
                    }
                    if (!Array.isArray(val) || !val.every((v) => typeof v === elem && (elem !== "number" || Number.isInteger(v)))) {
                        bundled.createToast({status: "error", description: `${name}: expected a JSON array of ${elem}s`})
                        return null
                    }
                    data[type + "_flags"][name] = val
                }
            }
            return data
        }

        // reportFlagUpdate toasts the result of PATCH /api/admin/v2/flags and returns the saved flag names.
        function reportFlagUpdate(res) {
            if (res.status !== "success") {
                bundled.apiToast(res)
                return []
            }
            if (res.data.errors.length > 0) {
                bundled.createToast({status: "error", description: res.data.errors.join("; ")})
            } else {
                bundled.createToast({status: "success", description: "Updated flags. Some changes may only apply after a restart"})
            }
            return res.data.updated
        }

        for (const form of document.querySelectorAll("form[data-flags-form]")) {
            form.addEventListener("submit", wrapFunc(async (e) => {
                e.preventDefault()
                const data = collectFlags(form)
                if (data === null) {
                    return
                }
                const saved = new Set(reportFlagUpdate(await bundled.patchCallV2("/admin/v2/flags", data)))
                for (const input of form.querySelectorAll("input[data-flag][data-type]")) {
                    if (saved.has(input.dataset.flag)) {
                        input.dataset.org_value = input.dataset.type === "bool" ? String(input.checked) : input.value
                    }
                }
            }))
        }
