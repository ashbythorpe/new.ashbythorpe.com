import { Component, element } from "../../custom-elements";

export interface CommentSubmitEvent {
    content: string;
    editId: string | null;
    replyId: string | null;
}

@element("create-comment", "./index.html")
export default class CreateComment extends Component {
    static observedAttributes = [
        "edit-id",
        "reply-id",
        "reply-username",
        "signed-in", // New attribute
    ];

    connectedCallback(): void {
        this.setupEventListeners();

        // Initialize auth state if the attribute is missing on load
        if (!this.hasAttribute("signed-in")) {
            this.signedIn = false;
        }
    }

    attributeChangedCallback(
        name: string,
        _oldValue: string | null,
        newValue: string | null,
    ): void {
        if (name === "edit-id") this.editing = !!newValue;
        if (name === "reply-id") this.replying = !!newValue;
        if (name === "signed-in") this.signedIn = newValue !== null;

        if (name === "reply-username") {
            const replySpan = this.select("#reply-username");
            if (replySpan) replySpan.textContent = newValue || "";
        }
    }

    // --- State Management (Updated to Modern Spec) ---
    get editing(): boolean {
        return this.internals.states.has("editing");
    }
    set editing(isEditing: boolean) {
        if (isEditing) this.internals.states.add("editing");
        else this.internals.states.delete("editing");
    }

    get replying(): boolean {
        return this.internals.states.has("replying");
    }
    set replying(isReplying: boolean) {
        if (isReplying) this.internals.states.add("replying");
        else this.internals.states.delete("replying");
    }

    // If they are NOT signed in, we add the "signed-out" state to trigger the CSS
    get signedIn(): boolean {
        return !this.internals.states.has("signed-out");
    }
    set signedIn(isSignedIn: boolean) {
        if (isSignedIn) this.internals.states.delete("signed-out");
        else this.internals.states.add("signed-out");
    }

    // --- Content / Pending Getters ---
    get content(): string {
        return this.select<HTMLTextAreaElement>("#content-input").value;
    }
    set content(val: string) {
        this.select<HTMLTextAreaElement>("#content-input").value = val;
    }

    get pending(): boolean {
        return this.select<HTMLButtonElement>("#submit-btn").disabled;
    }
    set pending(isPending: boolean) {
        const btn = this.select<HTMLButtonElement>("#submit-btn");
        btn.disabled = isPending;
        btn.setAttribute("aria-disabled", String(isPending));
    }

    // --- Public Methods ---
    public setErrors(
        fieldErrors: string[] = [],
        globalMessage: string | null = null,
    ): void {
        const fieldContainer = this.select("#field-errors");
        const formContainer = this.select("#form-errors");

        fieldContainer.innerHTML = fieldErrors
            .map((err) => `<p class="error-text">${err}</p>`)
            .join("");
        formContainer.innerHTML = globalMessage
            ? `<p class="error-text">${globalMessage}</p>`
            : "";
    }

    public resetForm(): void {
        this.content = "";
        this.setErrors();
        this.removeAttribute("edit-id");
        this.removeAttribute("reply-id");
        this.removeAttribute("reply-username");
    }

    // --- Internal Events ---
    private setupEventListeners(): void {
        const form = this.select<HTMLFormElement>("#comment-form");
        const cancelEdit = this.select<HTMLButtonElement>("#cancel-edit-btn");
        const cancelReply = this.select<HTMLButtonElement>("#cancel-reply-btn");
        const textarea = this.select<HTMLTextAreaElement>("#content-input");
        const signInBtn = this.select<HTMLAnchorElement>(
            "#trigger-sign-in-btn",
        );

        const redirect = window.location.pathname;
        const params = new URLSearchParams();
        params.append("redirect", redirect);
        signInBtn.href = `/auth/sign-in/?${params.toString()}`;

        // Clear errors on typing
        textarea.addEventListener("input", () => this.setErrors());

        cancelEdit.addEventListener("click", () => {
            this.removeAttribute("edit-id");
            this.dispatchEvent(
                new CustomEvent("cancel-edit", {
                    bubbles: true,
                    composed: true,
                }),
            );
        });

        cancelReply.addEventListener("click", () => {
            const oldReplyId = this.getAttribute("reply-id");
            this.removeAttribute("reply-id");
            this.removeAttribute("reply-username");

            this.dispatchEvent(
                new CustomEvent("cancel-reply", {
                    bubbles: true,
                    composed: true,
                    detail: { previousReplyId: oldReplyId },
                }),
            );
        });

        form.addEventListener("submit", (e) => {
            e.preventDefault();

            const content = this.content.trim();
            if (!content) return;

            const editId = this.getAttribute("edit-id");
            const replyId = this.getAttribute("reply-id");

            this.pending = true;

            this.dispatchEvent(
                new CustomEvent<CommentSubmitEvent>("comment-submit", {
                    bubbles: true,
                    composed: true,
                    detail: { content, editId, replyId },
                }),
            );
        });
    }
}
