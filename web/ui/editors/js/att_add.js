            let addCMChanged = false
            var addCM = new vendored.KNEditor({
	            textArea: document.getElementById("attCreateContent"),
	            language: "text",
	            dynamicSize: true,
            })
            addCM.onChange(() => addCMChanged = true)
            document.getElementById("addDetails").addEventListener("toggle", () => {
                addCM.refresh()
            })
            document.getElementById("attCreateName").addEventListener("input", e => {
                let newMode = bundled.get_cm_mode(e.target.value);
                console.log(newMode)
                if(addCM.getLanguage() !== newMode) {
                    addCM.setLanguage(newMode)
                }

                updateAttFlagsFromName(
                    e.target.value,
                    document.getElementById("attCreateVisible"),
                    document.getElementById("attCreatePrivate"),
                    document.getElementById("attCreateExec"),
                )
            })
            
            async function createNewAttachment(e) {
                e.preventDefault();
                let content = addCM.getText();
                let filename = document.getElementById("attCreateName").value;
        
                let form = new FormData();
                form.append("data", new File([content], filename, {type: "text/plain"}));
        
        
                form.append("visible", document.getElementById("attCreateVisible").checked);
                form.append("private", document.getElementById("attCreatePrivate").checked);
                form.append("exec", document.getElementById("attCreateExec").checked);
        
                let res = await bundled.multipartCall(apiPrefix+"/update/addAttachment", form)
                if (res.status !== "success") {
                    bundled.apiToast(res);
                    return
                }
                addCMChanged = false
                window.location.reload();
            };
        
        
            document.getElementById("addAttForm").addEventListener("submit", createNewAttachment)
            window.addEventListener("beforeunload", e => {
                if(addCMChanged || editCMChanged) {
                    e.preventDefault();
                    e.returnValue = true
                    return true
                }
            })
