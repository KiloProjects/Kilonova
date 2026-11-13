        function wrapFunc(func) {
            return async (e) => {
                if (!e.submitter) {
                    console.log(e.submitter)
                    return func(e)
                }
                e.submitter.disabled = true
                e.submitter.classList.add('btn-loading')
                await func(e)
                e.submitter.classList.remove('btn-loading')
                e.submitter.disabled = false
            }
        }

        document.getElementById("flagsForm").addEventListener("submit", wrapFunc(async (e) => {
            e.preventDefault()
            let data = {bool_flags: {}, string_flags: {}, int_flags: {}, int_array_flags: {}, str_array_flags: {}}
            for (let checkbox of document.getElementById("boolFlagsDiv").getElementsByTagName("input")) {
                if (checkbox.checked != (checkbox.dataset.org_value === "true")) {
                    data.bool_flags[checkbox.dataset.name] = checkbox.checked;
                }
            }
            for (let input of document.getElementById("stringFlagsDiv").getElementsByTagName("input")) {
                if (input.value != input.dataset.org_value) {
                    data.string_flags[input.dataset.name] = input.value;
                }
            }
            for (let input of document.getElementById("intFlagsDiv").getElementsByTagName("input")) {
                if (input.value != input.dataset.org_value) {
                    data.int_flags[input.dataset.name] = parseInt(input.value);
                }
            }
            for (let [div, key, type] of [["intArrayFlagsDiv", "int_array_flags", "number"], ["strArrayFlagsDiv", "str_array_flags", "string"]]) {
                for (let input of document.getElementById(div).getElementsByTagName("input")) {
                    if (input.value == input.dataset.org_value) {
                        continue
                    }
                    let val
                    try {
                        val = JSON.parse(input.value)
                    } catch (err) {
                        val = null
                    }
                    if (!Array.isArray(val) || !val.every((v) => typeof v === type && (type !== "number" || Number.isInteger(v)))) {
                        bundled.createToast({status: "error", description: `${input.dataset.name}: expected a JSON array of ${type}s`})
                        return
                    }
                    data[key][input.dataset.name] = val
                }
            }
            // console.log(Object.fromEntries(data.entries()))
            const res = await bundled.bodyCall("/admin/updateFlags", data)
            bundled.apiToast(res)
            if (res.status === "success") {
                for (let checkbox of document.getElementById("boolFlagsDiv").getElementsByTagName("input")) {
                    checkbox.dataset.org_value = checkbox.checked
                }
                for (let div of ["stringFlagsDiv", "intFlagsDiv", "intArrayFlagsDiv", "strArrayFlagsDiv"]) {
                    for (let input of document.getElementById(div).getElementsByTagName("input")) {
                        input.dataset.org_value = input.value
                    }
                }
            }
        }))
