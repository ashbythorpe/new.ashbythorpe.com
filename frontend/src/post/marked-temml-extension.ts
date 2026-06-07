import { marked, type TokenizerAndRendererExtension } from "marked";
import temml from "temml";
import "../styles/Temml-Local.css";

// Adapted from https://github.com/UziTech/marked-katex-extension
const inlineRule =
    /^(\${1,2})(?!\$)((?:\\.|[^\\\n])*?(?:\\.|[^\\\n\$]))\1(?=[\s?!\.,:？！。，：]|$)/;
const blockRule = /^(\${1,2})\n((?:\\[^]|[^\\])+?)\n\1(?:\n|$)/;

function renderMath(text: string, displayMode?: boolean) {
    try {
        return temml.renderToString(text, { displayMode, throwOnError: true });
    } catch (err) {
        console.error(err);
        if (err instanceof Error) {
            return `<span style="color: red;" title="${err.message}">${text}</span>`;
        }
    }
}

const temmlInline: TokenizerAndRendererExtension = {
    name: "temmlInline",
    level: "inline",
    start(src) {
        let index;
        let indexSrc = src;
        while (indexSrc) {
            index = indexSrc.indexOf("$");
            if (index === -1) return;

            // Ensure it's the start of a string or preceded by a space
            if (index === 0 || indexSrc.charAt(index - 1) === " ") {
                const possibleMath = indexSrc.substring(index);
                if (possibleMath.match(inlineRule)) {
                    return index;
                }
            }
            indexSrc = indexSrc.substring(index + 1).replace(/^\$+/, "");
        }
    },
    tokenizer(src) {
        const match = src.match(inlineRule);
        if (match) {
            return {
                type: "temmlInline",
                raw: match[0],
                text: match[2].trim(),
                displayMode: match[1].length === 2,
            };
        }
    },
    renderer(token) {
        return renderMath(token.text, token.displayMode);
    },
};

const temmlBlock: TokenizerAndRendererExtension = {
    name: "temmlBlock",
    level: "block",
    tokenizer(src) {
        const match = src.match(blockRule);
        if (match) {
            return {
                type: "temmlBlock",
                raw: match[0],
                text: match[2].trim(),
                displayMode: match[1].length === 2,
            };
        }
    },
    renderer(token) {
        return renderMath(token.text, token.displayMode) + "\n";
    },
};

marked.use({ extensions: [temmlInline, temmlBlock] });
