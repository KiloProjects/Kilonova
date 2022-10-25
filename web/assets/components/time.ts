import dayjs from 'dayjs';

type TimeHandler = () => (Promise<void> | void);

class UnifiedTimePasser {
	timers: Map<number, TimeHandler> = new Map<number, TimeHandler>();
	id_cnt: number = 0;
	int_id: number;

	constructor() {
		this.int_id = setInterval(() => this.runHandlers(), 1000)
	}

	async runHandlers(): Promise<void> {
		await Promise.all(Array.from(this.timers.values()).map(func => func()))
	}

	addHandler(h: TimeHandler): number {
		this.id_cnt++;
		this.timers.set(this.id_cnt, h)
		return this.id_cnt
	}

	removeHandler(id: number): void {
		if(this.timers.has(id)) {
			this.timers.delete(id)
		}
	}
}

const control = new UnifiedTimePasser();


function Countdown() {
}

function Timer() {
	let now = dayjs();

	function onHandle() {
	}

}

