const pref = __URL_PREFIX__;
async function updateID(e) {
	e.preventDefault()
	let q = {
		id: document.getElementById("vID").value,
        score: document.getElementById("score").value
	}
	let res = await bundled.postCall("/problem/__PBID__/update/test/__TEST_VID__/info", q);

	bundled.apiToast(res)
}

async function updateData(e) {
	e.preventDefault()
    var formdata = new FormData()
    const inputVal = document.getElementById("input").value
    if(__OK_IN__ || inputVal !== __ORIG_IN__) {
        formdata.set("input", new File([inputVal], "input", {type: "text/plain"}))
    }

    const outputVal = document.getElementById("output").value
    if(__OK_OUT__ || outputVal !== __ORIG_OUT__) {
        formdata.set("output", new File([outputVal], "output", {type: "text/plain"}))
    }

	let res = await bundled.multipartCall("/problem/__PBID__/update/test/__TEST_ID__/data", formdata)
	bundled.apiToast(res);
}


document.getElementById("newInputCheckbox").addEventListener("change", e => {
    document.getElementById("newInputLabel").classList.toggle("text-muted", !e.currentTarget.checked)
    document.getElementById("newInput").disabled = !e.currentTarget.checked
    document.getElementById("newInput").required = e.currentTarget.checked
})
document.getElementById("newOutputCheckbox").addEventListener("change", e => {
    document.getElementById("newOutputLabel").classList.toggle("text-muted", !e.currentTarget.checked)
    document.getElementById("newOutput").disabled = !e.currentTarget.checked
    document.getElementById("newOutput").required = e.currentTarget.checked
})
async function uploadReplacementData(e) {
    e.preventDefault()
    var formdata = new FormData()

    let newIn = document.getElementById("newInput"),
        newInBox = document.getElementById("newInputCheckbox")
    if(newInBox.checked && newIn.files.length > 0) {
        formdata.set("input", newIn.files[0])
    }
    let newOut = document.getElementById("newOutput"),
        newOutBox = document.getElementById("newOutputCheckbox")
    if(newOutBox.checked &&newOut.files.length > 0) {
        formdata.set("output", newOut.files[0])
    }

    let res = await bundled.multipartCall("/problem/__PBID__/update/test/__TEST_ID__/data", formdata)
    if(res.status === "success") {
        window.location.reload();
        return
    }
    bundled.apiToast(res)
}

async function removeTest(e) {
	e.preventDefault()

	if(!(await bundled.confirm(bundled.getText("confirmTestDelete")))) {
		return
	}
	let res = await bundled.postCall("/problem/__PBID__/update/test/__TEST_ID__/delete", {})
	if(res.status === "success") {
		window.location.assign(pref + "/problems/__PBID__/edit/test")
		return
	}
	bundled.apiToast(res);
}
document.getElementById("test_edit_form").addEventListener("submit", updateData)
document.getElementById("test_id_edit_form").addEventListener("submit", updateID)
document.getElementById("test_del_button").addEventListener("click", removeTest)
document.getElementById("test_reupload_form").addEventListener("submit", uploadReplacementData)
