import { defineConfig } from "vite";
import { customElementsPlugin } from "./vite-custom-elements-plugin";
import { djotPlugin } from "./vite-djot-plugin";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { fontPreloadPlugin } from "./vite-font-preload-plugin";

const __dirname = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
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
                "auth/sign-in": resolve(__dirname, "./auth/sign-in/index.html"),
                "auth/reset-password": resolve(__dirname, "./auth/reset-password/index.html"),
                "auth/verify-account": resolve(__dirname, "./auth/verify-account/index.html"),
            },
        },
    },
    server: {
        proxy: {
            "/api": {
                target: "http://localhost:3000",
                rewrite: (path) => path.replace(/^\/api/, ''),
            },

        },
    },
});
