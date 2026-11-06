var pbid = __PBID__, rowreg = /row-test-([0-9]+)/;
var scores = document.querySelectorAll("[id^='score-test-']");
for(let e of scores) {
	e.addEventListener('input', updateFinalScore);
}
var precision = __PRECISION__

const mgr = new bundled.CheckboxManager(document.getElementById("selectAllBox"), document.querySelectorAll("[id^='pb-test-']"))

function updateFinalScore() {
    if(document.getElementById("scoreOutput") === null) {
        return
    }

    var score = 0;
	var defaultPoints = __DEFAULT_POINTS__;
	score += defaultPoints;
	for(let e of scores) {
		score += Number(e.value);
	}
	let suffix = "";
	if(defaultPoints > 0) {
		suffix = ` (${defaultPoints} ${bundled.getText("byDefault")})`;
	}
    score = score.toFixed(precision)
    
	if(score != 100) {
		document.getElementById("scoreOutput").innerHTML = `<span class='text-red-600 dark:text-red-400'>${score}</span>` + suffix;	
	} else {
		document.getElementById("scoreOutput").innerHTML = score + suffix;
	}
}


updateFinalScore();

async function deleteTests() {
	var tests = [];
	for(let e of document.querySelectorAll("[id^='row-test-']")) {
		let id = parseInt(rowreg.exec(e.id)[1]);
		if(isNaN(id)) {
			console.error("id is somehow NaN", e.id)
			continue
		}
		let checked = document.getElementById(`pb-test-${id}`).checked;
		if(checked) {
			tests.push(id);
		}
	}
    if(tests.length == 0) {
        return;
    }
    if(tests.length == 1) {
        if(!(await bundled.confirm(bundled.getText("oneTestDeleteConfirm")))) {
            return
        }
    } else if(!(await bundled.confirm(bundled.getText("testDeleteConfirm", tests.length)))) {
        return
    }
	let res = await bundled.bodyCall(`/problem/${pbid}/update/bulkDeleteTests`, tests);
	if(res.status === "success") {
		window.location.reload();
		return;
	}
	bundled.apiToast(res);
}

async function updateTests() {
	var tests = {};
	for(let e of document.querySelectorAll("[id^='row-test-']")) {
        let id = parseInt(rowreg.exec(e.id)[1]);
		if(isNaN(id)) {
			console.error("id is somehow NaN", e.id)
			continue
		}
		let score = parseFloat(document.getElementById(`score-test-${id}`).value);
		if(isNaN(score)) {
			console.error("Score is somehow NaN", document.getElementById(`score-test-${id}`))
			continue
		}
		tests[id] = score;
	}
	console.log(tests);
	let res = await bundled.bodyCall(`/problem/${pbid}/update/bulkUpdateTestScores`, tests);
	bundled.apiToast(res);
}

async function uploadTests(e) {
	e.preventDefault()
	var form = new FormData();
	var files = document.getElementById("tests").files;
	if(files === null || files.length === 0) {
		bundled.createToast({status: "error", title: __NO_FILES__})
		return
	}
	form.append("testArchive", files[0]);
    const scoreParams = document.getElementById("scoreParams").value;
    if(scoreParams.length > 0) {
        form.append("scoreParameters", scoreParams);
    }
	form.append("forceStringIDs", document.getElementById("forceStringIDs").checked ? "true" : "false");

	let res = await bundled.multipartProgressCall("/problem/__PBID__/update/processTestArchive", form)
	if(res.status === "success") {
		window.location.reload();
		return
	}
	bundled.apiToast(res)
}

document.getElementById("test_add_form").addEventListener("submit", uploadTests)
