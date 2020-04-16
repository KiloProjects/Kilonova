import Vue from 'vue'
import CodeMirror from 'vue-codemirror'

// languages
import 'codemirror/mode/clike/clike.js'

// handy stuff
import 'codemirror/addon/edit/matchbrackets.js'
import 'codemirror/addon/edit/closebrackets.js'

// selection
import 'codemirror/addon/selection/active-line.js'

// keymap
import 'codemirror/keymap/sublime.js'

Vue.use(CodeMirror)
