export interface CommentData {
    id: number;
    userID: string | null;
    content: string;
    author: User;
    time: string;
    replyTo?: {
        id: number;
        name: string;
    };
    originalReplyTo?: number;
    numReplies?: number;
}

export interface Comment {
    id: number;
    author: User;
    content: string;
    createdAt: string;
}

export interface OriginalComment extends Comment {
    numReplies: number;
}

export interface Reply extends Comment {
    replyTo?: {
        id: number;
        name: string;
    };
    originalReplyTo: number;
}

export interface User {
    id: string;
    name: string;
}

