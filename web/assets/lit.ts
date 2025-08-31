import { LitElement, html } from "lit";
import { customElement, property } from "lit/decorators.js";
import { parseTime } from "./util";

@customElement("server-timestamp")
export class ServerTimestamp extends LitElement {
	@property({ type: Number })
	timestamp?: number;

	@property({ type: Boolean })
	extended: boolean = false;

	render() {
		if (typeof this.timestamp === "undefined") {
			return html`<span>...</span>`;
		}
		return html`<span>${parseTime(this.timestamp, this.extended)}</span>`;
	}
}
