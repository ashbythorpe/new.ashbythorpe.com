import { Component, element, staticElement } from "../../custom-elements";

@staticElement
@element("blog-card", "./index.html")
export default class extends Component {
    static observedAttributes = ["href"];

    attributeChangedCallback(
        name: string,
        _oldValue: string | null,
        newValue: string | null,
    ): void {
        if (name === "href" && newValue !== null) {
            const link: HTMLAnchorElement = this.select("#main-link");
            link.href = newValue;
        }
    }
}
