export default {
	async fetch(request, env, ctx): Promise<Response> {
		const paths = new URL(request.url).pathname.split("/");

		if (paths.length == 3 && paths[2] !== "") {
			// this is a file, send as download
			const { results } = await env.DB.prepare("SELECT * FROM files WHERE uuid = ?").bind(paths[1]).all();

			if (results[0]) {
				let file = await env.R2.get(results[0].uuid as string)
				return new Response(file?.body);
			}

			return new Response("404 not found", { status: 404 })

		} else if (paths.length == 2 || (paths.length == 3)) {
			const { results } = await env.DB.prepare("SELECT * FROM files WHERE uuid = ?").bind(paths[1]).all();

			if (results[0]) {
				return new Response(`
<!DOCTYPE html>
<html>
<head>
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous">
<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js" integrity="sha384-YvpcrYf0tY3lHB60NNkmXc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz" crossorigin="anonymous"></script>
</head>
<body class="p-2 bg-dark text-light" >
<a href="./${results[0].fileName}">${results[0].fileName}</a>
</body>
`, { status: 200, headers: { "Content-Type": "text/html" } })
			}

			return new Response("404 not found", { status: 404 });
		}

		// render a download page
		return Response.redirect("https://share.x4132.dev", 301);
	}
}

