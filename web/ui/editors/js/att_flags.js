var rowreg = /row-att-([0-9]+)/;
function updateAttFlagsFromName(name, cVisible, cPrivate, cExec) {
    let isChanging = false;
    if(name.startsWith("checker") || name.startsWith("grader") || name.startsWith("manager")) {
        if(cPrivate.checked === false || cExec.checked === false) {
            isChanging = true;
        }
        cPrivate.checked = true
        cExec.checked = true
    }
    if(name.toLowerCase().startsWith("lgrader") || name.toLowerCase().startsWith("_grader")) {
        if(cVisible.checked === false || cExec.checked === true) {
            isChanging = true;
        }
        cVisible.checked = true
        cExec.checked = false
    }
    if(name.endsWith(".h") || name.startsWith("stub")) {
        if(cVisible.checked === false || cExec.checked === false) {
            isChanging = true;
        }
        cVisible.checked = true
        cExec.checked = true
    }
    if(name === ".output_only") {
        if(cVisible.checked != false || cExec.checked != true || cPrivate.checked != true) {
            isChanging = true
        }
        cVisible.checked = false
        cExec.checked = true
        cPrivate.checked = true
    }
    if(isChanging) {
        bundled.createToast({status: "info", description: bundled.getText("att_suggest_text")})
    }
}
