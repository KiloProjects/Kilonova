            let editCMChanged = false
            var editCM = new vendored.KNEditor({
	            textArea: document.getElementById("attEditContent"),
	            language: "text",
	            // dynamicSize: true,
            })
            editCM.onChange(() => editCMChanged = true)
            document.getElementById("addDetails").addEventListener("toggle", () => {
                editCM.refresh()
            })

            async function editAttachment(e) {
                e.preventDefault();
                let content = editCM.getText();
                let filename = document.getElementById("editAttName").value;
        
                let form = new FormData();
                form.append("data", new File([content], filename, {type: "text/plain"}));
                form.append("id", document.getElementById("editAttID").value)
        
                let res = await bundled.multipartCall(apiPrefix+"/update/attachmentData", form)
                bundled.apiToast(res);
                if(res.status == "success") {
                    editCMChanged = false
                }
            };
        
            async function initAttEdit(id) {
                const rez = await bundled.getCall(apiPrefix+"/get/attachment/"+id)
                if(rez.status !== "success") {
                    bundled.apiToast(rez)
                    return
                }
        
                const mimetype = rez.data.mime_type
                if(mimetype.includes("image")) {
                    bundled.apiToast({status: "error", data: "Cannot edit images using editor!"})
                    return
                }
                if(mimetype.includes("archive")) {
                    bundled.apiToast({status: "error", data: "Cannot edit archives using editor!"})
                    return
                }
                if(mimetype.includes("pdf")) {
                    bundled.apiToast({status: "error", data: "Cannot edit PDF files using editor!"})
                    return
                }
        
                document.getElementById("addDetails").open = false
                document.getElementById("editAttID").value = id
                document.getElementById("editAttName").value = rez.data.metadata.name
                document.getElementById("ediAttNameView").innerText = rez.data.metadata.name
                editCM.setText(bundled.fromBase64(rez.data.data))
                editCM.setLanguage(bundled.get_cm_mode(rez.data.metadata.name));
                editCMChanged = false
        
                document.getElementById("editAttView").classList.toggle("hidden", false);
                editCM.refresh();
            }
        
            function toggleEdit(id) {
                if(document.getElementById("editAttID").value == id) {
                    closeAttEdit();
                } else {
                    initAttEdit(id)
                }
            }

			async function transcribeStatement(name) {
				const res = await bundled.postCall(apiPrefix+"/update/transcribeStatement", {
					"filename": name
				})
				bundled.apiToast(res)
			}
        
            async function closeAttEdit() {
                document.getElementById("editAttView").classList.toggle("hidden", true);
                document.getElementById("editAttID").value = ""
                document.getElementById("editAttName").value = ""
                editCM.setText("")
            }
        
            document.getElementById("editAttForm").addEventListener("submit", editAttachment)
