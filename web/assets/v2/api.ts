import defaultClient, {Client} from './requests';
import {User} from './models';

class API {
	client: Client = defaultClient

	async getSelf(): Promise<User> {
		let res = await this.client.getRequest("/user/getSelf")
		if(res.status === "success") {

		} else {
			throw new Error("");
		}
		return new User(res)
	}

	async getUser(id: number): Promise<User> {
		let res = await this.client.getRequest("/user/get", {id})
		return new User(res.data)
	}

}

export default new API();



