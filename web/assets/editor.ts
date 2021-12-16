import {keymap, EditorView} from "@codemirror/view"
import {EditorState} from "@codemirror/state"
import {history, historyKeymap} from "@codemirror/history"
import {defaultKeymap} from "@codemirror/commands"

const baseExtensions = [
	history(),
	keymap.of([...defaultKeymap, ...historyKeymap])
]

export function fromTextArea(textarea: HTMLTextAreaElement, extensions: any) {
	let view = new EditorView({
		state: EditorState.create({doc: textarea.value, extensions: [...baseExtensions, ...extensions]})
	})
	textarea.parentNode!.insertBefore(view.dom, textarea)
	textarea.style.display = "none"
	if(textarea.form) textarea.form.addEventListener("submit", () => {
		textarea.value = view.state.doc.toString()
	})
	return view
}
