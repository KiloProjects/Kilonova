package web

type Translation map[string]string
type Translations map[string]Translation

var defaultLang = "en"

var translations = Translations{
	"announcement": genTranslation("Pssst! If you have any suggestions for the website, send me a message on %s.", "Pssst! Dacă ai vreo sugestie pentru site, trimite-mi mesaj pe %s."),
	"num_solved":   genTranslation("(%d / %d solved)", "(%d / %d rezolvate)"),

	"problems":        genTranslation("Problems", "Probleme"),
	"problem":         genTranslation("Problem %d: %s", "Problema %d: %s"),
	"problemSingle":   genTranslation("Problem", "Problema"),
	"published":       genTranslation("Published", "Publicată"),
	"unpublished":     genTranslation("Unpublished", "Nepublicată"),
	"unverifiedEmail": genTranslation(`It looks like you haven't verified your email yet. Click <a class="underline text-black dark:text-white" href="/verify/resend">here</a> if you need to have your verification resent.`, `Se pare că încă nu ți-ai verificat e-mailul. Apasă <a class="underline text-black dark:text-white" href="/verify/resend">aici</a> dacă trebuie să reprimești verificarea.`),
	"noProblemLists":  genTranslation("There are no problem lists", "Nu există nicio listă de probleme"),
	"noListProblems":  genTranslation("There are no visible problems in the list", "Nu există nicio problemă vizibilă în lista de probleme"),
	"create":          genTranslation("[create]", "[creare]"),

	"title": genTranslation("Title", "Titlu"),
	"name":  genTranslation("Name", "Nume"),
	"desc":  genTranslation("Description", "Descriere"),
	"pbs":   genTranslation("Problems (IDs separated by a comma)", "Probleme (ID-uri separate prin virgulă"),

	"createButton": genTranslation("Create", "Creare"),
	"update":       genTranslation("Update", "Actualizare"),
	"add":          genTranslation("Add", "Adăugare"),
	"delete":       genTranslation("Delete", "Ștergere"),
	"upload":       genTranslation("Upload", "Încărcare"),

	// Titles
	"listCreate":      genTranslation("Create List", "Creare Listă"),
	"lists":           genTranslation("Lists", "Liste"),
	"listKNA":         genTranslation("[Download problem archive]", "[Descărcare listă cu probleme]"),
	"list":            genTranslation("List #%d", "Lista #%d"),
	"listUpdate":      genTranslation("Update List", "Actualizare Listă"),
	"profile":         genTranslation("Profile of", "Profilul lui"),
	"emailConfirm":    genTranslation("Verified Email", "Email Verificat"),
	"editIndex":       genTranslation("Edit Problem | Problem #%d: %s", "Editare Problemă | Problema #%d: %s"),
	"editDesc":        genTranslation("Edit Description | Problem #%d: %s", "Editare Descriere | Problema #%d: %s"),
	"editTestAdd":     genTranslation("Add Test | Problem #%d: %s", "Adăugare Test | Problema #%d: %s"),
	"editTestEdit":    genTranslation("Edit Test #%d | Problem #%d: %s", "Editare Test #%d | Problema #%d: %s"),
	"editAttachments": genTranslation("Edit Attachments | Problem #%d: %s", "Editare Atașamente | Problema #%d: %s"),
	"editSubtaskAdd":  genTranslation("Create Subtask", "Creare Subtask"),
	"editSubtaskEdit": genTranslation("Edit Subtask #%d", "Editare Subtask #%d"),
	"editSubtask":     genTranslation("Update SubTasks", "Actualizare SubTasks"),

	// TODO: might delete it
	"editChecker": genTranslation("Edit Checker | Problem #%d: %s", "Editare Verificator | Problema #%d: %s"),

	// User page
	"verifiedEmail": genTranslation("Verified Email", "Email Verificat"),
	"profilePic":    genTranslation("Profile picture for", "Poză profil pentru"),

	"email":  genTranslation("Email", "Email"),
	"yes":    genTranslation("Yes", "Da"),
	"no":     genTranslation("No", "Nu"),
	"resend": genTranslation("Resend", "Retrimite"),
	"send":   genTranslation("Send", "Trimite"),

	"editBio":   genTranslation("Edit Bio", "Editare Bio"),
	"createBio": genTranslation("Create Bio", "Creare Bio"),

	"oneSolvedProblem": genTranslation("One solved problem:", "O problemă rezolvată:"),
	"solvedProblems":   genTranslation("%d solved problems", "%d probleme rezolvate"),

	// admin
	"deleteAccount":      genTranslation("Prune User", "Ștergere Profil"),
	"delAccConfirmation": genTranslation("Are you sure you want to prune the user?", "Sunteți siguri că vreți să ștergeți utilizatorul?"),

	// email confirmation
	"confirmedEmail": genTranslation("Email confirmed for %s", "Email confirmat pentru %s"),
	"closeTab":       genTranslation("You may close the tab", "Puteți să închideți tabul"),

	// email resent
	"resentEmail": genTranslation("Resent email", "Email retrimis"),

	// footer
	"usefulInfo": genTranslation("Useful Links", "Link-uri utile"),
	"otherInfo":  genTranslation("Other information", "Alte informații"),

	"frontPage":      genTranslation("Front page", "Pagina principală"),
	"listOfProblems": genTranslation("List of Problems", "Listă Probleme"),
	"subList":        genTranslation("Submission List", "Listă submisii"),

	// navbar
	"profileLink":   genTranslation("Profile", "Profil"),
	"settings":      genTranslation("Settings", "Setări"),
	"proposerPanel": genTranslation("Proposer Panel", "Panou Propunător"),
	"adminPanel":    genTranslation("Admin Panel", "Panou Admin"),
	"problemLists":  genTranslation("Problem Lists", "Liste de Probleme"),

	"signUp": genTranslation("Sign Up", "Înregistrare"),
	"logIn":  genTranslation("Log In", "Logare"),
	"logOut": genTranslation("Log Out", "Log Out"),

	// topbar
	"problemEdit":   genTranslation("Editing Problem", "Editare Problemă"),
	"general":       genTranslation("General", "General"),
	"attachments":   genTranslation("Attachments", "Atașamente"),
	"tests":         genTranslation("Tests", "Teste"),
	"subTasks":      genTranslation("SubTasks", "SubTaskuri"),
	"createTest":    genTranslation("Create Test", "Creare Test"),
	"createSubTask": genTranslation("Create SubTask", "Creare SubTask"),
	"nthTest":       genTranslation("Test #%d", "Testul #%d"),
	"nthSubTask":    genTranslation("SubTask #%d", "SubTaskul #%d"),

	// edit/*.html
	"author":  genTranslation("Author", "Autor"),
	"source":  genTranslation("Source", "Sursă"),
	"pbType":  genTranslation("Problem Type", "Tip Problemă"),
	"classic": genTranslation("Classic", "Clasic"),
	"checker": genTranslation("Checker", "Verificator"),

	"visiblePb":     genTranslation("Visible problem", "Problemă vizibilă"),
	"consoleInput":  genTranslation("Console input", "Intrare din consolă"),
	"testName":      genTranslation("Test name", "Nume test"),
	"memoryLimit":   genTranslation("Memory limit", "Limită de memorie"),
	"stackLimit":    genTranslation("Stack limit", "Limită de stack"),
	"timeLimit":     genTranslation("Time limit", "Limită de timp"),
	"seconds":       genTranslation("seconds", "secunde"),
	"defaultPoints": genTranslation("Points by default", "Puncte din oficiu"),

	"id":    genTranslation("ID", "ID"),
	"score": genTranslation("Score", "Scor"),

	"updateProblem": genTranslation("Update problem info", "Actualizare date problemă"),
	"deleteProblem": genTranslation("Delete problem", "Șterge problema"),

	"deleteTests":      genTranslation("Delete selected tests", "Șterge testele selectate"),
	"updateTestScores": genTranslation("Update test scores", "Actualizează scorurile testelor"),
	"byDefault":        genTranslation("by default", "din oficiu"),

	"testArchive": genTranslation("Upload .zip archive with tests", "Încărcare arhivă .zip cu teste"),
	"archive":     genTranslation("Archive", "Arhivă"),
	"noFiles":     genTranslation("No files selected", "Niciun fișier specificat"),

	"testID":       genTranslation("Test ID", "ID Test"),
	"noTests":      genTranslation("There's no tests, add one first!", "Nu există niciun test, adaugă unul întâi!"),
	"noTestsError": genTranslation("There's no tests, this should not happen!", "Nu există niciun tests, așa ceva n-ar trebui să se întâmple!"),
	"emptySubTask": genTranslation("WARNING: Empty SubTask, will always show 0 points", "WARNING: SubTask gol, va afișa mereu 0 puncte"),

	"deleteSubTasks": genTranslation("Delete selected SubTasks", "Șterge SubTaskuri selectate"),
	"updateSubTasks": genTranslation("Update SubTask scores", "Actualizare scoruri SubTasks"),
	"noSubTasks":     genTranslation("There are no SubTasks, tests will be scored individually", "Nu există niciun SubTask, testele se vor evalua individual"),

	"testAssociations": genTranslation("Test Associations", "Asocieri teste"),

	"file":      genTranslation("File", "Fișier"),
	"size":      genTranslation("Size", "Dimensiune"),
	"visible":   genTranslation("Visible", "Vizibil"),
	"invisible": genTranslation("Invisible", "Invizibil"),
	"private":   genTranslation("Private", "Privat"),

	"deleteAttachments": genTranslation("Delete selected attachments", "Șterge atașamentele selectate"),
	"noAttachments":     genTranslation("There are no attachments for the problem", "Nu există vreun atașament asociat problemei"),

	"emptyTitle":           genTranslation("Empty title", "Titlu gol"),
	"confirmProblemDelete": genTranslation("Are you sure you want to delete the problem?", "Sigur vreți să ștergeți problema?"),
	"confirmTestDelete":    genTranslation("Are you sure you want to delete the test?", "Sunteți sigur că vreți să ștergeți testul?"),
	"checkerWarning":       genTranslation("WARNING: The problem doesn't use a checker. You can set one, but it will not be used", "WARNING: Problema nu utilizează un verificator. Puteți să setați unul, însă nu va fi folosit"),

	// util/statusCode.html
	"errorMessage":    genTranslation("That's an error", "Asta-i o eroare!"),
	"errorLogin":      genTranslation("Maybe you can log in to view this", "Probabil vei putea să accesezi dacă te loghezi"),
	"errorLoginError": genTranslation("Can't get login modal", "N-am putut afișa modalul de autentificare"),

	// pb.html
	"edit":         genTranslation("[edit]", "[editare]"),
	"generalInfo":  genTranslation("General info", "Informații generale"),
	"uploader":     genTranslation("Uploader", "Uploader"),
	"input":        genTranslation("Input", "Intrare"),
	"console":      genTranslation("Console Input", "Consolă"),
	"visibility":   genTranslation("Visibility", "Vizibilitate"),
	"submissions":  genTranslation("Submissions", "Submisii"),
	"qualitySubs":  genTranslation("Special Submissions", "Submisii Evidențiate"),
	"oldSubs":      genTranslation("Older submissions", "Submisii anterioare"),
	"loading":      genTranslation("Loading...", "Se încarcă..."),
	"noSub":        genTranslation("No submissions", "Nicio submisie"),
	"seeMore":      genTranslation("See ${hidden} more submission${hidden==1?'':'s'}", "Vezi încă ${hidden} ${hidden>10 ? 'de' : ''} submisii"),
	"language":     genTranslation("Language", "Limbaj"),
	"uploadSub":    genTranslation("Upload submission", "Încărcare submisie"),
	"finishedEval": genTranslation("Submission evaluated", "Evaluare finalizată"),
	"finalScore":   genTranslation("Score for submission #${id}", "Scor submisie #${id}"),
	"view":         genTranslation("View", "Vizualizare"),
	"uploaded":     genTranslation("Submission uploaded", "Submisie încărcată"),

	"attachmentPretty": genTranslation("I know this needs to be prettier", "Știu că trebuie înfrumusețat"),

	"removeSub":  genTranslation("Remove submission", "Ștergere submisie"),
	"reevaluate": genTranslation("Reevaluate submission", "Reevaluare submisie"),
	"sub":        genTranslation("Submission", "Submisia"),

	// submissions.html
	"subStatus": genTranslation("Status of submissions", "Stare submisii"),

	"page":      genTranslation("Page", "Pagină"),
	"userID":    genTranslation("User ID", "ID User"),
	"problemID": genTranslation("Problem ID", "ID Problemă"),
	"status":    genTranslation("Status", "Status"),

	"finished":   genTranslation("Finished", "Finalizat"),
	"working":    genTranslation("Working", "În lucru"),
	"waiting":    genTranslation("Waiting", "În așteptare"),
	"special":    genTranslation("Special", "Evidențiată"),
	"compileErr": genTranslation("Compile error", "Eroare de compilare"),
	"sorting":    genTranslation("Sorting criteria", "Criteriu de sortare"),
	"maxTime":    genTranslation("Max Time", "Timp Maxim"),
	"maxMem":     genTranslation("Max Memory", "Memorie Maximă"),
	"ascending":  genTranslation("Ascending Sort", "Criteriu Crescător"),

	"fetch": genTranslation("Fetch data", "Filtrare"),

	"time":       genTranslation("Time", "Timp"),
	"memory":     genTranslation("Memory", "Memorie"),
	"uploadDate": genTranslation("Upload date", "Dată încărcare"),

	"noSubFound": genTranslation("No submission found", "Nicio submisie găsită"),

	"show":       genTranslation("Show", "Afișare"),
	"hide":       genTranslation("Hide", "Ascunde"),
	"filterBar":  genTranslation("filter bar", "bara de filtrare"),
	"filterLink": genTranslation("Copy filter link", "Copiere link filtre"),

	// settings
	"settingsHeader": genTranslation("Profile settings", "Setările profilului"),
	"updateBio":      genTranslation("Update Bio", "Actualizare Bio"),
	"makeSubs":       genTranslation("Make submissions implicitly", "Fă submisiile implicit"),

	"password":      genTranslation("Password", "Parolă"),
	"passwordCheck": genTranslation("Check Password", "Verificare Parolă"),
	"newEmail":      genTranslation("New email", "Email nou"),

	"newPwd":        genTranslation("New password", "Parolă nouă"),
	"newPwdConfirm": genTranslation("Confirm new password", "Confirmare parolă nouă"),

	"differentPwds": genTranslation("Passwords do not match", "Parolele nu se potrivesc"),

	"updatePwd":   genTranslation("Update password", "Actualizare parolă"),
	"updateEmail": genTranslation("Update email", "Actualizare email"),

	// proposer panel
	"createPb":    genTranslation("Create problem", "Creare problemă"),
	"problemName": genTranslation("Problem name", "Nume problemă"),

	// auth
	"username":       genTranslation("Username", "Username"),
	"signupReminder": genTranslation(`Don't have an account? <a href="/signup">Sign Up</a>`, `N-ai cont? <a href="/signup">Înregistrează-te</a>`),
	"loginReminder":  genTranslation(`Already have an account? <a href="/login">Log In</a>`, `Ai deja cont? <a href="/login">Loghează-te</a>`),
	"authenticate":   genTranslation("Authenticate", "Autentificare"),

	// admin
	"resetSubs":    genTranslation("Reset waiting submissions", "Resetare submisii în așteptare"),
	"changeRoles":  genTranslation("Change roles", "Schimbă rolurile"),
	"makeAdmin":    genTranslation("Make Admin", "Fă Administrator"),
	"makeProposer": genTranslation("Make Proposer", "Fă Propunător"),
	"admins":       genTranslation("Admins", "Administratori"),
	"proposers":    genTranslation("Proposers", "Propunători"),

	"mainPageAdmin":  genTranslation("Manage main page", "Administrare pagina principală"),
	"mainPagePbList": genTranslation("Problem list (IDs, comma-separated)", "Listă de probleme (IDuri, separate prin o virgulă)"),
	"mainPageAllPbs": genTranslation("Show all problems", "Afișare toate problemele"),
	"description":    genTranslation("Description", "Descriere"),

	"knaAdmin":      genTranslation("Manage .kna archives", "Administrare arhive .kna"),
	"createArchive": genTranslation("Create Archive", "Creare Arhivă"),
	"authorID":      genTranslation("Author ID", "ID Autor"),
}

func genTranslation(en string, ro string) Translation {
	return Translation{
		"en": en,
		"ro": ro,
	}
}
