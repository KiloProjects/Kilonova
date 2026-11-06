            document.getElementById("attUploadPrivate").addEventListener("change", (e) => {
                if (e.target.checked === true) {
                    document.getElementById("visibleUploadLabel").style.display = "none"
                } else {
                    document.getElementById("visibleUploadLabel").style.display = null
                }
            })
            document.getElementById("attCreatePrivate").addEventListener("change", (e) => {
                if (e.target.checked === true) {
                    document.getElementById("visibleCreateLabel").style.display = "none"
                } else {
                    document.getElementById("visibleCreateLabel").style.display = null
                }
            })
            document.getElementById("attUploadNameLabel").classList.add("hidden")
            function updateAttFlags(newName) {
                updateAttFlagsFromName(
                    newName,
                    document.getElementById("attUploadVisible"),
                    document.getElementById("attUploadPrivate"),
                    document.getElementById("attUploadExec"),
                )
            }
            document.getElementById("attUploadName").addEventListener("input", e => updateAttFlags(e.target.value))
            document.getElementById("attFile").addEventListener("input", e => {
                document.getElementById("attUploadNameLabel").classList.toggle("hidden", e.target.files.length !== 1);
                if(e.target.files.length !== 1) {
                    return
                }
                document.getElementById("attUploadName").value = e.target.files[0].name
                updateAttFlags(e.target.files[0].name)
            })

            async function uploadAttachment(e) {
                e.preventDefault()
                let files = document.getElementById("attFile").files;
                if (files === null || files.length == 0) {
                    bundled.createToast({ status: "error", title: bundled.getText("noFiles") });
                    return
                }
                const fname = document.getElementById("attUploadName").value;
                if(files.length === 1 && fname.length === 0) {
                    bundled.createToast({status: "error", title: "Invalid name"})
                    return
                }
                for(let file of files) {
                    bundled.apiToast({status: "info", data: `Uploading ${file.name}`});
                    let form = new FormData();
                    form.append("data", file, (files.length === 1 ? fname : undefined));

                    form.append("visible", document.getElementById("attUploadVisible").checked);
                    form.append("private", document.getElementById("attUploadPrivate").checked);
                    form.append("exec", document.getElementById("attUploadExec").checked);
            
                    let res = await bundled.multipartProgressCall(apiPrefix+"/update/addAttachment", form)
                    if (res.status !== "success") {
                        bundled.apiToast(res);
                        return
                    }
                }
                window.location.reload();
            }

            document.getElementById("attCreate").addEventListener("submit", uploadAttachment)
