import { Component, element } from "../../custom-elements";
import { duration } from "../../post";
import { postName } from "../../routes";
import type { CommentData, User } from "../../types";

interface CreateCommentResult {
    id: number;
    createdAt: string;
    originalReplyTo?: number;
}

@element("create-comment", "./index.html")
export default class CreateComment extends Component {
    static observedAttributes = [
        "edit-id",
        "reply-id",
        "reply-username",
        "signed-in",
    ];

    #user: User | null = null;
    #replyUsername: string | null = null;
    #turnstileToken: string | null = null;

    connectedCallback(): void {
        this.setupEventListeners();
        this.#setupTurnstile();
    }

    attributeChangedCallback(
        name: string,
        _oldValue: string | null,
        newValue: string | null,
    ): void {
        if (name === "edit-id") this.editing = newValue != null;
        if (name === "reply-id") this.replying = newValue != null;

        if (name === "reply-username") {
            console.log(newValue);
            this.#replyUsername = newValue;
            const replySpan = this.select("#reply-username");
            if (replySpan) replySpan.textContent = newValue || "";
        }
    }

    setUser(user: User) {
        this.#user = user;
    }

    get signedIn() {
        if (this.internals.states.has("signed-in")) {
            return true;
        } else if (this.internals.states.has("signed-out")) {
            return false;
        } else {
            return null;
        }
    }

    set signedIn(signedIn: boolean | null) {
        if (signedIn === true) {
            this.internals.states.add("signed-in");
        } else {
            this.internals.states.delete("signed-in");
        }

        if (signedIn === false) {
            this.internals.states.add("signed-out");
        } else {
            this.internals.states.delete("signed-out");
        }
    }

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

    #resetForm(): void {
        this.content = "";
        this.removeAttribute("edit-id");
        this.removeAttribute("reply-id");
        this.removeAttribute("reply-username");
    }

    #setupTurnstile() {
        turnstile.render("#log-in-turnstile", {
            sitekey: import.meta.env.VITE_TURNSTILE_INVISIBLE_SITE_KEY,
            callback: (token) => {
                this.#turnstileToken = token;
            },
        });
    }

    // --- Internal Events ---
    private setupEventListeners(): void {
        const form = this.select<HTMLFormElement>("#comment-form");
        const cancelReply = this.select<HTMLButtonElement>("#cancel-reply-btn");
        const textarea = this.select<HTMLTextAreaElement>("#content-input");
        const signInBtn = this.select<HTMLAnchorElement>(
            "#trigger-sign-in-btn",
        );
        const errorElement = this.select("#error");

        const redirect = window.location.pathname;
        const params = new URLSearchParams();
        params.append("redirect", redirect);
        signInBtn.href = `/auth/sign-in/?${params.toString()}`;

        textarea.addEventListener("input", () => {
            errorElement.textContent = "";
        });

        cancelReply.addEventListener("click", () => {
            this.removeAttribute("reply-id");
            this.removeAttribute("reply-username");
        });

        form.addEventListener("submit", async (e) => {
            e.preventDefault();

            const content = this.content.trim();
            if (!content) return;

            const replyIdString = this.getAttribute("reply-id");
            console.log(replyIdString);
            const replyId = replyIdString ? parseInt(replyIdString) : null;

            this.pending = true;

            let result;
            try {
                result = await this.#createComment(content, replyId);
            } catch (e) {
                if (e instanceof Error) {
                    errorElement.textContent = e.message;
                }
                this.pending = false;
                return;
            }

            this.pending = false;
            const replyUsername = this.#replyUsername;

            this.#resetForm();

            const comment: CommentData = {
                id: result.id,
                userID: this.#user!.id,
                content,
                author: this.#user!,
                time: duration(new Date(result.createdAt)),
                replyTo:
                    replyId && replyUsername
                        ? {
                              id: replyId,
                              name: replyUsername,
                          }
                        : undefined,
                originalReplyTo: result.originalReplyTo,
            };

            this.dispatchEvent(
                new CustomEvent("new-comment", {
                    bubbles: true,
                    composed: true,
                    detail: comment,
                }),
            );
        });
    }

    async #createComment(
        content: string,
        replyId: number | null,
    ): Promise<CreateCommentResult> {
        const body: any = {
            content,
            turnstileToken: this.#turnstileToken,
        };
        if (replyId !== null) {
            body.replyId = replyId;
        }

        let response;
        try {
            response = await fetch(`/api/post/${postName()}/create-comment`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(body),
            });
        } catch (e) {
            throw new Error("Network error");
        }

        if (!response.ok) {
            const { message } = await response.json();
            throw new Error(message);
        }

        return await response.json();
    }
}
