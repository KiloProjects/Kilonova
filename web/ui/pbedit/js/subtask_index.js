var pbid = __PBID__, rowreg = /row-subtask-([0-9]+)/;
var scores = document.querySelectorAll("[id^='score-subtask-']");
for(let e of scores) {
	e.addEventListener('input', updateFinalScore);
}
var precision = __PRECISION__

function updateFinalScore() {
	var score = 0;
	var defaultPoints = __DEFAULT_POINTS__;
	score += defaultPoints;
	for(let e of scores) {
		score += Number(e.value);
	}
	let suffix = "";
	if(defaultPoints > 0) {
		suffix = ` (${defaultPoints} ${bundled.getText('byDefault')})`;
	}
    score = score.toFixed(precision)
    
	if(score != 100) {
		document.getElementById("scoreOutput").innerHTML = `<span class='text-red-600 dark:text-red-400'>${score}</span>` + suffix;	
	} else {
		document.getElementById("scoreOutput").innerHTML = score + suffix;
	}
}

const mgr = new bundled.CheckboxManager(document.getElementById("selectAllBox"), document.querySelectorAll("[id^='pb-subtask-']"))
updateFinalScore();

async function deleteSubTasks() {
	var subTasks = [];
	for(let e of document.querySelectorAll("[id^='row-subtask-']")) {
		let id = parseInt(rowreg.exec(e.id)[1]);
		if(isNaN(id)) {
			console.error("id is somehow NaN", e.id)
			continue
		}
		let checked = document.getElementById(`pb-subtask-${id}`).checked;
		if(checked) {
			subTasks.push(id);
		}
	}
    if(subTasks.length == 0) {
        return;
    }
    if(subTasks.length == 1) {
        if(!(await bundled.confirm(bundled.getText("oneSubtaskDeleteConfirm")))) {
            return
        }
    } else if(!(await bundled.confirm(bundled.getText("subtaskDeleteConfirm", subTasks.length)))) {
        return
    }
	let res = await bundled.bodyCall(`/problem/${pbid}/update/bulkDeleteSubTasks`, subTasks);
	if(res.status === "success") {
		window.location.reload();
		return;
	}
	bundled.apiToast(res);
}

async function updateSubTasks() {
	var subTasks = {};
	for(let e of document.querySelectorAll("[id^='row-subtask-']")) {
		let id = parseInt(rowreg.exec(e.id)[1]);
		if(isNaN(id)) {
			console.error("id is somehow NaN", e.id)
			continue
		}
		let score = document.getElementById(`score-subtask-${id}`).value;
		subTasks[id] = Number(score);
	}
	console.log(subTasks);
	let res = await bundled.bodyCall(`/problem/${pbid}/update/bulkUpdateSubTaskScores`, subTasks);
	if(res.status === "success") {
		window.location.reload();
		return;
	}
	bundled.apiToast(res);
}

