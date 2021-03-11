import { serve } from "https://deno.land/std@0.85.0/http/server.ts";
import { parse } from "https://deno.land/std@0.85.0/flags/mod.ts";
import { multiParser } from 'https://deno.land/x/multiparser@v2.0.3/mod.ts';
let args = parse(Deno.args);

args.hostname = args.hostname || "0.0.0.0";
args.port = args.port || 8040;
const server = serve({ hostname: args.hostname, port: args.port })
console.log(`Running markdown renderer server at http://${args.hostname}:${args.port}/`)

for await (const req of server) {
	try {
		let carr = ((await multiParser(req))?.files?.md?.content);
		if(!carr) {
			let headers = new Headers();
			headers.append("Content-Type", "text/html; charset=utf-8")
			req.respond({headers, body: "<div></div>"})
			continue
		}
		let content = new TextDecoder('utf-8').decode(carr)
		console.log(content);
		let headers = new Headers();
		headers.append("Content-Type", "text/html; charset=utf-8")
		req.respond({headers, body: "<h1> TEST </h1>"})
	} catch(e) {
		console.warn("Exception caught:", e);
	}
}
