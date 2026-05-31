export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);

  url.hostname = "internal-go-api.local";

  url.pathname = url.pathname.replace(/^\/api/, "");

  const modifiedRequest = new Request(url.toString(), {
    method: request.method,
    headers: request.headers,
    body: request.body,
    redirect: "manual",
  });

  try {
    return await env.GO_BACKEND.fetch(modifiedRequest);
  } catch (err) {
    return new Response(
      JSON.stringify({ error: "Internal Gateway Error", details: err.message }),
      {
        status: 502,
        headers: { "Content-Type": "application/json" },
      },
    );
  }
}
