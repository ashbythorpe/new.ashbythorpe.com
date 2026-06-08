import type { Plugin } from "vite";
import { access, constants, readFile } from "node:fs/promises";
import { resolve } from "path";
import ts from "typescript";
import { HTMLElement, parse } from "node-html-parser";
import MagicString from "magic-string";
import { dirname } from "node:path";
import { bundle as bundleCss, transform as transformCss } from "lightningcss";

interface ComponentBundle {
    name: string;
    templatePath: string;
    sourceFile: string;
    /** Full template HTML with inline <style> tags — used only by transformIndexHtml for SSR. */
    template: string;
    /** Template HTML with <style> tags stripped — exported to the client for dynamic construction. */
    templateString: string;
    cssText: string;
    cssFiles: Set<string>;
    static: boolean;
}

interface ElementDefinition {
    name: string;
    templatePath: string;
    sourceFile: string;
    classNode: ts.ClassDeclaration;
    static: boolean;
}

interface PluginOptions {
    templateBaseDir?: string;
    enableSSR?: boolean;
}

/**
 * Process `<include src="./path/to/file.svg" />` elements by inlining
 * the referenced file's contents. Any extra attributes on the <include>
 * element are forwarded onto the root element of the inlined content
 * (e.g. class, id, aria-label).
 *
 * Returns the set of resolved file paths that were inlined, so callers
 * can register them as watch dependencies.
 */
async function processIncludes(
    root: HTMLElement,
    baseDir: string,
    projectRoot: string,
): Promise<Set<string>> {
    const includedFiles = new Set<string>();

    const includeElements = root.querySelectorAll("include");
    for (const el of includeElements) {
        const src = el.getAttribute("src");
        if (!src) {
            console.warn("<include> element missing src attribute");
            continue;
        }

        let filePath: string;
        try {
            filePath = await resolveFile(src, baseDir, projectRoot);
        } catch (e) {
            console.warn(`Could not resolve <include src="${src}">: ${e}`);
            continue;
        }

        includedFiles.add(filePath);

        const content = await readFile(filePath, "utf-8");
        const parsed = parse(content.trim());

        // Forward attributes from <include> onto the root element of
        // the inlined content (typically <svg>).
        const inlinedRoot = parsed.firstChild;
        if (inlinedRoot instanceof HTMLElement) {
            for (const [attr, value] of Object.entries(el.attributes)) {
                if (attr === "src") continue;
                // Merge class attributes rather than overwriting
                if (attr === "class") {
                    const existing = inlinedRoot.getAttribute("class") || "";
                    inlinedRoot.setAttribute(
                        "class",
                        existing ? `${existing} ${value}` : value,
                    );
                } else {
                    inlinedRoot.setAttribute(attr, value);
                }
            }
        }

        // Recursively process nested includes in the inlined content
        const nestedFiles = await processIncludes(
            parsed,
            dirname(filePath),
            projectRoot,
        );
        for (const f of nestedFiles) {
            includedFiles.add(f);
        }

        el.replaceWith(parsed.toString());
    }

    return includedFiles;
}

/**
 * Resolve a relative path against a base directory, falling back to
 * the project root.
 */
async function resolveFile(
    relativePath: string,
    baseDir: string,
    projectRoot: string,
): Promise<string> {
    const primary = resolve(baseDir, relativePath);
    try {
        await access(primary, constants.F_OK);
        return primary;
    } catch {
        const fallback = resolve(projectRoot, relativePath);
        try {
            await access(fallback, constants.F_OK);
            return fallback;
        } catch {
            throw new Error(
                `Could not find "${relativePath}" in ${baseDir} or ${projectRoot}`,
            );
        }
    }
}

