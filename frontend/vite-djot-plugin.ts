import { glob, readFile, stat, writeFile, mkdir } from "node:fs/promises";
import { extname, join, relative, resolve, dirname } from "node:path";
import { Plugin } from "vite";
import djot from "@djot/djot";
import {
    BundledLanguage,
    BundledTheme,
    createHighlighter,
    HighlighterGeneric,
} from "shiki";
import temml from "temml";

interface PostMeta {
    title: string;
    date: string;
    summary: string;
    url: string;
    filePath: string;
}

interface BlogIndexOptions {
    /** Directory containing .dj blog posts (relative to root) */
    postsDir: string;
    /** Template HTML file for the blog index pages */
    template: string;
    /** Output directory for index pages (relative to root) */
    outputDir: string;
    /** Number of posts per page */
    postsPerPage?: number;
}

interface PluginOptions {
    languages?: string[];
    template: string;
    blogIndex?: BlogIndexOptions;
}

export function djotPlugin(options: PluginOptions): Plugin {
    const { languages = [], template: templatePath, blogIndex } = options;

    let root: string = "";
    let resolvedTemplate: string;
    let template: string;
    let dev = true;

    const djotFiles: string[] = [];
    const trackedFiles = new Set<string>();
    const postMetadata = new Map<string, PostMeta>();
    let blogIndexDirty = true;

    return {
        name: "vite-djot",

        async configResolved(config) {
            await initHighlighter(languages);
            dev = config.command === "serve";
        },

        async buildStart() {
            template = await readFile(resolvedTemplate, "utf-8");
            postMetadata.clear();

            for (const file of djotFiles) {
                const { meta, resolvedPath } = await buildDjotFile(
                    file,
                    template,
                    root,
                );
                trackedFiles.add(resolvedPath);
                if (meta) {
                    postMetadata.set(resolvedPath, meta);
                }
            }

            if (blogIndex) {
                await generateBlogIndex(
                    blogIndex,
                    [...postMetadata.values()],
                    root,
                );
                blogIndexDirty = false;
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

            const djotInputs: Record<string, string> = {};

            for (const file of djotFiles) {
                const relativePath = relative(root, file);
                const inputName = relativePath.replace(/\.dj$/, "");
                const htmlPath = `${inputName}.html`;

                djotInputs[inputName] = htmlPath;
            }

            // Add blog index pages as inputs
            if (blogIndex) {
                const postsPerPage = blogIndex.postsPerPage ?? 10;
                const postsDir = resolve(root, blogIndex.postsDir);
                const blogPostCount = djotFiles.filter((f) =>
                    f.startsWith(postsDir),
                ).length;
                const totalPages = Math.max(
                    1,
                    Math.ceil(blogPostCount / postsPerPage),
                );

                const indexDir = blogIndex.outputDir;
                djotInputs["blog"] = `${indexDir}/index.html`;
                for (let page = 2; page <= totalPages; page++) {
                    djotInputs[`blog-page-${page}`] =
                        `${indexDir}/page/${page}/index.html`;
                }
            }

            if (!config.build) config.build = {};
            if (!config.build.rolldownOptions)
                config.build.rolldownOptions = {};
            if (!config.build.rolldownOptions.input)
                config.build.rolldownOptions.input = {};

            if (typeof config.build.rolldownOptions.input === "string") {
                config.build.rolldownOptions.input = {
                    main: config.build.rolldownOptions.input,
                    ...djotInputs,
                };
            } else {
                config.build.rolldownOptions.input = {
                    ...config.build.rolldownOptions.input,
                    ...djotInputs,
                };
            }
        },

        async handleHotUpdate({ file }) {
            if (file.endsWith(".dj") && trackedFiles.has(file)) {
                const { meta, resolvedPath } = await buildDjotFile(
                    file,
                    template,
                    root,
                );

                // Update just this file's entry in the metadata map
                if (meta) {
                    postMetadata.set(resolvedPath, meta);
                } else {
                    postMetadata.delete(resolvedPath);
                }

                if (blogIndex) {
                    await generateBlogIndex(
                        blogIndex,
                        [...postMetadata.values()],
                        root,
                    );
                    blogIndexDirty = false;
                }

                return [];
            }

            if (file === resolvedTemplate) {
                template = await readFile(resolvedTemplate, "utf-8");

                // Template changed — rebuild all posts (HTML changed)
                for (const f of trackedFiles) {
                    const { meta, resolvedPath } = await buildDjotFile(
                        f,
                        template,
                        root,
                    );
                    if (meta) {
                        postMetadata.set(resolvedPath, meta);
                    } else {
                        postMetadata.delete(resolvedPath);
                    }
                }

                if (blogIndex) {
                    await generateBlogIndex(
                        blogIndex,
                        [...postMetadata.values()],
                        root,
                    );
                    blogIndexDirty = false;
                }

                return [];
            }
        },

        configureServer(server) {
            server.watcher.add("**/*.dj");
            server.watcher.add(resolvedTemplate);

            if (blogIndex) {
                const resolvedBlogTemplate = resolve(root, blogIndex.template);
                server.watcher.add(resolvedBlogTemplate);
            }

            server.middlewares.use(async (req, res, next) => {
                // Handle blog index pages in dev
                if (blogIndex && req.url) {
                    const blogBase = `/${blogIndex.outputDir}`;
                    const isBlogIndex =
                        req.url === blogBase ||
                        req.url === `${blogBase}/` ||
                        req.url === `${blogBase}/index.html`;
                    const pageMatch = req.url.match(
                        new RegExp(
                            `^${blogBase}/page/(\\d+)/?(?:index\\.html)?$`,
                        ),
                    );

                    if ((isBlogIndex || pageMatch) && blogIndexDirty) {
                        await generateBlogIndex(
                            blogIndex,
                            [...postMetadata.values()],
                            root,
                        );
                        blogIndexDirty = false;
                    }
                }

                if (req.url?.endsWith(".html") && !req.url.includes("@")) {
                    const htmlPath = req.url.slice(1);
                    const djotPath = resolve(
                        root,
                        htmlPath.replace(/\.html$/, ".dj"),
                    );

                    try {
                        const stats = await stat(djotPath);
                        if (stats.isFile()) {
                            if (!trackedFiles.has(resolve(djotPath))) {
                                const { meta, resolvedPath } =
                                    await buildDjotFile(
                                        djotPath,
                                        template,
                                        root,
                                    );
                                trackedFiles.add(resolvedPath);
                                if (meta) {
                                    postMetadata.set(resolvedPath, meta);
                                    blogIndexDirty = true;
                                }
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
                            if (!trackedFiles.has(resolve(djotPath))) {
                                const { meta, resolvedPath } =
                                    await buildDjotFile(
                                        djotPath,
                                        template,
                                        root,
                                    );
                                trackedFiles.add(resolvedPath);
                                if (meta) {
                                    postMetadata.set(resolvedPath, meta);
                                    blogIndexDirty = true;
                                }
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

// --- Blog index generation ---

async function generateBlogIndex(
    options: BlogIndexOptions,
    posts: PostMeta[],
    root: string,
): Promise<void> {
    const { template: templatePath, outputDir, postsPerPage = 10 } = options;

    const resolvedTemplate = resolve(root, templatePath);
    const blogTemplate = await readFile(resolvedTemplate, "utf-8");

    // Sort posts by date, newest first
    const sorted = [...posts].sort(
        (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
    );

    const totalPages = Math.max(1, Math.ceil(sorted.length / postsPerPage));

    for (let page = 1; page <= totalPages; page++) {
        const start = (page - 1) * postsPerPage;
        const pagePosts = sorted.slice(start, start + postsPerPage);

        const postListHtml = pagePosts
            .map(
                (post) => `
            <blog-card href="${post.url}">
                <span slot="title">${post.title}</span>
                <time slot="date" datetime="${post.date}">${formatDate(post.date)}</time>
                <span slot="summary">${post.summary}</span>
            </blog-card>
            `,
            )
            .join("\n");

        const paginationHtml = buildPagination(page, totalPages, outputDir);

        const finalHtml = blogTemplate
            .replaceAll("{{POST_LIST}}", postListHtml)
            .replaceAll("{{PAGINATION}}", paginationHtml)
            .replaceAll("{{PAGE_NUMBER}}", String(page))
            .replaceAll("{{TOTAL_PAGES}}", String(totalPages));

        // Page 1 goes to outputDir/index.html, page 2+ to outputDir/page/N/index.html
        let outputPath: string;
        if (page === 1) {
            outputPath = resolve(root, outputDir, "index.html");
        } else {
            outputPath = resolve(
                root,
                outputDir,
                "page",
                String(page),
                "index.html",
            );
        }

        await mkdir(dirname(outputPath), { recursive: true });
        await writeFile(outputPath, finalHtml);
    }
}

function buildPagination(
    currentPage: number,
    totalPages: number,
    baseDir: string,
): string {
    if (totalPages <= 1) return "";

    const baseUrl = `/${baseDir}`;

    function pageUrl(page: number): string {
        if (page === 1) return `${baseUrl}/`;
        return `${baseUrl}/page/${page}/`;
    }

    const parts: string[] = [];
    parts.push(`<nav class="pagination" aria-label="Blog pages">`);

    if (currentPage > 1) {
        parts.push(
            `<a href="${pageUrl(currentPage - 1)}" class="pagination-prev" rel="prev">Previous</a>`,
        );
    } else {
        parts.push(
            `<span class="pagination-prev disabled" aria-disabled="true">Previous</span>`,
        );
    }

    parts.push(`<span class="pagination-pages">`);
    for (let i = 1; i <= totalPages; i++) {
        if (i === currentPage) {
            parts.push(
                `<span class="pagination-page current" aria-current="page">${i}</span>`,
            );
        } else {
            parts.push(
                `<a href="${pageUrl(i)}" class="pagination-page">${i}</a>`,
            );
        }
    }
    parts.push(`</span>`);

    if (currentPage < totalPages) {
        parts.push(
            `<a href="${pageUrl(currentPage + 1)}" class="pagination-next" rel="next">Next</a>`,
        );
    } else {
        parts.push(
            `<span class="pagination-next disabled" aria-disabled="true">Next</span>`,
        );
    }

    parts.push(`</nav>`);
    return parts.join("\n");
}

// --- Djot processing ---

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

interface DjotResult {
    title: string;
    html: string;
    meta: Record<string, string>;
}

async function buildDjotFile(
    path: string,
    template: string,
    root: string,
): Promise<{ meta: PostMeta | null; resolvedPath: string }> {
    const resolvedPath = resolve(path);
    const { title, html, meta } = await processDjot(resolvedPath);

    let header = "";
    header += `<time datetime="${meta.date}">${formatDate(meta.date)}</time>`;
    header += `<h1>${title}</h1>`;
    header += `<p class="post-summary">${meta.summary}</p>`;

    const finalHtml = template
        .replaceAll("{{TITLE}}", title)
        .replaceAll("{{HEADER}}", header)
        .replaceAll("{{CONTENT}}", html);

    const htmlPath = resolvedPath.replace(/\.dj$/, ".html");
    await writeFile(htmlPath, finalHtml);

    // Return metadata for blog index if this post has a date
    if (meta.date) {
        const relativePath = relative(root, htmlPath);
        const url =
            "/" +
            relativePath.replace(/index\.html$/, "").replace(/\.html$/, "/");

        return {
            meta: {
                title,
                date: meta.date,
                summary: meta.summary || "",
                url,
                filePath: resolvedPath,
            },
            resolvedPath,
        };
    }

    return { meta: null, resolvedPath };
}

async function processDjot(path: string): Promise<DjotResult> {
    const content = await readFile(path, "utf-8");

    const ast = djot.parse(content);

    let title = "";
    const meta: Record<string, string> = {};

    djot.applyFilter(ast, () => ({
        heading: (node: djot.Heading) => {
            if (node.level === 1 && title === "") {
                for (const child of node.children) {
                    if (child.tag === "str") {
                        title = child.text;
                    }
                }
                if (node.attributes) {
                    for (const [key, value] of Object.entries(
                        node.attributes,
                    )) {
                        meta[key] = value;
                    }
                }

                return { stop: [] };
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

            inline_math(node) {
                try {
                    return temml.renderToString(node.text, {
                        displayMode: false,
                    });
                } catch (e) {
                    console.warn(`Failed to render inline math: ${e}`);
                    return `<code>${node.text}</code>`;
                }
            },

            display_math(node) {
                try {
                    return temml.renderToString(node.text, {
                        displayMode: true,
                    });
                } catch (e) {
                    console.warn(`Failed to render display math: ${e}`);
                    return `<pre><code>${node.text}</code></pre>`;
                }
            },
        },
    });

    return { title, html, meta };
}

function formatDate(dateStr: string): string {
    const date = new Date(dateStr + "T00:00:00");
    return date.toLocaleDateString("en-GB", {
        day: "numeric",
        month: "long",
        year: "numeric",
    });
}
