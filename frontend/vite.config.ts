import { defineConfig } from "vite";
import { customElementsPlugin } from "./vite-custom-elements-plugin";
import { djotPlugin } from "./vite-djot-plugin";
import path, { dirname, resolve } from "node:path";
import fs from "node:fs";
import { fileURLToPath } from "node:url";
import { fontPreloadPlugin } from "./vite-font-preload-plugin";
import purgecss from "@fullhuman/postcss-purgecss";

const __dirname = dirname(fileURLToPath(import.meta.url));

function getUsedComponentVariables(): string[] {
    const usedVars = new Set<string>();
    const componentsDir = path.resolve(__dirname, "src");

    if (!fs.existsSync(componentsDir)) return [];

    const files = fs.readdirSync(componentsDir, { recursive: true });

    const regex = /var\((--[\w-]+)\)/g;

    for (const file of files) {
        const filePath = path.join(componentsDir, file.toString());

        if (fs.statSync(filePath).isFile()) {
            const content = fs.readFileSync(filePath, "utf-8");

            let match;
            while ((match = regex.exec(content)) !== null) {
                usedVars.add(match[1]);
            }
        }
    }

    return Array.from(usedVars);
}

export default defineConfig(({ command }) => {
    const postCSSPlugins = [];

    if (command === "build") {
        postCSSPlugins.push(
            purgecss({
                content: ["./**/*.html"],
                variables: true,
                safelist: {
                    variables: getUsedComponentVariables(),
                },
            }),
        );
    }

    return {
        plugins: [
            customElementsPlugin(),
            djotPlugin({
                languages: ["typescript", "javascript", "prisma"],
                template: "post/template.html",
                blogIndex: {
                    postsDir: "blog",
                    template: "blog/template.html",
                    outputDir: "blog",
                    postsPerPage: 10,
                },
            }),
            fontPreloadPlugin([
                {
                    name: "inter",
                    import: "@fontsource/inter/files/inter-latin-400-normal.woff2",
                    type: "font/woff2",
                },
            ]),
        ],
        appType: "mpa",
        build: {
            rolldownOptions: {
                input: {
                    main: resolve(__dirname, "index.html"),
                    contact: resolve(__dirname, "contact/index.html"),
                    projects: resolve(__dirname, "projects/index.html"),
                    "auth/sign-in": resolve(
                        __dirname,
                        "./auth/sign-in/index.html",
                    ),
                    "auth/reset-password": resolve(
                        __dirname,
                        "./auth/reset-password/index.html",
                    ),
                    "auth/verify-account": resolve(
                        __dirname,
                        "./auth/verify-account/index.html",
                    ),
                },
            },
        },
        server: {
            proxy: {
                "/api": {
                    target: "http://localhost:3000",
                    rewrite: (path) => path.replace(/^\/api/, ""),
                },
            },
        },
        css: {
            postcss: {
                plugins: postCSSPlugins,
            },
        },
        resolve: {
            alias: {
                temml: path.resolve(__dirname, "./src/lib/temml.min.js"),
            },
        },
    };
});
