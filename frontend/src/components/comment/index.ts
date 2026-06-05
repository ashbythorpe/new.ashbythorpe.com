import { Component, element } from "../../custom-elements";
import { postName } from "../../routes";
import type { CommentData } from "../../types";
import { ReplyList } from "../replies";

@element("blog-comment", "./index.html")
export class BlogComment extends Component {
    static observedAttributes = ["comment-id"];

    #id!: number;
    #originalReplyTo?: number;

    static create({
        id,
        content,
        author,
        time,
        owned,
        replyTo,
        originalReplyTo,
        numReplies,
    }: CommentData): BlogComment {
        const element = document.createElement("blog-comment") as BlogComment;

        console.log(element);
        console.log(element instanceof BlogComment);

        element.#id = id;
        element.#originalReplyTo = originalReplyTo;

        element.id = `comment-${id}`;
        element.setAttribute("comment-id", String(id));

        if (owned) {
            element.internals.states.add("owned");
        }

        element.addSlot("content", content);

        if (replyTo !== undefined) {
            const replyAnchor = document.createElement("a");
            replyAnchor.href = `#comment-${replyTo.id}`;
            replyAnchor.textContent = `@${replyTo.name} `;
            element.addSlot("reply-link", replyAnchor);
        }

        // Author slot
        element.addSlot("author", author);
        element.addSlot("time", time);

        if (numReplies !== undefined && numReplies !== 0) {
            const replies = ReplyList.create({ id, numReplies });
            element.addSlot("replies", replies);
        }

        return element;
    }

    // Property getter/setter for the internal state
    get owned(): boolean {
        return this.internals.states.has("owned");
    }

    set owned(isOwned: boolean) {
        if (isOwned) {
            this.internals.states.add("owned");
        } else {
            this.internals.states.delete("owned");
        }
    }

    connectedCallback(): void {
        const self_link = this.select<HTMLButtonElement>("#self-link");
        self_link.addEventListener("click", async () => {
            console.log("Disabling");
            self_link.disabled = true;

            const url = new URL(
                window.location.origin + window.location.pathname,
            );

            if (this.#originalReplyTo !== undefined) {
                url.searchParams.append(
                    "open",
                    this.#originalReplyTo.toString(),
                );
            }

            url.hash = `comment-${this.#id}`;

            try {
                await navigator.clipboard.writeText(url.toString());

                this.internals.states.add("copied");

                setTimeout(() => {
                    self_link.disabled = false;
                    this.internals.states.delete("copied");
                }, 1000);
            } catch {
                self_link.disabled = false;
            }
        });

        this.select("#reply-btn").addEventListener("click", () => {
            this.dispatchEvent(
                new CustomEvent("comment-reply", {
                    bubbles: true,
                    composed: true,
                }),
            );
        });

        const editInput = this.select<HTMLTextAreaElement>("#edit-input");

        this.select("#edit-btn").addEventListener("click", () => {
            // Grab the current text from the light DOM slot and put it in the textarea
            const contentSlot = this.querySelector('[slot="content"]');
            if (contentSlot) {
                editInput.value = contentSlot.textContent || "";
            }
            this.internals.states.add("editing");
            // Optional: Focus the input automatically
            setTimeout(() => editInput.focus(), 0);
        });

        this.select("#delete-btn").addEventListener("click", async () => {
            const result = await fetch(
                `/api/post/${postName()}/comment/${this.#id}`,
                {
                    method: "DELETE",
                    headers: {
                        "Content-Type": "application/json",
                    },
                },
            );

            if (!result.ok) {
                throw new Error(await result.text());
            }

            this.remove();
        });

        this.select("#cancel-edit-btn").addEventListener("click", () => {
            this.internals.states.delete("editing");
        });

        this.select("#save-edit-btn").addEventListener("click", async () => {
            const newText = editInput.value.trim();
            if (!newText) return; // Prevent saving empty comments

            // Optimistically update the UI in the light DOM
            const contentSlot = this.querySelector('[slot="content"]');
            if (contentSlot) {
                contentSlot.textContent = newText;
            }

            // Close the edit state
            this.internals.states.delete("editing");

            await fetch(`/api/post/${postName()}/edit-comment/${this.#id}`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({ text: newText }),
            });
        });
    }

    // attributeChangedCallback(
    //     name: string,
    //     _oldValue: string | null,
    //     newValue: string | null,
    // ): void {
    //     if (name === "comment-id" && newValue !== null) {
    //         const link: HTMLAnchorElement = this.select("#self-link");
    //         // link.href = `#comment-${newValue}`;
    //     }
    // }
}
