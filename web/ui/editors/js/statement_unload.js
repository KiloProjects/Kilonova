    let changed = false;
    window.addEventListener("beforeunload", e => {
        if(changed) {
            e.preventDefault();
            e.returnValue = true
            return true
        }
    })
