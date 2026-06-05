import { Component, element } from "../../custom-elements";
import { duration } from "../../post";
import { postName } from "../../routes";
import type { Reply } from "../../types";
import { BlogComment } from "../comment";

export interface Params {
    id: number;
    numReplies: number;
}

@element("reply-list", "./index.html")
export class ReplyList extends Component {
    #id!: number;
    #loading: boolean = false;
    #loaded: boolean = false;

    static create({ id, numReplies }: Params): ReplyList {
        const element = document.createElement("reply-list") as ReplyList;

        element.#id = id;

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

                        replyList.replaceChildren(
                            ...replies.map((reply) =>
                                BlogComment.create({
                                    id: reply.id,
                                    content: reply.content,
                                    author: reply.author,
                                    time: duration(new Date(reply.createdAt)),
                                    owned: reply.owned,
                                    replyTo: reply.replyTo,
                                    originalReplyTo: reply.originalReplyTo,
                                }),
                            ),
                        );

                        this.#loaded = true;
                    } catch {
                        this.internals.states.delete("open");
                    }

                    this.#loading = false;
                    this.internals.states.delete("loading");
                }
            }
        });
    }
}
