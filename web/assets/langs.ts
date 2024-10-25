export const languages = {
	c: "text/x-csrc",
	cpp: "text/x-c++src",
	cpp11: "text/x-c++src",
	cpp14: "text/x-c++src",
	cpp17: "text/x-c++src",
	cpp20: "text/x-c++src",
	golang: "text/x-go",
	haskell: "text/x-haskell",
	java: "text/x-java",
	kotlin: "text/x-kotlin",
	nodejs: "text/javascript",
	python3: "text/x-python",
	pascal: "text/x-pascal",
	php: "text/x-php",
	rust: "text/rust",
	outputOnly: "text/plain",
};

export const prettyLanguages = {
	c: "C",
	cpp: "C++",
	cpp11: "C++11",
	cpp14: "C++14",
	cpp17: "C++17",
	cpp20: "C++20",
	golang: "Go",
	haskell: "Haskell",
	java: "Java",
	kotlin: "Kotlin",
	nodejs: "Node.js",
	python3: "Python 3",
	pascal: "Pascal",
	php: "PHP",
	rust: "Rust",
	outputOnly: "Output Only",
};

type CMMode = {
	mimeType: string;
	prettyName: string;
	extensions: string[];
};

export const cm_modes: { [name: string]: CMMode } = {
	c: {
		mimeType: "text/x-csrc",
		prettyName: "C",
		extensions: ["c"],
	},
	cpp: {
		mimeType: "text/x-c++src",
		prettyName: "C++",
		extensions: ["cpp", "cxx", "cpp11", "cpp14", "cpp17", "cpp20", "h", "hpp", "hxx"],
	},
	golang: {
		mimeType: "text/x-go",
		prettyName: "Golang",
		extensions: ["go"],
	},
	haskell: {
		mimeType: "text/x-haskell",
		prettyName: "Haskell",
		extensions: ["hs", "lhs"],
	},
	java: {
		mimeType: "text/x-java",
		prettyName: "Java",
		extensions: ["java"],
	},
	kotlin: {
		mimeType: "text/x-kotlin",
		prettyName: "Kotlin",
		extensions: ["kt"],
	},
	nodejs: {
		mimeType: "text/javascript",
		prettyName: "Node.js",
		extensions: ["js"],
	},
	python: {
		mimeType: "text/x-python",
		prettyName: "Python",
		extensions: ["py", "py3"],
	},
	pascal: {
		mimeType: "text/x-pascal",
		prettyName: "Pascal",
		extensions: ["pas"],
	},
	php: {
		mimeType: "text/x-php",
		prettyName: "PHP",
		extensions: ["php"],
	},
	rust: {
		mimeType: "text/rust",
		prettyName: "Rust",
		extensions: ["rs"],
	},
	markdown: {
		mimeType: "text/markdown",
		prettyName: "Markdown",
		extensions: ["md"],
	},
	plaintext: {
		mimeType: "text/plain",
		prettyName: "Plain text",
		extensions: ["txt"],
	},
};

export function get_cm_mode(filename: string): string {
	let ext = filename.split(".").pop();
	if (typeof ext === "undefined") {
		return "text/plain";
	}
	for (let opt of Object.values(cm_modes)) {
		if (opt.extensions.includes(ext)) {
			return opt.mimeType;
		}
	}
	return "text/plain";
}
