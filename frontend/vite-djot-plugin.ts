import { glob, readFile, stat, writeFile } from "node:fs/promises";
import { extname, join, relative, resolve } from "node:path";
import { Plugin } from "vite";
import djot from "@djot/djot";
import {
    BundledLanguage,
    BundledTheme,
    createHighlighter,
    HighlighterGeneric,
} from "shiki";

interface PluginOptions {
    languages?: string[];
    template: string;
}

export function djotPlugin(options: PluginOptions): Plugin {
    const { languages = [], template: templatePath } = options;

    let root: string = "";
    let resolvedTemplate: string;
    let template: string;
    let dev = true;

    const djotFiles: string[] = [];
    const trackedFiles = new Set<string>();

    return {
        name: "vite-djot",

        async configResolved(config) {
            await initHighlighter(languages);
            dev = config.command === "serve";
        },

        async buildStart() {
            template = await readFile(resolvedTemplate, "utf-8");

            if (!dev) {
                for (const file of djotFiles) {
                    await buildDjotFile(file, template);
                }
            }
        },

        async config(config) {
            if (config.root) {
                root = config.root;
            }
            resolvedTemplate = resolve(root, templatePath);

            const foundFiles = glob("**/*.dj", {
                cwd: root,
                exclude: ["node_modules/**"],
            });

            for await (const file of foundFiles) {
                djotFiles.push(join(root, file));
            }

            const djotInputs = {};

            for (const file of djotFiles) {
                const relativePath = relative(root, file);
                const inputName = relativePath.replace(/\.dj$/, "");
                const htmlPath = `${inputName}.html`;

                djotInputs[inputName] = htmlPath;
            }

            if (!config.build) config.build = {};
            if (!config.build.rollupOptions) config.build.rollupOptions = {};
            if (!config.build.rollupOptions.input)
                config.build.rollupOptions.input = {};

            if (typeof config.build.rollupOptions.input === "string") {
                config.build.rollupOptions.input = {
                    main: config.build.rollupOptions.input,
                    ...djotInputs,
                };
            } else {
                config.build.rollupOptions.input = {
                    ...config.build.rollupOptions.input,
                    ...djotInputs,
                };
            }
        },

        async handleHotUpdate({ file }) {
            if (file.endsWith(".dj") && trackedFiles.has(file)) {
                await buildDjotFile(file, template);

                return [];
            }

            if (file === resolvedTemplate) {
                template = await readFile(resolvedTemplate, "utf-8");
                for (const file of trackedFiles) {
                    await buildDjotFile(file, template);
                }

                return [];
            }
        },

        configureServer(server) {
            server.watcher.add("**/*.dj");
            server.watcher.add(resolvedTemplate);

            server.middlewares.use(async (req, res, next) => {
                if (req.url?.endsWith(".html") && !req.url.includes("@")) {
                    const htmlPath = req.url.slice(1);
                    const djotPath = resolve(
                        root,
                        htmlPath.replace(/\.html$/, ".dj"),
                    );

                    try {
                        const stats = await stat(djotPath);
                        if (stats.isFile()) {
                            if (!trackedFiles.has(djotPath)) {
                                await buildDjotFile(djotPath, template);
                                trackedFiles.add(djotPath);
                            }
                        }
                    } catch (e) {
                        // File doesn't exist
                    }
                } else if (
                    req.url &&
                    !req.url?.includes("@") &&
                    extname(req.url) === ""
                ) {
                    const path = req.url.slice(1);
                    const djotPath = resolve(root, path, "index.dj");

                    try {
                        const stats = await stat(djotPath);
                        if (stats.isFile()) {
                            if (!trackedFiles.has(djotPath)) {
                                await buildDjotFile(djotPath, template);
                                trackedFiles.add(djotPath);
                            }
                        }
                    } catch (e) {
                        // File doesn't exist
                    }
                }

                next();
            });
        },
    };
}

let highlighter: HighlighterGeneric<BundledLanguage, BundledTheme> | null =
    null;

async function initHighlighter(languages: string[]) {
    if (highlighter === null) {
        highlighter = await createHighlighter({
            themes: ["everforest-light"],
            langs: languages,
        });
    }
}

async function buildDjotFile(path: string, template: string) {
    const { title, html } = await processDjot(path);

    const finalHtml = template
        .replaceAll("{{TITLE}}", title)
        .replaceAll("{{CONTENT}}", html);

    const htmlPath = path.replace(/\.dj$/, ".html");

    await writeFile(htmlPath, finalHtml);
}

interface DjotResult {
    title: string;
    html: string;
}

async function processDjot(path: string): Promise<DjotResult> {
    const content = await readFile(path, "utf-8");

    const ast = djot.parse(content);

    let title = "";

    djot.applyFilter(ast, () => ({
        heading: (node: djot.Heading) => {
            if (node.level === 1 && title === "") {
                for (const child of node.children) {
                    if (child.tag === "str") {
                        title = child.text;
                    }
                }
            }
        },
    }));

    const html = djot.renderHTML(ast, {
        overrides: {
            code_block(node, context) {
                const language = node.lang;

                if (!language) {
                    return context.renderAstNode(node);
                }

                const code = node.text;

                if (!highlighter) {
                    console.error("Internal error: highlighter not created");
                    return context.renderAstNode(node);
                }

                try {
                    const highlighted = highlighter.codeToHtml(code, {
                        lang: language,
                        theme: "everforest-light",
                    });
                    return highlighted;
                } catch (e) {
                    console.warn(`Failed to highlight ${language}: ${e}`);
                    return context.renderAstNode(node);
                }
            },
        },
    });

    return { title, html };
}
