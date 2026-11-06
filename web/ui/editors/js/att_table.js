
            const mgr = new bundled.CheckboxManager(document.getElementById("selectAllBox"), document.querySelectorAll("[id^='pb-att-']"))

            async function deleteAttachments() {
                var attachments = [];
                for (let e of document.querySelectorAll("[id^='row-att-']")) {
                    let id = parseInt(rowreg.exec(e.id)[1]);
                    if (isNaN(id)) {
                        console.error("id is somehow NaN", e.id)
                        continue
                    }
                    let checked = document.getElementById(`pb-att-${id}`).checked;
                    if (checked) {
                        attachments.push(id);
                    }
                }
                if(attachments.length == 0) {
                    return;
                }
                if(attachments.length == 1) {
                    if(!(await bundled.confirm(bundled.getText("oneAttachmentDeleteConfirm")))) {
                        return
                    }
                } else if(!(await bundled.confirm(bundled.getText("attachmentDeleteConfirm", attachments.length)))) {
                    return
                }
                let res = await bundled.bodyCall(apiPrefix+"/update/bulkDeleteAttachments", attachments)
                if (res.status === "success") {
                    window.location.reload();
                    return;
                }
                bundled.apiToast(res)
            }

            async function updateAttachments() {
                var atts = {};
                for (let e of document.querySelectorAll("[id^='row-att-']")) {
                    let id = parseInt(rowreg.exec(e.id)[1]);
                    if (isNaN(id)) {
                        console.error("id is somehow NaN", e.id)
                        continue
                    }
                    let visible = document.getElementById(`visible-att-${id}`).checked;
                    let private = document.getElementById(`private-att-${id}`).checked;
                    let exec = document.getElementById(`exec-att-${id}`).checked;
                    atts[id] = { visible, private, exec }
                }
                let res = await bundled.bodyCall(apiPrefix+"/update/bulkUpdateAttachmentInfo", atts)
                bundled.apiToast(res)
            }

            (() => {
                for (let el of document.querySelectorAll("[id^='name-att-'] > a")) {
                    el.innerHTML = `<i class="fas ${bundled.getFileIcon(el.innerText)} fa-fw"></i> ${el.innerText}`
                }
                for (let el of document.querySelectorAll("[id^='size-att-']")) {
                    el.innerText = bundled.sizeFormatter(Number(el.innerText))
                }
            })()

