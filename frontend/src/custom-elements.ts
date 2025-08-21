export function element(name: string, _templatePath: string): ClassDecorator {
    function decorator<T extends Component>(ComponentClass: T): T {
        /* @ts-ignore */
        class EnhancedElement extends ComponentClass {
            static name: string = name;
        }

        /* @ts-ignore */
        EnhancedElement.__initialize();

        /* @ts-ignore */
        return EnhancedElement;
    }

    /* @ts-ignore */
    return decorator;
}

export const staticElement: (name: string, _templatePath: string) => ClassDecorator = element;

export abstract class Component extends HTMLElement {
    static name: string;
    static __templateString?: string;
    static __template: HTMLTemplateElement;

    /* @ts-ignore */
    shadowRoot: ShadowRoot;

    private static defineElement() {
        this.__template = document.createElement("template");

        if (this.__templateString === undefined) {
            console.error(`Template not found for ${this.name}`);
        } else{
            this.__template.innerHTML = this.__templateString;
        }

        /* @ts-ignore */
        customElements.define(this.name, this);
    }

    static __initialize() {
        /* @ts-ignore */
        if (this.constructor.__dev) {
            setTimeout(() => this.defineElement(), 0);
        } else {
            this.defineElement();
        }
    }

    constructor() {
        super();

        /* @ts-ignore */
        if (!this.constructor.__dev) {
            /* @ts-ignore */
            if (this.shadowRoot !== null) {
                return;
            } else if (!HTMLTemplateElement.prototype.hasOwnProperty('shadowRootMode')) {
                // Polyfill for browsers that don't support the `shadowrootmode` property
                const template = this.querySelector("template[shadowrootmode='open']") as HTMLTemplateElement;

                if (template !== null) {
                    const shadowRoot = this.attachShadow({ mode: "open" })

                    shadowRoot.appendChild(template.content);
                    template.remove();

                    return;
                }
            }
        }

        /* @ts-ignore */
        if (!this.constructor.__template) {
            console.error(`Template not found for ${this.name}`);
            return;
        }

        this.shadowRoot = this.attachShadow({ mode: "open" });

        /* @ts-ignore */
        const template = this.constructor.__template.cloneNode(true);

        this.shadowRoot.appendChild(template.content);
    }

    get name(): string {
        /* @ts-ignore */
        return this.constructor.name;
    }
}
