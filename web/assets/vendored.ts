import CopyButtonPlugin from "highlightjs-copy";

import htmx from "htmx.org";
window.htmx = htmx;

import Idiomorph from "idiomorph";
// Copy-pasted from idiomorph-htmx.js, since it errors out for whatever reason
function createMorphConfig(swapStyle) {
	if (swapStyle === "morph" || swapStyle === "morph:outerHTML") {
		return { morphStyle: "outerHTML" };
	} else if (swapStyle === "morph:innerHTML") {
		return { morphStyle: "innerHTML" };
	} else if (swapStyle.startsWith("morph:")) {
		return Function("return (" + swapStyle.slice(6) + ")")();
	}
}
// @ts-expect-error
htmx.defineExtension("morph", {
	isInlineSwap: function (swapStyle) {
		let config = createMorphConfig(swapStyle);
		return config.swapStyle === "outerHTML" || config.swapStyle == null;
	},
	handleSwap: function (swapStyle, target, fragment) {
		let config = createMorphConfig(swapStyle);
		if (config) {
			// @ts-expect-error
			return Idiomorph.morph(target, fragment.children, config);
		}
	},
});

import "katex/contrib/copy-tex";

document.addEventListener("DOMContentLoaded", () => {
	const x = new CopyButtonPlugin();
	document.querySelectorAll("pre.chroma code").forEach((val) => {
		x["after:highlightElement"]({ el: val, text: (val as HTMLPreElement).innerText.replaceAll("\n\n", "\n") });
	});
	document.querySelectorAll(".statement-content pre:not(.chroma) code").forEach((val) => {
		x["after:highlightElement"]({ el: val, text: (val as HTMLPreElement).innerText.replaceAll("\n\n", "\n").trimEnd() });
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
