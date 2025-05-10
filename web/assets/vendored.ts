import CopyButtonPlugin from "highlightjs-copy";
import htmx from "htmx.org";
window.htmx = htmx;
import "idiomorph/dist/idiomorph-ext.esm.js";

import "katex/contrib/copy-tex";

document.addEventListener("DOMContentLoaded", () => {
	const x = new CopyButtonPlugin();
	document.querySelectorAll("pre.chroma code").forEach((val) => {
		x["after:highlightElement"]({ el: val, text: (val as HTMLElement).innerText.replaceAll("\n\n", "\n") });
		val.parentElement?.querySelector(".hljs-copy-container")?.style.setProperty("--hljs-theme-padding", "16px");
	});
	document.querySelectorAll(".statement-content pre:not(.chroma) code").forEach((val) => {
		x["after:highlightElement"]({ el: val, text: (val as HTMLElement).innerText.replaceAll("\n\n", "\n").trimEnd() });
		val.parentElement?.querySelector(".hljs-copy-container")?.style.setProperty("--hljs-theme-padding", "16px");
	});
});

import CodeMirror from "codemirror";
window.CodeMirror = CodeMirror;
window.CodeMirror.defaults.lineNumbers = true;
window.CodeMirror.defaults.indentUnit = 4;
window.CodeMirror.defaults.tabSize = 4;
window.CodeMirror.defaults.indentWithTabs = true;
window.CodeMirror.defaults.lineWrapping = true;
window.CodeMirror.defaults.viewportMargin = Infinity;

import "codemirror/addon/mode/overlay";
import "codemirror/mode/meta";
import "codemirror/mode/markdown/markdown";
import "codemirror/mode/gfm/gfm";
import "codemirror/mode/clike/clike";
import "codemirror/mode/pascal/pascal";
import "codemirror/mode/go/go";
import "codemirror/mode/python/python";
import "codemirror/mode/haskell/haskell";
import "codemirror/mode/javascript/javascript";
import "codemirror/mode/php/php";
import "codemirror/mode/rust/rust";
// import "codemirror/mode/stex/stex";
