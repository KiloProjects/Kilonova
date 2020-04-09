import Vue from 'vue'
import CodeMirror from 'vue-codemirror'

// languages
import 'codemirror/mode/clike/clike.js'

// keymap
import 'codemirror/keymap/sublime.js'

Vue.use(CodeMirror, {
    theme: 'monokai',
    tabSize: 4,
    lineNumbers: true,
    line: true,
    keyMap: 'sublime'
})
