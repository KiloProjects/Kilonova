var pbid = __PBID__, rowreg = /row-test-([0-9]+)/;

const mgr = new bundled.CheckboxManager(document.getElementById("selectAllBox"), document.querySelectorAll("[id^='pb-test-']"))

async function createSubTask(e) {
	e.preventDefault();
	let data = {
		visible_id: parseInt(document.getElementById('subtask-id').value),
		score: parseInt(document.getElementById('subtask-score').value),
		tests: []
	};
	
	if(isNaN(data.visible_id) || isNaN(data.score)) {
		bundled.apiToast({status: "error", data: "Invalid score/id. Please contact administrator"})
		return
	}

	for(let e of document.querySelectorAll("[id^='row-test-']")) {
		let id = parseInt(rowreg.exec(e.id)[1]);
		if(isNaN(id)) {
			console.error("id is somehow NaN", e.id)
			continue
		}
		let checked = document.getElementById(`pb-test-${id}`).checked;
		if(checked) {
			data.tests.push(id);
		}
	}

	console.log(data);
	let res = await bundled.bodyCall("/problem/__PBID__/update/addSubTask", data)
	if(res.status === "success") {
		window.location.reload();
		return;
	}
	bundled.apiToast(res);
}
document.getElementById("subtask-create").addEventListener("submit", createSubTask);
