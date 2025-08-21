import { defineConfig } from "vite";
import { customElementsPlugin } from "./vite-custom-elements-plugin";
import { djotPlugin } from "./vite-djot-plugin";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
    plugins: [
        customElementsPlugin({
            transformComponent(name, actualElement, templateElement) {
                if (name === "nav-bar") {
                    const selected = actualElement.getAttribute("selected");

                    const item = templateElement.querySelector(`#navbar-${selected}`);

                    if (item) {
                        item.classList.add("selected");
                    } else {
                        console.error("Invalid `selected` attribute");
                    }
                }
            },
        }),
        djotPlugin({
            languages: ["typescript", "javascript", "prisma"],
            template: "post/template.html",
        }),
    ],
    esbuild: {
        target: "es2022",
    },
    appType: "mpa",
    build: {
        rollupOptions: {
            input: {
                main: resolve(__dirname, "index.html"),
            },
        },
    },
});
