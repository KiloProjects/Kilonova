
const languageStrings = {
	// subs_view.js
	"copyErr": genTranslation("Couldn't copy to clipboard", "Nu am putut copia în clipboard"),
	"nResults": genTranslation((n) => {
		if(n == 1) { return "One result" }
		return `${n} results`
	}, (n) => {
		if(n == 1) { return "Un rezultat" }
		if(n < 20) { return `${n} rezultate` }
		return `${n} de rezultate`
	}),
	"evaluated": genTranslation("Evaluated", "Evaluat"),
	"points": genTranslation("points", "puncte"),
	"evaluating": genTranslation("Evaluating...", "În evaluare..."),

	// sub_mgr.js
	"sub": genTranslation("Submission", "Submisia"),
	"author": genTranslation("Author", "Autor"),
	"problem": genTranslation("Problem", "Problemă"),
	"uploadDate": genTranslation("Upload Date", "Data Încărcării"),
	"status": genTranslation("Status", "Status"),
	"lang": genTranslation("Language", "Limbaj"),
	"qualitySub": genTranslation("Special submission", "Submisie evidențiată"),
	"size": genTranslation("Size", "Mărime"),
	"defaultPoints": genTranslation("Points by default", "Puncte din oficiu"),
	"score": genTranslation("Score", "Scor"),
	"compileErr": genTranslation("Compile Error", "Eroare de Compilare"),
	"compileMsg": genTranslation("Compilation Message", "Mesaj Compilare"),
	"copied": genTranslation("Copied to clipboard", "Copiat în clipboard"),
	"test": genTranslation("Test", "Testul"),
	"subTask": genTranslation("SubTask", "SubTask-ul"),
	"seeTests": genTranslation("See individual tests", "Vizualizare teste individuale"),
	"id": genTranslation("ID", "ID"),
	"time": genTranslation("Time", "Timp"),
	"memory": genTranslation("Memory", "Memorie"),
	"verdict": genTranslation("Verdict", "Verdict"),
	"output": genTranslation("Output", "Ieșire"),
	"subTasks": genTranslation("SubTasks", "SubTasks"),
	"waiting": genTranslation("Waiting...", "În așteptare..."),
	"correct": genTranslation("correct", "corect"),

	"source": genTranslation("Source Code:", "Codul Sursă:"),
	"copy": genTranslation("Copy", "Copiere"),
	"download": genTranslation("Download", "Descărcare"),

	"maxTime": genTranslation("Max Time", "Timp maxim"),
	"maxMemory": genTranslation("Max Memory", "Memorie maximă"),
	
	"makeCode": genTranslation("Make code", "Fă codul"),
	"visible": genTranslation("visible", "vizibil"),
	"invisible": genTranslation("invisible", "invisible"),

	"makeQuality": genTranslation("Make Special", "Evidențiere"),
	"dropQuality": genTranslation("Remove Special Status", "Scoatere Evidențiere"),
}

export function getText(lang, key) {
	if(key in languageStrings) {
		if(lang in languageStrings[key]) {
			return languageStrings[key][lang]
		}
		console.error("Language", lang, "not found in key", key)
	}
	console.error("Unknown key", key)
}

function genTranslation(en, ro) {
	return {'en': en, 'ro': ro}
}

