    let cm = new vendored.KNEditor({
	    textArea: document.getElementById("description"),
	    language: "md",
        dynamicSize: true,
        autoFocus: true,
        vimMode: localStorage.getItem("vimMode") === "true",
    })
    cm.onChange(() => changed = true)

    document.getElementById("formatDocumentButton").addEventListener("click", () => {
        let val = cm.getText();
        let newVal = rewriteLaTeXMarkdown(val);
        if(val === newVal) {
            bundled.createToast({
                status: "info",
                description: bundled.getText("notif.edit.no_changes")
            })
            return
        }
        cm.setText(newVal);
        bundled.createToast({
            status: "success",
            description: bundled.getText("notif.edit.rewrote_rule")
        })
    })

    cm.onPaste((pasteText) => {
	    if(document.getElementById("autoformat_paste")?.checked) {
            let updated = false;
            let newVal = pasteText.map((val) => {
                const transformed = rewriteLaTeXMarkdown(val);
                if(transformed !== val) updated = true;
                return transformed;
            });
            if(updated) {
                bundled.createToast({
                    status: "success",
                    description: bundled.getText("notif.edit.rewrote_rule")
                })
	            return newVal
            }
        }
    })

    function formatStatement(lang, type) {
        if(type == "") {
            return `statement-${lang}.md`
        }
        return `statement-${lang}-${type}.md`
    }

    function setType(newType) {
        document.getElementById("variantType").value = newType
        let varText = ""
        if(typeof newType == 'string' && newType.length > 0) {
            varText = `${bundled.getText("selectedType")}: ${bundled.maybeGetText("stmt_type."+newType)}`
        }
        document.getElementById("selectedVariant").innerText = varText
    }
    setType(initialVariantType)

    async function reloadStatement(newLang, newType) {
        const rez = await bundled.getCall(`${apiPrefix}/get/attachmentByName/${formatStatement(newLang, newType)}`)
        if(rez.status !== "success") {
            document.getElementById("private_checkbox").checked = false
            document.getElementById("variantSelect").value = newLang
            currentLang = newLang
            setType(newType)
            if (document.getElementById("attID").value != -1) {
                cm.setText("")
                changed = false
            }
            document.getElementById("attID").value = -1
            return
        }

        document.getElementById("private_checkbox").checked = rez.data.metadata.private
        document.getElementById("attID").value = rez.data.metadata.id
        document.getElementById("variantSelect").value = newLang
        currentLang = newLang
        setType(newType)
        cm.setText(bundled.fromBase64(rez.data.data))
        changed = false
    }

    document.querySelectorAll("#variantList a").forEach(val => {
        val.addEventListener("click",async e => {
            e.preventDefault()
            if (changed && !(await bundled.confirm(bundled.getText("variantChangeConfirm")))) {
                return
            }
            await reloadStatement(e.target.dataset.lang, e.target.dataset.type)
        })
    })

    async function updateStatement(e) {
        e.preventDefault();
        let content = cm.getText();
        let filename = formatStatement(document.getElementById("variantSelect").value, document.getElementById("variantType").value);
        let id = document.getElementById("attID").value

        let form = new FormData();
        form.append("data", new File([content], filename, {type: "text/plain"}));
        form.append("private", document.getElementById("private_checkbox").checked);
        
        console.log(id)
        if(id.toString() !== "-1") {
            // Update attachment
            form.append("id", id)
    
            let res = await bundled.multipartCall(apiPrefix+"/update/attachmentData", form)
            bundled.apiToast(res);
            if(res.status === "success") {
                changed = false
            }
        } else {
            // Create attachment
            let res = await bundled.multipartCall(apiPrefix+"/update/addAttachment", form)
            if (res.status !== "success") {
                bundled.apiToast(res);
                return
            }

            changed = false
            bundled.apiToast({status: "success", data: "Created statement."})
            document.getElementById("attID").value = res.data
        }
    }

    document.addEventListener('keydown', e => {
        if ((e.ctrlKey || e.metaKey) && e.key === 's') {
            updateStatement(e);
        }
    });

    document.getElementById("variantSelect").addEventListener("change", async e => {
        if (changed && !(await bundled.confirm(bundled.getText("variantChangeConfirm")))) {
            e.preventDefault()
            e.target.value = currentLang
            return
        }
        reloadStatement(e.target.value, document.getElementById("variantType").value)
    })
    document.getElementById("statementUpdForm").addEventListener("submit", updateStatement)
