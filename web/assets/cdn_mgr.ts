let extensionIcons = new Map<string, string>();

for (let t of ["png", "jpg", "jpeg", "gif"]) {
	extensionIcons[t] = "fa-file-image";
}

for (let t of ["mp4", "mkv"]) {
	extensionIcons[t] = "fa-file-video";
}

extensionIcons["pdf"] = "fa-file-pdf";
extensionIcons["csv"] = "fa-file-csv";
extensionIcons["ppt"] = "fa-file-powerpoint";
extensionIcons["pptx"] = "fa-file-powerpoint";
extensionIcons["xls"] = "fa-file-excel";
extensionIcons["xlsx"] = "fa-file-excel";
extensionIcons["doc"] = "fa-file-word";
extensionIcons["docx"] = "fa-file-word";

for (let t of ["zip", "rar", "7z", "gz", "tar"]) {
	extensionIcons[t] = "fa-file-archive";
}

for (let t of ["c", "cpp", "go", "pas", "py", "py3", "java", "js"]) {
	extensionIcons[t] = "fa-file-code";
}

for (let t of ["md", "html", "txt"]) {
	extensionIcons[t] = "fa-file-alt";
}

export function getFileIcon(name: string): string {
	let ext = name.split(".").pop();
	if (ext === undefined) {
		return "fa-file";
	}
	if (ext.includes("cpp")) {
		ext = ext.replace(/[0-9]+$/, "");
	}
	if (extensionIcons.has(ext)) {
		return extensionIcons.get(ext)!;
	}
	return "fa-file";
}
