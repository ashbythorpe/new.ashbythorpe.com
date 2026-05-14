export interface CommentData {
    id: number;
    content: string;
    author: string;
    time: string;
    owned: boolean;
    replyTo?: {
        id: number;
        name: string;
    };
    originalReplyTo?: number;
    numReplies?: number;
}

export interface Comment {
    id: number;
    author: string;
    text: string;
    createdAt: string;
    owned: boolean
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