export function customElementsPlugin(options: PluginOptions = {}): Plugin {
    const { templateBaseDir = "src/components", enableSSR = true } = options;

    const componentBundles = new Map<string, ComponentBundle>();
    let root = "";
    let isDev = false;

    return {
        name: "vite-custom-elements",

        configResolved(config) {
            root = config.root;
            isDev = config.command === "serve";
        },

        transform: {
            order: "pre",
            async handler(code, id) {
                if (!/\.(ts|js)$/.test(id)) return null;

                const elements = parseElementDecorators(code, id);
                if (elements.length === 0) return null;

                const s = new MagicString(code);

                const imports: string[] = [];

                for (const element of elements) {
                    const bundle = await createComponentBundle(element, id);
                    componentBundles.set(element.name, bundle);

                    const templateId = `virtual:template:${element.name}`;
                    const templateVarName = toCamelCase(element.name) + "Template";
                    const cssVarName = toCamelCase(element.name) + "Css";

                    if (!element.static || isDev) {
                        if (element.static) {
                            // Static elements only need the template string (for dev HMR);
                            // no CSS import since adoptedStyleSheets is never used.
                            imports.push(
                                `import { template as ${templateVarName} } from "${templateId}";`,
                            );
                        } else {
                            // Non-static elements import both — template for the fallback
                            // dynamic-attach path, css for adoptedStyleSheets.
                            imports.push(
                                `import { template as ${templateVarName}, css as ${cssVarName} } from "${templateId}";`,
                            );
                        }
                    }

                    if (!element.static || isDev) {
                        const injected = element.static
                            ? `\n    static __templateString = ${templateVarName};\n`
                            : `\n    static __templateString = ${templateVarName};\n    static __cssText = ${cssVarName};\n`;

                        s.appendLeft(element.classNode.members.pos, injected);
                    }

                    this.addWatchFile(bundle.templatePath);

                    for (const file of bundle.cssFiles) {
                        this.addWatchFile(file);
                    }
                }

                if (imports.length > 0) {
                    s.prepend(imports.join("\n") + "\n");
                }

                const res = s.toString();

                return {
                    code: res,
                    map: s.generateMap({ hires: true, source: id }),
                };
            },
        },

        resolveId(id) {
            if (id.startsWith("virtual:template:")) {
                return id;
            }
            return null;
        },

        load(id) {
            const templateMatch = id.match(/^virtual:template:(.+)$/);
            if (templateMatch) {
                const elementName = templateMatch[1];
                const bundle = componentBundles.get(elementName);

                if (!bundle) {
                    this.warn(
                        `Component bundle for "${elementName}" not found`,
                    );
                    return {
                        code: 'export const template = ""; export const css = "";',
                        moduleType: "js",
                    };
                }

                // Export templateString (styles stripped) as `template` — this is what
                // the client uses for dynamic construction, where adoptedStyleSheets
                // supplies the styles. The full template (with <style> tags) is only
                // ever used server-side in transformIndexHtml and is never shipped to
                // the client. Static elements never import `css`, so it is tree-shaken.
                const res = [
                    `export const template = ${JSON.stringify(bundle.templateString)};`,
                    `export const css = ${JSON.stringify(bundle.cssText)};`,
                ].join("\n");

                return { code: res, moduleType: "js" };
            }

            return null;
        },

        transformIndexHtml: {
            order: "post",
            async handler(html, ctx) {
                const htmlRoot = parse(html);
                let hasChanges = false;

                // Process <include> elements in the HTML page itself
                const includedFiles = await processIncludes(
                    htmlRoot,
                    dirname(ctx.filename),
                    root,
                );
                if (includedFiles.size > 0) {
                    hasChanges = true;
                }

                // SSR: inline component templates
                if (enableSSR) {
                    const customElements = new Set(componentBundles.keys());
                    const processedElements = new Set<HTMLElement>();

                    async function processComponentsAtLevel(
                        containerElement: HTMLElement,
                    ): Promise<boolean> {
                        let levelHasChanges = false;

                        const directCustomElements = getDirectCustomElements(
                            containerElement,
                            customElements,
                        );

                        for (const element of directCustomElements) {
                            const tagName = element.tagName.toLowerCase();
                            const bundle = componentBundles.get(tagName);

                            if (bundle && !processedElements.has(element)) {
                                try {
                                    const templateContent = bundle.template;
                                    let templateElement =
                                        parse(templateContent);

                                    // Recursively process nested components
                                    await processComponentsAtLevel(
                                        templateElement,
                                    );

                                    // template already contains inlined <style> tags,
                                    // so shadowrootmode="open" gets styles for free.
                                    const shadowTemplate = `<template shadowrootmode="open">${templateElement.innerHTML}</template>`;
                                    element.insertAdjacentHTML(
                                        "afterbegin",
                                        shadowTemplate,
                                    );

                                    processedElements.add(element);
                                    levelHasChanges = true;
                                } catch (error) {
                                    console.warn(
                                        `Failed to process SSR for ${tagName}:`,
                                        error,
                                    );
                                }
                            }
                        }

                        return levelHasChanges;
                    }

                    const ssrChanged = await processComponentsAtLevel(htmlRoot);
                    if (ssrChanged) {
                        hasChanges = true;
                    }
                }

                return hasChanges ? htmlRoot.outerHTML : html;
            },
        },

        async handleHotUpdate(ctx) {
            for (const [name, bundle] of componentBundles) {
                const shouldReload =
                    bundle.templatePath === ctx.file ||
                    bundle.cssFiles.has(ctx.file);

                if (shouldReload) {
                    await reloadBundle(bundle);

                    const templateModule = ctx.server.moduleGraph.getModuleById(
                        `virtual:template:${name}`,
                    );

                    if (templateModule)
                        await ctx.server.reloadModule(templateModule);

                    const componentModule =
                        ctx.server.moduleGraph.getModuleById(bundle.sourceFile);
                    if (componentModule) {
                        await ctx.server.reloadModule(componentModule);
                    }
                }
            }
        },
    };

    async function createComponentBundle(
        element: ElementDefinition,
        sourceFile: string,
    ): Promise<ComponentBundle> {
        const dir = dirname(sourceFile);

        const templatePath = await resolveFile(
            element.templatePath,
            dir,
            resolve(root, templateBaseDir),
        );

        const { template, templateString, cssText, cssFiles } = await loadTemplate(templatePath);

        return {
            name: element.name,
            templatePath,
            sourceFile,
            template,
            templateString,
            cssText,
            cssFiles,
            static: element.static,
        };
    }

    async function reloadBundle(bundle: ComponentBundle): Promise<void> {
        const { template, templateString, cssText, cssFiles } = await loadTemplate(bundle.templatePath);

        bundle.template = template;
        bundle.templateString = templateString;
        bundle.cssText = cssText;
        bundle.cssFiles = cssFiles;
    }

    async function loadTemplate(templatePath: string): Promise<{
        template: string;
        templateString: string;
        cssText: string;
        cssFiles: Set<string>;
    }> {
        let rawTemplate: string;
        try {
            rawTemplate = await readFile(templatePath, "utf-8");
        } catch (e) {
            throw new Error(
                `Error opening template file (${templatePath}): ${e}`,
            );
        }

        const templateElement = parse(rawTemplate);

        const cssFiles: Set<string> = new Set();
        const cssChunks: string[] = [];

        // Process <include> elements in the template
        await processIncludes(
            templateElement,
            dirname(templatePath),
            root,
        );

        for (const linkElement of templateElement.querySelectorAll(
            "link[rel='stylesheet']",
        )) {
            const href = linkElement.getAttribute("href");

            if (href === undefined) continue;

            let cssFile: string;
            try {
                cssFile = await resolveFile(href, dirname(templatePath), root);
            } catch (e) {
                console.warn(`Error loading CSS file ${href}: ${e}`);
                continue;
            }

            const result = bundleCss({
                filename: cssFile,
                minify: true,
                sourceMap: false,
            });

            const minified = result.code.toString();
            cssChunks.push(minified);

            // Keep <style> inline in the template for SSR / shadowrootmode consumers.
            linkElement.replaceWith(`<style>${minified}</style>`);

            // Track the entry file itself
            cssFiles.add(cssFile);

            // Track any @import dependencies
            if (result.dependencies) {
                for (const dep of result.dependencies) {
                    if (dep.type === "import") {
                        cssFiles.add(resolve(dirname(cssFile), dep.url));
                    }
                }
            }
        }

        for (const styleElement of templateElement.querySelectorAll("style")) {
            const result = transformCss({
                filename: "inline.css",
                code: Buffer.from(styleElement.textContent),
                minify: true,
                sourceMap: false,
            });

            const minified = result.code.toString();
            styleElement.textContent = minified;
            cssChunks.push(minified);
        }

        // template: full HTML with <style> tags — for transformIndexHtml SSR only,
        //           never shipped to the client as a JS string.
        const template = templateElement.outerHTML;

        // templateString: <style> tags stripped — exported via the virtual module
        //                 for dynamic construction, where adoptedStyleSheets
        //                 supplies the styles instead.
        for (const styleElement of templateElement.querySelectorAll("style")) {
            styleElement.remove();
        }
        const templateString = templateElement.outerHTML;

        return {
            template,
            templateString,
            // Concatenation is safe: minified CSS blocks are self-contained.
            cssText: cssChunks.join(""),
            cssFiles,
        };
    }
}

