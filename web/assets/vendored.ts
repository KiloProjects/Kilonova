import CopyButtonPlugin from "highlightjs-copy";
import htmx from "htmx.org";
import {Idiomorph} from 'idiomorph';
import * as Sentry from "@sentry/browser";
import "katex/contrib/copy-tex";
import CodeMirror from "codemirror";
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

(() => {
	var platformInfo = JSON.parse(document.getElementById("platform_info")?.textContent ?? "{}");
	if (typeof platformInfo.sentryDSN == 'string') {
		console.log("starting sentry", platformInfo.sentryDSN)
		Sentry.init({
			dsn: platformInfo.sentryDSN,
			sendDefaultPii: true,
			debug: true,
			environment: platformInfo.debug ? "development" : "production",
			release: platformInfo.internalVersion ?? "unknown",
			ignoreErrors: ["visitor-analytics", "twipla", "va-endpoint", "domPath are required"],
		})
	}
})()

function createMorphConfig(swapStyle) {
	if (swapStyle === "morph" || swapStyle === "morph:outerHTML") {
		return {
			morphStyle: "outerHTML",
		};
	} else if (swapStyle === "morph:innerHTML") {
		return {morphStyle: "innerHTML"};
	} else if (swapStyle.startsWith("morph:")) {
		return Function("return (" + swapStyle.slice(6) + ")")();
	}
}

// @ts-ignore
htmx.defineExtension("morph", {
	isInlineSwap: function (swapStyle) {
		let config = createMorphConfig(swapStyle);
		return config?.morphStyle === "outerHTML" || config?.morphStyle == null;
	},
	handleSwap: function (swapStyle, target, fragment) {
		let config = createMorphConfig(swapStyle);
		if (config) {
			config.callbacks = config.callbacks || {};
			config.callbacks.beforeAttributeUpdated = (attName: string, node: Element, mutationType: 'update' | 'remove'): boolean => {
				if (typeof node == 'undefined') return true;
				return !(node.nodeName == "DETAILS" && attName == "open")
			}
			return Idiomorph.morph(target, fragment.children, config);
		}
	},
});

window.htmx = htmx;

function initialVendoredLoad(el: Element) {
	const x = new CopyButtonPlugin();
	el.querySelectorAll("pre.chroma code").forEach((val) => {
		x["after:highlightElement"]({el: val, text: (val as HTMLElement).innerText.replaceAll("\n\n", "\n")});
		val.parentElement?.querySelector(".hljs-copy-container")?.style.setProperty("--hljs-theme-padding", "16px");
	});
	el.querySelectorAll(".statement-content pre:not(.chroma) code").forEach((val) => {
		x["after:highlightElement"]({el: val, text: (val as HTMLElement).innerText.replaceAll("\n\n", "\n").trimEnd()});
		val.parentElement?.querySelector(".hljs-copy-container")?.style.setProperty("--hljs-theme-padding", "16px");
	});
}


document.addEventListener("DOMContentLoaded", () => {
	initialVendoredLoad(document.documentElement);
});

htmx.on(("htmx:afterSwap"), (el) => {
	if (el.target != null) {
		initialVendoredLoad((<Element>el.target));
	}
});

window.CodeMirror = CodeMirror;
window.CodeMirror.defaults.lineNumbers = true;
window.CodeMirror.defaults.indentUnit = 4;
window.CodeMirror.defaults.tabSize = 4;
window.CodeMirror.defaults.indentWithTabs = true;
window.CodeMirror.defaults.lineWrapping = true;
window.CodeMirror.defaults.viewportMargin = Infinity;

// import "codemirror/mode/stex/stex";
