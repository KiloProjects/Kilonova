
export class CheckboxManager {
	constructor(setAllCheckbox, checkboxes) {
		this.overall = setAllCheckbox
		this.checks = checkboxes

		this.overall.addEventListener('input', (e) => this.setAllChecked(e))
		for(let e of this.checks) {
			e.addEventListener('input', () => this.updateAllChecked())
		}

		this.updateAllChecked()
	}

	updateAllChecked() {
		var numChecked = 0;
		for(let e of this.checks) {
			numChecked += e.checked;
		}
		if(numChecked == this.checks.length) {
			this.overall.indeterminate = false;
			this.overall.checked = true;
		} else if(numChecked == 0) {
			this.overall.indeterminate = false;
			this.overall.checked = false;
		} else {
			this.overall.checked = false;
			this.overall.indeterminate = true;
		}
	}

	setAllChecked(e) {
		for(let ee of this.checks) {
			ee.checked = e.target.checked
		}
	}
};
