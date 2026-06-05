import "./components/navbar/index.ts";
import "./components/pagination/index.ts";
import "./components/create-comment/index.ts";
import { BlogComment } from "./components/comment/index.ts";
import "./styles/post.css";
import { postName } from "./routes.ts";
import type { OriginalComment } from "./types.ts";
import type CreateComment from "./components/create-comment/index.ts";

interface CommentsResult {
    totalComments: number;
    comments: OriginalComment[];
}

const commentsContainer = document.querySelector("#comments-container");
const paginationElement = document.querySelector("blog-pagination");

async function renderPage(page: number) {
    const { totalComments, comments }: CommentsResult = await fetch(
        `/api/post/${postName()}/comments?page=${page}`,
    ).then((x) => x.json());

    console.log(comments);

    commentsContainer?.replaceChildren(
        ...comments.map((comment) =>
            BlogComment.create({
                id: comment.id,
                content: comment.content,
                author: comment.author,
                time: duration(new Date(comment.createdAt)),
                owned: comment.owned,
                numReplies: comment.numReplies,
            }),
        ),
    );

    paginationElement?.setAttribute("current-page", page.toString());
    paginationElement?.setAttribute(
        "total-pages",
        Math.max(Math.ceil(totalComments / 10), 1).toString(),
    );
}

const page = parseInt(
    new URL(window.location.href).searchParams.get("page") ?? "1",
    10,
);
renderPage(page);

paginationElement?.addEventListener("page-change", (event) => {
    const page = (event as CustomEvent).detail.page;

    renderPage(page);

    window.history.replaceState({}, "", `?page=${page}`);
});

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

const navbar = document.querySelector("nav-bar") as HTMLElement;
const createCommentElement = document.querySelector(
    "create-comment",
) as CreateComment;

async function checkSignedIn() {
    const { name } = await fetch("/api/auth/name").then((x) => x.json());

    console.log(name);

    if (name !== null) {
        navbar.setAttribute("signed-in", "");
        createCommentElement.setAttribute("signed-in", "");
        createCommentElement.addSlot("username", name);
    }
}

checkSignedIn();
