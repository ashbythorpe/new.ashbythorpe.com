export async function onRequest(context) {
  return await context.env.API_WORKER.fetch(context.request);
}
