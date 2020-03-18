import Vue from "vue";
import VueCodemirror from "vue-codemirror";

import "codemirror/mode/clike/clike";
import "codemirror/addon/dialog/dialog";
import "codemirror/addon/dialog/dialog.css";
import "codemirror/addon/search/searchcursor";
import "codemirror/addon/search/search";
import "codemirror/addon/edit/closebrackets";
import "codemirror/lib/codemirror.css";
import "codemirror/theme/monokai.css";
import "codemirror/keymap/sublime";
import "codemirror/addon/selection/active-line.js";

Vue.use(VueCodemirror);
