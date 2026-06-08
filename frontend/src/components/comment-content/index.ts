import { Component, element } from "../../custom-elements";

@element("comment-content", "./index.html")
export default class CommentContent extends Component {
    static create(): CommentContent {
        const element = document.createElement("comment-content") as CommentContent;

        return element;
    }

    setContent(content: string | Node) {
        const container = this.select("#content");

        if (typeof content === "string") {
            container.textContent = content;
        } else {
            container.replaceChildren(content);
        }
    }
}
