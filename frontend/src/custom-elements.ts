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

export function staticElement<TFunction extends Function>(target: TFunction): TFunction | void {
    return target;
}

export function noShadowRoot<TFunction extends Function>(target: TFunction): TFunction | void {
    /* @ts-ignore */
    target.__shadowRoot = false;

    return target;
}

export abstract class Component extends HTMLElement {
    static name: string;
    static __shadowRoot: boolean = true;
    static __templateString?: string;
    static __template: HTMLTemplateElement;

    /* @ts-ignore */
    shadowRoot: ShadowRoot;

    private static defineElement() {
        if (this.__templateString !== undefined) {
            this.__template = document.createElement("template");
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
            if (HTMLElement.prototype.hasOwnProperty("attachInternals")) {
                const internals = this.attachInternals();

                if (internals.shadowRoot) {
                    this.shadowRoot = internals.shadowRoot;
                    return;
                }
            }

            if (
                !HTMLTemplateElement.prototype.hasOwnProperty("shadowRootMode")
            ) {
                // Polyfill for browsers that don't support the `shadowrootmode` property
                const template = this.querySelector(
                    "template[shadowrootmode='open']",
                ) as HTMLTemplateElement;

                if (template !== null) {
                    console.log("Polyfill time!");
                    this.shadowRoot = this.attachShadow({ mode: "open" });

                    this.shadowRoot.appendChild(template.content);
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


        /* @ts-ignore */
        if (!this.constructor.__shadowRoot) {
            return;
        }

        this.shadowRoot = this.attachShadow({ mode: "open" });

        /* @ts-ignore */
        const template = this.constructor.__template.cloneNode(true);

        this.shadowRoot.appendChild(template.content);
    }

    connectedCallback() {
        /* @ts-ignore */
        if (!this.constructor.__shadowRoot && this.constructor.__dev) {
            /* @ts-ignore */
            const template = this.constructor.__template.cloneNode(true)
            this.append(template.content);
        }
    }

    get name(): string {
        /* @ts-ignore */
        return this.constructor.name;
    }
}
