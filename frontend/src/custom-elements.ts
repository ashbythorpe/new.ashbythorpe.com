export function element(name: string, _templatePath: string): ClassDecorator {
    function decorator<T extends Component>(ComponentClass: T): T {
        /* @ts-ignore */
        ComponentClass.__name = name;

        /* @ts-ignore */
        ComponentClass.__initialize();

        /* @ts-ignore */
        return ComponentClass;
    }

    /* @ts-ignore */
    return decorator;
}

export function staticElement<TFunction extends Function>(
    target: TFunction,
): TFunction | void {
    return target;
}

export abstract class Component extends HTMLElement {
    static __name: string;
    static __templateString?: string;
    static __template: HTMLTemplateElement;

    /* @ts-ignore */
    shadowRoot: ShadowRoot;
    protected internals!: ElementInternals;

    private static defineElement() {
        if (this.__templateString !== undefined) {
            this.__template = document.createElement("template");
            this.__template.innerHTML = this.__templateString;
        }

        /* @ts-ignore */
        customElements.define(this.__name, this);
    }

    static __initialize() {
        /* @ts-ignore */
        if (import.meta.env.DEV) {
            console.log("Go!");
            this.defineElement();
            // setTimeout(() => this.defineElement(), 0);
        } else {
            this.defineElement();
        }
    }

    constructor() {
        super();
        console.log("I'm in here!");

        /* @ts-ignore */
        if (HTMLElement.prototype.hasOwnProperty("attachInternals")) {
            this.internals = this.attachInternals();

            if (this.internals.shadowRoot) {
                this.shadowRoot = this.internals.shadowRoot;
                return;
            }
        }

        if (!HTMLTemplateElement.prototype.hasOwnProperty("shadowRootMode")) {
            // Polyfill for browsers that don't support the `shadowrootmode` property
            const template = this.querySelector(
                "template[shadowrootmode='open']",
            ) as HTMLTemplateElement;

            if (template !== null) {
                console.log("Polyfilling shadowrootmode='open'");
                this.shadowRoot = this.attachShadow({ mode: "open" });

                this.shadowRoot.appendChild(template.content);
                template.remove();

                return;
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
        return this.constructor.__name;
    }

    protected select<T extends Element = HTMLElement>(selectors: string): T {
        const element = this.shadowRoot.querySelector(selectors);

        if (element === null) {
            throw new Error(`Element with selector '${selectors}' not found`);
        }

        return element as T;
    }

    protected selectAll<T extends Element = HTMLElement>(
        selectors: string,
    ): NodeListOf<T> {
        return this.shadowRoot.querySelectorAll(selectors) as NodeListOf<T>;
    }

    protected addSlot(slot: string | null, content: Node | string) {
        let node;
        if (typeof content === "string") {
            if (slot === null) {
                node = document.createTextNode(content);
            } else {
                node = document.createElement("span");
                node.textContent = content;
            }
        } else {
            node = content;
        }

        if (slot !== null && node instanceof Element) {
            node.slot = slot;
        }

        this.appendChild(node);
    }

    connectedCallback?(): void;
    disconnectedCallback?(): void;
    connectedMoveCallback?(): void;
    adoptedCallback?(): void;
    attributeChangedCallback?(
        name: string,
        oldValue: string | null,
        newValue: string | null,
    ): void;
}
