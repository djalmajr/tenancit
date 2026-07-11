const events = [];
Bun.serve({
  port: 9090,
  async fetch(request) {
    const url = new URL(request.url);
    if (url.pathname === "/healthz") return new Response("ok");
    if (url.pathname === "/events" && request.method === "GET") return Response.json(events);
    if (url.pathname === "/events" && request.method === "DELETE") { events.splice(0); return new Response(null, { status: 204 }); }
    if (url.pathname === "/hook" && request.method === "POST") {
      events.push({ body: await request.text(), headers: Object.fromEntries(request.headers) });
      return new Response(null, { status: 204 });
    }
    return new Response("not found", { status: 404 });
  },
});
