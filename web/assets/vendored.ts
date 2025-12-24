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

import Alpine from "alpinejs";

window.Alpine = Alpine;
Alpine.start();

(() => {
	var platformInfo: PlatformInfo = JSON.parse(document.getElementById("platform_info")?.textContent ?? "{}");
	if (typeof platformInfo.sentryDSN == 'string') {
		console.log("starting sentry", platformInfo.sentryDSN)
		Sentry.init({
			dsn: platformInfo.sentryDSN,
			sendDefaultPii: true,
			debug: true,
			environment: platformInfo.debug ? "development" : "production",
			release: platformInfo.internalVersion ?? "unknown",
			ignoreErrors: ["visitor-analytics", "twipla", "va-endpoint", "domPath are required", "Server is under maintenance"],
		})
		Sentry.setUser({
			id: platformInfo.user_id,
			username: platformInfo.user?.name,
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

CodeMirror.defaults.lineNumbers = true;
CodeMirror.defaults.indentUnit = 4;
CodeMirror.defaults.tabSize = 4;
CodeMirror.defaults.indentWithTabs = true;
CodeMirror.defaults.lineWrapping = true;
CodeMirror.defaults.viewportMargin = Infinity;
CodeMirror.defineInitHook(bundled.CodeMirrorThemeHook);
if (bundled.isDarkMode()) {  // Dark Mode
	CodeMirror.defaults.theme = "monokai";
}
window.CodeMirror = CodeMirror;

// import "codemirror/mode/stex/stex";

export type KNEditorOptions = {
	textArea: HTMLTextAreaElement;
	language: "md" | "text" | string;

	dynamicSize?: boolean;
	autoFocus?: boolean;
}

export class KNEditor {
	cm: CodeMirror.EditorFromTextArea;
	// TODO:
	//  - Check all codemirror function calls have been adapted
	//  - Migrate to CM6
	//  - Eliminate unnecessary wrappers for get/setLanguage (modes)
	//     - Abstract away language internals

	constructor(opts: KNEditorOptions) {
		let cmSettings: CodeMirror.EditorConfiguration = {}
		if (opts.language == "md") {
			cmSettings.mode = {
				name: "gfm",
				gitHubSpice: false,
				emoji: false
			}
		} else if(opts.language == "text" || opts.language == "text/plain") {
			cmSettings.mode = "text/plain";
		} else {
			cmSettings.mode = opts.language;
		}
		this.cm = CodeMirror.fromTextArea(opts.textArea, cmSettings)

		if (opts.dynamicSize) {
			this.cm.setSize(null, "100%");
			// min-height: 250px;
			this.cm.getWrapperElement().style.minHeight = "250px";
		}
		if (opts.autoFocus) {
			this.cm.focus();
		}
	}

	getText() {
		return this.cm.getValue();
	}

	setText(content: string) {
		this.cm.setValue(content);
	}

	onChange(callback: () => void) {
		this.cm.on("change", callback);
	}

	onPaste(cb: (pasteText: string[]) => undefined | string[]) {
		this.cm.on('beforeChange', (cm, event) => {
			if(event.origin === "paste") {
				let newVal = cb(event.text);
				if(typeof newVal !== 'undefined') {
					event.update?.(undefined, undefined, newVal);
				}
			}
		})
	}

	setLanguage(lang: string) {
		this.cm.setOption("mode", lang);
	}

	getLanguage() {
		return this.cm.getOption("mode");
	}

	refresh() {
		this.cm.refresh();
	}
}
