export class User {
	id: number;
	created_at: string;
	name: string;
	admin: boolean;
	proposer: boolean;
	email?: string;
	//password?: string;
	bio: string;
	default_visible: boolean;
	
	verified_email?: boolean;

	banned: boolean;
	disabled: boolean;

	// The constructor builds a new user from an object 
	constructor(data: any) {
		// TODO: do this safer
		this.id = data["id"]
		this.created_at = data["created_at"]
		this.name = data["name"]
		this.admin = data["admin"]
		this.proposer = data["proposer"]
		this.email = data["email"] ?? undefined
		this.bio = data["bio"]
		this.default_visible = data["default_visible"]
		this.verified_email = data["verified_email"] ?? undefined
		this.banned = data["banned"]
		this.disabled = data["disabled"]
	}
};

export class Problem {
	id: number;
	created_at: string;
	name: string;
	description: string;
	short_description: string;
	test_name: string;
	author_id: number;
	//TODO...

	constructor(data: any) {
		this.id = data["id"]
		this.created_at = data["created_at"]
		this.name = data["name"]
		this.description = data["description"]
		this.short_description = data["short_description"]
		this.test_name = data["test_name"]
		this.author_id = data["author_id"]
		// TODO...
	}
	
};

