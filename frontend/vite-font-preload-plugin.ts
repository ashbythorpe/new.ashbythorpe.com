import { HtmlTagDescriptor, Plugin } from "vite";

interface Font {
    name: string;
    import: string;
    type: string;
}

export function fontPreloadPlugin(fonts: Font[] = []): Plugin {
    return {
        name: "font-preload-production",
        apply: "build",

        transformIndexHtml: {
            order: "post",
            handler(html, context) {
                if (!context.bundle) return html;

                const preloadLinks: HtmlTagDescriptor[] = [];

                // Find font assets in the bundle
                const fontAssets = Object.entries(context.bundle)
                    .filter(
                        ([fileName, chunk]) =>
                            chunk.type === "asset" &&
                            (fileName.endsWith(".woff2") ||
                                fileName.endsWith(".woff") ||
                                fileName.endsWith(".ttf")),
                    )
                    .map(([fileName]) => fileName);

                // Match configured fonts to bundle assets
                fonts.forEach((fontConfig) => {
                    const matchingAsset = fontAssets.find((assetName) => {
                        const importName = fontConfig.import
                            .split("/")
                            .pop()!
                            .replace(".woff2", "")
                            .replace(".woff", "")
                            .replace(".ttf", "");

                        return assetName.includes(importName);
                    });

                    if (matchingAsset) {
                        preloadLinks.push({
                            tag: "link",
                            attrs: {
                                rel: "preload",
                                as: "font",
                                type: fontConfig.type,
                                href: `/${matchingAsset}`,
                            },
                            injectTo: "head",
                        });
                    }
                });

                return preloadLinks;
            },
        },
    };
}
