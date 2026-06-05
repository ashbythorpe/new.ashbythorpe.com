import { Component, element, staticElement } from "../../custom-elements";

@staticElement
@element("nav-bar", "./index.html")
export default class extends Component {
    static observedAttributes = ["selected", "signed-in"];

    connectedCallback(): void {
        const logoutBtn = this.select<HTMLButtonElement>("#logout-btn");

        logoutBtn.addEventListener("click", async () => {
            await fetch("/api/auth/logout", {
                method: "POST",
            });
            window.location.reload();
        });
    }

    attributeChangedCallback(
        name: string,
        _oldValue: string | null,
        newValue: string | null,
    ): void {
        if (name === "signed-in") {
            this.signedIn = newValue !== null;
        }
    }

    get signedIn(): boolean {
        return this.internals.states.has("signed-in");
    }

    set signedIn(isSignedIn: boolean) {
        if (isSignedIn) this.internals.states.add("signed-in");
        else this.internals.states.delete("signed-in");
    }
}