function getDirectCustomElements(
    container: HTMLElement,
    customElements: Set<string>,
): HTMLElement[] {
    const directElements: HTMLElement[] = [];

    function traverse(node: HTMLElement) {
        if (node.nodeType === 1) {
            const tagName = node.tagName?.toLowerCase();

            if (tagName && customElements.has(tagName)) {
                directElements.push(node);
                return;
            }
        }

        if (node.childNodes) {
            for (const child of node.childNodes) {
                if (child instanceof HTMLElement) {
                    traverse(child);
                }
            }
        }
    }

    traverse(container);
    return directElements;
}

function parseElementDecorators(
    code: string,
    filePath: string,
): ElementDefinition[] {
    const sourceFile = ts.createSourceFile(
        filePath,
        code,
        ts.ScriptTarget.Latest,
        true,
    );

    const elements: ElementDefinition[] = [];

    function visit(node: ts.Node) {
        if (ts.isClassDeclaration(node)) {
            const decorators = ts.getDecorators?.(node);

            if (decorators) {
                const definition = parseElementDefinition(
                    node,
                    decorators,
                    filePath,
                );

                if (definition !== null) {
                    elements.push(definition);
                }
            }
        }

        ts.forEachChild(node, visit);
    }

    visit(sourceFile);
    return elements;
}

function parseElementDefinition(
    node: ts.ClassDeclaration,
    decorators: readonly ts.Decorator[],
    filePath: string,
): ElementDefinition | null {
    let definition: Omit<ElementDefinition, "static" | "shadowRoot"> | null =
        null;
    let isStatic = false;

    for (const decorator of decorators) {
        if (
            ts.isCallExpression(decorator.expression) &&
            ts.isIdentifier(decorator.expression.expression)
        ) {
            const decoratorName = decorator.expression.expression.text;
            if (decoratorName === "element") {
                const args = decorator.expression.arguments;
                if (
                    args.length >= 2 &&
                    ts.isStringLiteral(args[0]) &&
                    ts.isStringLiteral(args[1])
                ) {
                    definition = {
                        name: args[0].text,
                        templatePath: args[1].text,
                        sourceFile: filePath,
                        classNode: node,
                    };
                }
            }
        } else if (ts.isIdentifier(decorator.expression)) {
            const decoratorName = decorator.expression.text;
            if (decoratorName === "staticElement") {
                isStatic = true;
            }
        }
    }

    if (definition !== null) {
        return {
            ...definition,
            static: isStatic,
        };
    } else {
        return null;
    }
}

function toCamelCase(str: string): string {
    return str.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
}
