import { marked } from "marked";
import { temmlBlock, temmlInline } from "./marked-temml-extensions.ts";

marked.use({ extensions: [temmlInline, temmlBlock] });

self.onmessage = (event: MessageEvent<string>) => {
    const result = marked.parse(event.data);

    self.postMessage(result);
};
