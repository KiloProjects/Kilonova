
    async function reevaluateSubs() {
        if (!(await bundled.confirm(bundled.getText("confirmSubReevaluate")))) {
            return
        }
        let res = await bundled.postCall(`/problem/${problem.id}/reevaluateSubs`, {})
        bundled.apiToast(res)
    }

    async function updateProblem(e) {
        e.preventDefault();
        const data = {
            name: document.getElementById("problemName").value,
            source_credits: document.getElementById("sourceCredits").value,
            console_input: document.getElementById("consoleInput").checked,
            scoring_strategy: document.getElementById("scoring_strategy").value,
            test_name: document.getElementById("testName").value,
            memory_limit: Math.trunc(parseFloat(document.getElementById("memoryLimit").value) * 1024),
            time_limit: parseFloat(document.getElementById("timeLimit").value || "0"),
            default_points: parseFloat(document.getElementById("defaultPoints").value || "0"),
            score_scale: parseFloat(document.getElementById("scoreScale").value || "0"),
            source_size: parseFloat(document.getElementById("sourceSize").value || "0"),
            score_precision: parseInt(document.getElementById("scorePrecision").value || "0"),
            visible_tests: document.getElementById("visibleTests").checked,
            task_type: document.getElementById("taskType").value,
            communication_processes: parseInt(document.getElementById("communicationProcesses").value || "1"),
        }

        if (data.name === "") {
            bundled.createToast({ status: "error", description: bundled.getText("emptyTitle") });
            return
        }
        bundled.apiToast(await bundled.postCall(`/problem/${problem.id}/update/`, data));
    }

    document.getElementById("scorePrecision").addEventListener("change", e => {
        document.getElementById("defaultPoints").step = Math.pow(10, -e.currentTarget.value)
    })

    async function deleteProblem(e) {
        e.preventDefault();
        if (!(await bundled.confirm(bundled.getText("confirmProblemDelete")))) {
            return
        }
        let res = await bundled.postCall(`/problem/${problem.id}/delete`, {})
        if (res.status === "success") {
            window.location.assign("/");
            return
        }
        bundled.apiToast(res)
    }

    document.getElementById("updateProblemForm").addEventListener("submit", updateProblem);
    document.getElementById("deleteProblemButton").addEventListener("click", deleteProblem);

    document.getElementById("testName").disabled = document.getElementById("consoleInput").checked;
    document.getElementById("consoleInput").addEventListener("change", (e) => {
        document.getElementById("testName").disabled = document.getElementById("consoleInput").checked;
    });
