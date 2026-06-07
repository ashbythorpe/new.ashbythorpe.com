import "./marked-temml-extension.ts";
import { marked } from "marked";
import DOMPurify from "dompurify";

DOMPurify.addHook("afterSanitizeAttributes", function (node) {
    if (node.nodeName === "A") {
        node.setAttribute("rel", "nofollow ugc noopener noreferrer");

        node.setAttribute("target", "_blank");
    }
});

export function renderComment(rawInput: string): string {
    const rawHTML = marked.parse(rawInput) as unknown as string;

    console.log(rawHTML);

    const cleanHTML = DOMPurify.sanitize(rawHTML, {
        USE_PROFILES: { mathMl: true },
        ADD_TAGS: ["b", "i", "em", "strong", "p", "br", "a", "code", "pre"],
        RETURN_DOM_FRAGMENT: false,
    });

    return cleanHTML;
}
