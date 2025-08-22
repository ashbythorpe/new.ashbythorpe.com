import type { IndexHtmlTransformContext, Plugin } from "vite";
import { access, constants, readFile } from "node:fs/promises";
import { resolve } from "path";
import ts, { SyntaxKind } from "typescript";
import { HTMLElement, parse } from "node-html-parser";
import MagicString from "magic-string";
import { dirname } from "node:path";
import * as esbuild from "esbuild";

interface ComponentBundle {
    name: string;
    templatePath: string;
    sourceFile: string;
    template: string;
    cssFiles: Set<string>;
    shadowRoot: boolean;
}

interface ElementDefinition {
    name: string;
    templatePath: string;
    sourceFile: string;
    classNode: ts.ClassDeclaration;
    static: boolean;
    shadowRoot: boolean;
}

interface PluginOptions {
    templateBaseDir?: string;
    enableSSR?: boolean;
    transformComponent?: (
        name: string,
        actualElement: HTMLElement,
        templateElement: HTMLElement,
        ctx: IndexHtmlTransformContext,
    ) =>
        | HTMLElement
        | null
        | undefined
        | void
        | Promise<HTMLElement | null | undefined | void>;
}

export function customElementsPlugin(options: PluginOptions = {}): Plugin {
    const {
        templateBaseDir = "src/components",
        enableSSR = true,
        transformComponent = null,
    } = options;

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

                    // Create virtual import IDs
                    const templateId = `virtual:template:${element.name}`;

                    const templateVarName =
                        toCamelCase(element.name) + "Template";

                    console.log(element.name);
                    console.log(element.static);
                    console.log(isDev);
                    if (!element.static || isDev) {
                        imports.push(
                            `import ${templateVarName} from "${templateId}";`,
                        );
                    }

                    if (!element.static || isDev) {
                        let additions = `\n    static __templateString = ${templateVarName};\n`;

                        if (isDev) {
                            additions += `\n    static __dev = true;\n`;
                        }

                        s.appendLeft(element.classNode.members.pos, additions);
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
                    return 'export default "";';
                }

                const res = `export default ${JSON.stringify(bundle.template)};`;
                return res;
            }

            return null;
        },

        transformIndexHtml: {
            order: "post",
            async handler(html, ctx) {
                if (!enableSSR) return html;

                const root = parse(html);

                const customElements = new Set(componentBundles.keys());

                // Process components in dependency order (deepest first)
                const processedElements = new Set<HTMLElement>();

                async function processComponentsAtLevel(
                    containerElement: HTMLElement,
                ): Promise<boolean> {
                    let hasChanges = false;

                    // Find all direct child custom elements (not nested in other custom elements)
                    const directCustomElements =
                        getDirectCustomElements(containerElement);

                    for (const element of directCustomElements) {
                        const tagName = element.tagName.toLowerCase();
                        const bundle = componentBundles.get(tagName);

                        if (bundle && !processedElements.has(element)) {
                            try {
                                const templateContent = bundle.template;
                                let templateElement = parse(templateContent);

                                if (transformComponent) {
                                    const result = await transformComponent(
                                        bundle.name,
                                        element,
                                        templateElement,
                                        ctx,
                                    );

                                    if (result) {
                                        templateElement = result;
                                    }
                                }

                                // First, recursively process any nested components in the template
                                await processComponentsAtLevel(templateElement);

                                if (bundle.shadowRoot) {
                                    // Then render this component
                                    const shadowTemplate = `<template shadowrootmode="open">${templateElement.innerHTML}</template>`;
                                    element.insertAdjacentHTML(
                                        "afterbegin",
                                        shadowTemplate,
                                    );
                                } else {
                                    element.insertAdjacentHTML(
                                        "afterbegin",
                                        templateElement.innerHTML,
                                    );
                                }

                                processedElements.add(element);
                                hasChanges = true;
                            } catch (error) {
                                console.warn(
                                    `Failed to process SSR for ${tagName}:`,
                                    error,
                                );
                            }
                        }
                    }

                    return hasChanges;
                }

                // Helper function to get only direct custom element children
                function getDirectCustomElements(
                    container: HTMLElement,
                ): HTMLElement[] {
                    const directElements: HTMLElement[] = [];

                    function traverse(
                        node: HTMLElement,
                        isDirectChild: boolean = true,
                    ) {
                        if (node.nodeType === 1) {
                            // Element node
                            const tagName = node.tagName?.toLowerCase();

                            if (tagName && customElements.has(tagName)) {
                                if (isDirectChild) {
                                    directElements.push(node);
                                }
                                // Don't traverse into custom elements - their children will be handled
                                // when the custom element itself is processed
                                return;
                            }
                        }

                        // Continue traversing for non-custom elements
                        if (node.childNodes) {
                            for (const child of node.childNodes) {
                                if (child instanceof HTMLElement) {
                                    traverse(child, isDirectChild);
                                }
                            }
                        }
                    }

                    traverse(container);
                    return directElements;
                }

                const hasChanges = await processComponentsAtLevel(root);
                return hasChanges ? root.outerHTML : html;
            },
        },

        async handleHotUpdate(ctx) {
            // Check if this file affects any component
            for (const [name, bundle] of componentBundles) {
                const shouldReload =
                    bundle.templatePath === ctx.file ||
                    bundle.cssFiles.has(ctx.file);

                if (shouldReload) {
                    await reloadBundle(bundle);

                    // Invalidate virtual modules
                    const templateModule = ctx.server.moduleGraph.getModuleById(
                        `virtual:template:${name}`,
                    );

                    if (templateModule)
                        await ctx.server.reloadModule(templateModule);

                    // Reload component module
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

        const templatePath = await tryLoadFile(element.templatePath, dir);

        const { template, cssFiles } = await loadTemplate(templatePath);

        return {
            name: element.name,
            templatePath,
            sourceFile,
            template,
            cssFiles,
            shadowRoot: element.shadowRoot,
        };
    }

    async function reloadBundle(bundle: ComponentBundle): Promise<void> {
        const { template, cssFiles } = await loadTemplate(bundle.templatePath);

        bundle.template = template;
        bundle.cssFiles = cssFiles;
    }

    async function loadTemplate(templatePath: string): Promise<{
        template: string;
        cssFiles: Set<string>;
    }> {
        let template: string;
        try {
            template = await readFile(templatePath, "utf-8");
        } catch (e) {
            throw new Error(
                `Error opening template file (${templatePath}): ${e}`,
            );
        }

        const templateElement = parse(template);

        const cssFiles: Set<string> = new Set();

        for (const linkElement of templateElement.querySelectorAll(
            "link[rel='stylesheet']",
        )) {
            const href = linkElement.getAttribute("href");

            if (href === undefined) continue;

            let cssFile: string;
            try {
                cssFile = await tryLoadFile(href, dirname(templatePath));
            } catch (e) {
                console.warn(`Error loading CSS file ${href}: ${e}`);
                continue;
            }

            const { outputFiles, metafile } = await esbuild.build({
                entryPoints: [cssFile],
                bundle: true,
                minify: true,
                write: false,
                metafile: true,
            });

            const minified = outputFiles[0].text;

            linkElement.replaceWith(`<style>${minified}</style>`);

            for (const file of Object.keys(metafile.inputs)) {
                cssFiles.add(resolve(root, file));
            }
        }

        for (const styleElement of templateElement.querySelectorAll("style")) {
            const { code: minified } = await esbuild.transform(
                styleElement.textContent,
                {
                    loader: "css",
                    minify: true,
                },
            );

            styleElement.textContent = minified;
        }

        return {
            template: templateElement.outerHTML,
            cssFiles,
        };
    }

    async function tryLoadFile(
        relativePath: string,
        dir: string,
    ): Promise<string> {
        let templatePath = resolve(dir, relativePath);

        try {
            await access(templatePath, constants.F_OK);
        } catch (error) {
            const basePath = resolve(root, templateBaseDir);
            templatePath = resolve(basePath, relativePath);

            try {
                await access(templatePath, constants.F_OK);
            } catch (error) {
                throw new Error(`Could not access ${relativePath}: ${error}`);
            }
        }

        return templatePath;
    }
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
            // Use TypeScript 5.0+ standardized decorators API
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
    let shadowRoot = true;

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
            } else if (decoratorName === "noShadowRoot") {
                shadowRoot = false;
            }
        }
    }

    if (definition !== null) {
        return {
            ...definition,
            static: isStatic,
            shadowRoot,
        };
    } else {
        return null;
    }
}

function toCamelCase(str: string): string {
    return str.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
}
