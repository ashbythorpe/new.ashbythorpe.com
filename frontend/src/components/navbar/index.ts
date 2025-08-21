import { Component, staticElement } from "../../custom-elements";
import "../icons/home.ts";
import "../icons/info.ts";
import "../icons/code-bracket.ts";
import "../icons/pencil-square.ts";
import "../icons/github.ts";

@staticElement("nav-bar", "./index.html")
export default class extends Component {
    static observedAttributes = ["selected"];

    attributeChangedCallback(
        name: string,
        oldValue: string | null,
        newValue: string | null,
    ) {
        if (name === "selected") {
            if (oldValue === newValue) return;

            if (oldValue !== null) {
                const item = this.shadowRoot.querySelector(
                    `#navbar-${oldValue}`,
                );

                if (item) {
                    item.classList.remove("selected");
                }
            }

            if (newValue !== null) {
                const item = this.shadowRoot.querySelector(
                    `#navbar-${newValue}`,
                );

                if (item) {
                    item.classList.add("selected");
                } else {
                    console.warn("Invalid `selected` value");
                }
            }
        }
    }
}
