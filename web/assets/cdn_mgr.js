
let extensionIcons = {};

for(let t of ['png', 'jpg', 'jpeg', 'gif']) {
	extensionIcons[t] = 'fa-file-image'
}

for(let t of ['mp4', 'mkv']) {
	extensionIcons[t] = 'fa-file-video'
}

extensionIcons['pdf'] = 'fa-file-pdf'
extensionIcons['csv'] = 'fa-file-csv'
extensionIcons['ppt'] = 'fa-file-powerpoint'
extensionIcons['pptx'] = 'fa-file-powerpoint'
extensionIcons['xls'] = 'fa-file-excel'
extensionIcons['xlsx'] = 'fa-file-excel'
extensionIcons['doc'] = 'fa-file-word'
extensionIcons['docx'] = 'fa-file-word'

for(let t of ['zip', 'rar', '7z', 'gz', 'tar']) {
	extensionIcons[t] = 'fa-file-archive'
}

for(let t of ['c', 'cpp', 'go', 'pas', 'py', 'py3', 'java', 'js']) {
	extensionIcons[t] = 'fa-file-code'
}

for(let t of ['md', 'html', 'txt']) {
	extensionIcons[t] = 'fa-file-alt'
}

export function getFileIcon(name) {
	let ext = name.split('.').pop();
	if(extensionIcons[ext] != null) {
		return extensionIcons[ext]
	}
	return "fa-file"
}

export { extensionIcons };

export class CDNManager {
	// startPath is an array with the respective path pieces to start with
	constructor(startPath, target_id, can_edit) {
		this.path = startPath
		this.target_id = target_id

		this.loading = true
		this.dirs = []
		this.files = []

		this.can_edit = can_edit

		this.class_id = Math.random().toString(36).substring(7);
	}

	async loadDir() {
		let res = await bundled.getCall("/cdn/readDir", {path: this.path.join('/')})
		if(res.status !== "success") {
			bundled.apiToast(res)
			return
		}
		this.dirs = res.data.dirs.filter(e => e.type == "directory")
		this.files = res.data.dirs.filter(e => e.type == "file")
		this.path = res.data.path
		this.canReadFirst = res.data.can_read_first
		this.loading = false

		this.render()
	}

	async createFolder(e) {
		e.preventDefault();
	}

	async uploadFile(e) {
		e.preventDefault();
		console.log(e.target.value)
	}

	async moveToDir(newPath) {
		this.path = newPath
		await this.poll()
	}

	async goToRelativeDir(dir) {
		if(dir == "..") {
			this.path.push(dir)
		} else {
			this.path.pop()
		}

		await this.poll()
	}

	async deleteObject(path) {
		let p  = [...this.path, path].join('/')
		let res = await bundled.postCall("/cdn/deleteObject", {path: p})
		if(res.status === "success") {
			await this.poll();
		}
		bundled.apiToast(res);
	}


	displayDir() {
	}

	displayFrame() {
		
	}

	displayLoading() {
		
	}

	displayEditing() {
	}

	render() {
		let target = document.getElementById(this.target_id)
		let node = null
		if(this.loading) {
			node = displayFrame()
		} else {
			node = displayLoading()
		}
		target.parentNode.replaceChild(node, target)
	}

};

