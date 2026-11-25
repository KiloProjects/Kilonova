/// <reference types="vite/client" />

// <kn-pdf src="..."> shows a PDF with pdf.js instead of the browser's own viewer, so it
// looks and behaves the same everywhere (mobile browsers have no usable inline viewer).
// This is its own bundle, loaded only by the pages that show a PDF: see ui.PDFViewer.

// The viewer components read the core library off globalThis.pdfjsLib, which this import
// sets, so it has to come first. Both have to be the same build of the same version.
import {getDocument, GlobalWorkerOptions} from "pdfjs-dist/legacy/build/pdf.mjs";
import {EventBus, LinkTarget, PDFLinkService, PDFViewer} from "pdfjs-dist/legacy/web/pdf_viewer.mjs";
import workerSrc from "pdfjs-dist/legacy/build/pdf.worker.min.mjs?url";
import viewerCSS from "pdfjs-dist/legacy/web/pdf_viewer.css?inline";

GlobalWorkerOptions.workerSrc = workerSrc;

// Copied there by the pdfjs-runtime-assets plugin in vite.config.ts.
const runtimeAssets = import.meta.env.BASE_URL + "pdfjs/";

// pdf.js puts color-scheme and all of its custom properties on :root, which would restyle
// the whole page. Keep them on the element instead, and let it follow the site's theme.
const sheet = new CSSStyleSheet();
sheet.replaceSync(
	viewerCSS.replaceAll(":root", "kn-pdf") +
		`
kn-pdf { display: block; position: relative; height: 90vh; margin: 0.5rem 0; color-scheme: inherit; background: rgb(128 128 128 / 0.15); }
kn-pdf > .kn-pdf-scroll { position: absolute; inset: 0; overflow: auto; }
kn-pdf > .kn-pdf-toolbar { position: absolute; top: 0.5rem; right: 1.5rem; z-index: 1; display: flex; gap: 0.25rem; }
`,
);
document.adoptedStyleSheets = [...document.adoptedStyleSheets, sheet];

class KNPDF extends HTMLElement {
	private teardown?: () => void;

	connectedCallback() {
		const src = this.getAttribute("src");
		if (!src) {
			return;
		}

		// PDFViewer wants an absolutely positioned scroll container holding the pages element.
		const scroller = document.createElement("div");
		scroller.className = "kn-pdf-scroll";
		scroller.tabIndex = 0; // keyboard scrolling
		const pages = document.createElement("div");
		pages.className = "pdfViewer";
		scroller.append(pages);

		const eventBus = new EventBus();
		const linkService = new PDFLinkService({eventBus, externalLinkTarget: LinkTarget.BLANK});
		const viewer = new PDFViewer({container: scroller, viewer: pages, eventBus, linkService});
		linkService.setViewer(viewer);

		const toolbar = document.createElement("div");
		toolbar.className = "kn-pdf-toolbar";
		toolbar.append(
			zoomButton("−", "pdf.zoom_out", () => viewer.decreaseScale()),
			zoomButton("+", "pdf.zoom_in", () => viewer.increaseScale()),
		);
		this.style.height = "";
		this.replaceChildren(scroller, toolbar);

		eventBus.on("pagesinit", () => {
			viewer.currentScaleValue = "auto";
		});
		// "auto" depends on the width of the container, so it is worked out again when that
		// changes. A zoom level the user picked is a number, and stays.
		const resizes = new ResizeObserver(() => {
			const scale = viewer.currentScaleValue;
			if (scale && isNaN(Number(scale))) {
				viewer.currentScaleValue = scale;
			}
			viewer.update();
		});
		resizes.observe(scroller);

		const task = getDocument({
			url: src,
			cMapUrl: runtimeAssets + "cmaps/",
			standardFontDataUrl: runtimeAssets + "standard_fonts/",
			wasmUrl: runtimeAssets + "wasm/",
			iccUrl: runtimeAssets + "iccs/",
		});
		let alive = true;
		this.teardown = () => {
			alive = false;
			resizes.disconnect();
			viewer.setDocument(null);
			task.destroy();
		};
		task.promise
			.then((doc) => {
				viewer.setDocument(doc);
				linkService.setDocument(doc);
			})
			.catch((err) => {
				if (!alive) {
					return; // destroying the task rejects its promise
				}
				console.error("Could not load PDF", src, err);
				this.teardown?.();
				this.style.height = "auto";
				this.textContent = bundled.getText("pdf.load_error");
			});
	}

	disconnectedCallback() {
		this.teardown?.();
		this.teardown = undefined;
		this.replaceChildren();
	}
}

function zoomButton(label: string, titleKey: string, onClick: () => void): HTMLButtonElement {
	const button = document.createElement("button");
	button.type = "button";
	button.className = "btn btn-blue";
	button.textContent = label;
	button.title = bundled.getText(titleKey);
	button.setAttribute("aria-label", button.title);
	button.addEventListener("click", onClick);
	return button;
}

customElements.define("kn-pdf", KNPDF);
