import { Component, element } from "../../custom-elements";
import { duration } from "../../post";
import { postName } from "../../routes";
import type { CommentData, Reply } from "../../types";
import { BlogComment } from "../comment";

export interface Params {
    userID: string | null;
    id: number;
    numReplies: number;
}

@element("reply-list", "./index.html")
export class ReplyList extends Component {
    #id!: number;
    #userID!: string | null;
    #loading: boolean = false;
    #loaded: boolean = false;
    #numReplies: number = 0;

    static create({ id, userID, numReplies }: Params): ReplyList {
        const element = document.createElement("reply-list") as ReplyList;

        element.#id = id;
        element.#userID = userID;
        element.#numReplies = numReplies;

        element.addSlot("num-replies", numReplies.toString());

        return element;
    }

    connectedCallback(): void {
        const replyList = this.select("#replies-list");
        const showReplies = this.select("#show-replies");

        showReplies.addEventListener("click", async () => {
            if (this.internals.states.has("open")) {
                this.internals.states.delete("open");
                this.internals.states.delete("loading");
            } else {
                this.internals.states.add("open");
                if (!this.#loaded) {
                    this.internals.states.add("loading");
                }

                if (!this.#loading && !this.#loaded) {
                    try {
                        const replies: Reply[] = await fetch(
                            `/api/post/${postName()}/replies/${this.#id}`,
                        ).then((x) => x.json());

                        for (const reply of replies) {
                            if (
                                replyList.querySelector(
                                    `#comment-${reply.id}`,
                                ) === null
                            ) {
                                replyList.appendChild(
                                    BlogComment.create({
                                        id: reply.id,
                                        userID: this.#userID,
                                        content: reply.content,
                                        author: reply.author,
                                        time: duration(
                                            new Date(reply.createdAt),
                                        ),
                                        replyTo: reply.replyTo,
                                        originalReplyTo: reply.originalReplyTo,
                                    }),
                                );
                            }
                        }

                        this.#loaded = true;
                    } catch {
                        this.internals.states.delete("open");
                    }

                    this.#loading = false;
                    this.internals.states.delete("loading");
                }
            }
        });

        this.addEventListener("comment-deleted", () => {
            this.numReplies -= 1;

            if (this.numReplies == 0) {
                this.remove();
            }
        });
    }

    get numReplies() {
        return this.#numReplies;
    }

    set numReplies(value: number) {
        this.#numReplies = value;

        const numRepliesSlot = this.querySelector("span[slot='num-replies']");
        if (numRepliesSlot) {
            numRepliesSlot.textContent = this.#numReplies.toString();
        }
    }

    addReply(reply: CommentData) {
        const replyList = this.select("#replies-list");

        replyList.appendChild(BlogComment.create(reply));
        this.numReplies += 1;
    }
}
