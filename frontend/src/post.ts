import "./components/navbar/index.ts";
import { BlogComment } from "./components/comment/index.ts";
import "./styles/post.css";
import { postName } from "./routes.ts";
import type { Comment, CommentData, OriginalComment } from "./types.ts";

interface CommentsResult {
    totalComments: number;
    comments: OriginalComment[];
}

const comments: CommentsResult = await fetch(
    `/api/post/${postName()}/comments`,
).then((x) => x.json());
console.log(comments);

const commentsContainer = document.querySelector("#comments-container");

commentsContainer?.replaceChildren(
    ...comments.comments.map((comment) =>
        BlogComment.create({
            id: comment.id,
            content: comment.text,
            author: comment.author,
            time: duration(new Date(comment.createdAt)),
            owned: comment.owned,
            numReplies: comment.numReplies,
        }),
    ),
);

export function duration(date: Date): string {
    const diffInMs = date.getTime() - Date.now(); // Negative for past
    const diffInDays = Math.round(diffInMs / (1000 * 60 * 60 * 24));

    // Create the formatter (uses browser language by default)
    const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

    if (diffInDays === 0) return "Today";
    if (diffInDays === -1) return "Yesterday";

    if (Math.abs(diffInDays) < 7) return rtf.format(diffInDays, "day");
    if (Math.abs(diffInDays) < 30)
        return rtf.format(Math.round(diffInDays / 7), "week");
    if (Math.abs(diffInDays) < 365)
        return rtf.format(Math.round(diffInDays / 30), "month");

    return rtf.format(Math.round(diffInDays / 365), "year");
}
