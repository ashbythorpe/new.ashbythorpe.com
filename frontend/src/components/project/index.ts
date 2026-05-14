import { Component, element } from "../../custom-elements";

@element("project-card", "./index.html")
export default class extends Component {
    static observedAttributes = ["href"];

    attributeChangedCallback(
        name: string,
        _oldValue: string | null,
        newValue: string | null,
    ): void {
        if (name === "href" && newValue !== null) {
            const link: HTMLAnchorElement = this.select("#link");
            link.href = newValue;
        }
    }
}
