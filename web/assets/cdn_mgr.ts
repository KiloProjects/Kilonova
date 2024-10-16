let extensionIcons = new Map<string, string>();

for (let t of ["png", "jpg", "jpeg", "gif"]) {
	extensionIcons.set(t, "fa-file-image");
}

for (let t of ["mp4", "mkv"]) {
	extensionIcons.set(t, "fa-file-video");
}

extensionIcons.set("pdf", "fa-file-pdf");
extensionIcons.set("csv", "fa-file-csv");
extensionIcons.set("ppt", "fa-file-powerpoint");
extensionIcons.set("pptx", "fa-file-powerpoint");
extensionIcons.set("xls", "fa-file-excel");
extensionIcons.set("xlsx", "fa-file-excel");
extensionIcons.set("doc", "fa-file-word");
extensionIcons.set("docx", "fa-file-word");

for (let t of ["zip", "rar", "7z", "gz", "tar"]) {
	extensionIcons.set(t, "fa-file-archive");
}

for (let t of ["c", "cpp", "go", "pas", "py", "py3", "java", "js", "kt"]) {
	extensionIcons.set(t, "fa-file-code");
}

for (let t of ["md", "html", "txt"]) {
	extensionIcons.set(t, "fa-file-lines");
}

export function getFileIcon(name: string): string {
	let ext = name.trim().split(".").pop();
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
