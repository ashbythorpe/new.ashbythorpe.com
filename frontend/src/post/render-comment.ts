import RenderWorker from "./markdown-worker.ts?worker";
import DOMPurify from "dompurify";

interface PendingRequest {
    text: string;
    resolve: (html: string) => void;
    reject: (error: Error) => void;
}

let worker: Worker;

const queue: PendingRequest[] = [];
let activeRequest: PendingRequest | null = null;
let terminateTimer: number | null = null;

function startWorker() {
    worker = new RenderWorker();

    worker.onmessage = (event: MessageEvent<string>) => {
        if (activeRequest !== null) {
            if (terminateTimer) {
                clearTimeout(terminateTimer);
            }

            activeRequest.resolve(event.data);
            activeRequest = null;

            processNext();
        }
    };
}

startWorker();

function processNext() {
    if (activeRequest !== null) {
        return;
    }

    activeRequest = queue.shift() ?? null;

    if (activeRequest === null) {
        return;
    }

    worker.postMessage(activeRequest.text);

    terminateTimer = setTimeout(() => {
        console.error(
            `Worker hung on comment.`,
        );

        if (activeRequest !== null) {
            activeRequest.reject(new Error("Timeout"));
            activeRequest = null;
        }

        worker.terminate();

        startWorker();

        processNext();
    }, 2500);
}


export async function renderComment(text: string): Promise<string> {
    const rawHTML = await new Promise<string>((resolve, reject) => {
        queue.push({ text, resolve, reject });
        processNext();
    });

    const cleanHTML = DOMPurify.sanitize(rawHTML, {
        USE_PROFILES: { mathMl: true },
        ADD_TAGS: ["b", "i", "em", "strong", "p", "br", "a", "code", "pre"],
        RETURN_DOM_FRAGMENT: false,
    });

    return cleanHTML;
}
