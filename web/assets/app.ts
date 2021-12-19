import 'preact/devtools';


// language definition

const languages = {
	"c": "text/x-csrc",
	"cpp": "text/x-c++src",
	"golang": "text/x-go",
	"haskell": "text/x-haskell",
	"java": "text/x-java",
	"python": "text/x-python",
}

export { languages };

import cookie from 'js-cookie';
export {cookie};

export * from './net';
export * from './util';
export * from './toast';
export * from './components';

export { getText } from './translation';

export { NavBarManager } from './navbar.js';
export { CheckboxManager } from './checkbox_mgr.js';
export { getFileIcon, extensionIcons } from './cdn_mgr';
