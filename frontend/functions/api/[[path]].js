export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);

  url.hostname = "api.ashbythorpe.com";

  url.pathname = url.pathname.replace(/^\/api/, "");

  const modifiedRequest = new Request(url.toString(), {
    method: request.method,
    headers: request.headers,
    body: request.body,
    redirect: "manual",
  });

  try {
    const response = await fetch(modifiedRequest);

    return response;
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
