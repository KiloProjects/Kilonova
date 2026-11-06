
["input", "output"].map(type => document.getElementById(type+"_submit_style").addEventListener("change", (e) => {
    const val = e.currentTarget.value;
    document.getElementById(type+"_file_label")?.classList.toggle("hidden", val === "text")
    document.getElementById(type+"_textarea_label")?.classList.toggle("hidden", val !== "text")
}))

function getValue(type) {
    if(document.getElementById(type+"_submit_style").value === "text") {
        return new File([document.getElementById(type+"_textarea").value], "test."+type)
    }
    const files = document.getElementById(type+"_file").files
    if(files.length > 1) {
        throw new Error(bundled.getText("invalid_file"))
    }
    if(files.length === 0) {
        throw new Error(bundled.getText("no_file"))
    }
    return files[0]
}

async function addTest(e) {
	e.preventDefault()
    try {
        var data = new FormData();
        data.append("visibleID", document.getElementById("vID").value);
        data.append("score", document.getElementById("score").value);
        data.append("input", getValue("input"));
        data.append("output", getValue("output"));

        let res = await bundled.multipartCall("/problem/__PBID__/update/addTest", data)
        if(res.status === "success") {
            window.location.reload();
        }
    } catch(e) {
        bundled.apiToast({status: "error", data: e.toString()})
        return
    }
}
document.getElementById("test_create_form").addEventListener("submit", addTest)
