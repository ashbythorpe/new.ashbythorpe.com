package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Comment struct {
	ID        int    `json:"id"`
	Author    User   `json:"author"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type OriginalComment struct {
	Comment
	NumReplies int `json:"numReplies"`
}

type Reply struct {
	Comment
	ReplyTo         *ReplyTo `json:"replyTo,omitempty"`
	OriginalReplyTo int      `json:"originalReplyTo"`
}

type ReplyTo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func CountComments(ctx context.Context, postName string) (int, error) {
	var totalItems int
	query := `SELECT COUNT(*) FROM comments WHERE post_name = ? AND original_reply_to IS NULL`

	err := DB.QueryRowContext(ctx, query, postName).Scan(&totalItems)

	return totalItems, err
}

func GetComments(ctx context.Context, postName string, page int) ([]OriginalComment, error) {
	query := `
	SELECT comments.id, users.id AS author_id, users.name AS author, comments.content, strftime('%Y-%m-%dT%H:%M:%SZ', comments.created_at) as created_at, COUNT(replies.id) AS replies
	FROM comments
	LEFT JOIN users ON comments.user_id = users.id
	LEFT JOIN comments AS replies ON comments.id = replies.original_reply_to
	WHERE comments.post_name = ? AND comments.original_reply_to IS NULL
	GROUP BY comments.id
	ORDER BY comments.created_at DESC
	LIMIT 10 OFFSET ?
	`

	rows, err := DB.QueryContext(ctx, query, postName, (page-1)*10)
	if err != nil {
		return nil, err
	}

	comments := []OriginalComment{}

	defer rows.Close()
	for rows.Next() {
		var comment OriginalComment

		err := rows.Scan(&comment.ID, &comment.Author.ID, &comment.Author.Name, &comment.Content, &comment.CreatedAt, &comment.NumReplies)
		if err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func GetReplies(ctx context.Context, postName string, id int) ([]Reply, error) {
	query := `
	SELECT comments.id, user.id AS author_id, users.name AS author, comments.content, strftime('%Y-%m-%dT%H:%M:%SZ', comments.created_at) as created_at, comments.reply_to, reply_users.name, comments.original_reply_to
	FROM comments
	LEFT JOIN users ON comments.user_id = users.id
	LEFT JOIN comments AS original_comments ON comments.reply_to = original_comments.id
	LEFT JOIN users AS reply_users ON original_comments.user_id = reply_users.id
	WHERE comments.post_name = ? AND comments.original_reply_to = ?
	ORDER BY comments.created_at DESC
	`

	rows, err := DB.QueryContext(ctx, query, postName, id)
	if err != nil {
		return nil, err
	}

	comments := []Reply{}

	defer rows.Close()
	for rows.Next() {
		var comment Reply
		var replyToID *int
		var replyToName *string

		err := rows.Scan(&comment.ID, &comment.Author.ID, &comment.Author.Name, &comment.Content, &comment.CreatedAt, &replyToID, &replyToName, &comment.OriginalReplyTo)
		if err != nil {
			return nil, err
		}

		if replyToID != nil && replyToName != nil {
			comment.ReplyTo = &ReplyTo{*replyToID, *replyToName}
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

type CreateCommentResult struct {
	ID int
	OriginalReplyTo *int
}

func CreateComment(ctx context.Context, postName string, userID uuid.UUID, content string, replyTo *int) (CreateCommentResult, error) {
	var result CreateCommentResult

	if replyTo != nil {
		originalReplyToQuery := `SELECT original_reply_to, post_name FROM comments WHERE id = ?`

		row := DB.QueryRowContext(ctx, originalReplyToQuery, replyTo)

		var post string

		if err := row.Scan(&result.OriginalReplyTo, &post); err != nil {
			return result, err
		}

		if postName != post {
			return result, errors.New("wrong post")
		}

		if result.OriginalReplyTo == nil {
			result.OriginalReplyTo = replyTo
		}
	}

	query := `
		INSERT INTO comments (post_name, user_id, content, reply_to, original_reply_to) VALUES (?, ?, ?, ?, ?) RETURNING id
	`

	row := DB.QueryRowContext(ctx, query, postName, userID[:], content, replyTo, result.OriginalReplyTo)
	err := row.Scan(&result.ID)

	return result, err
}

func EditComment(ctx context.Context, id int, userID uuid.UUID, content string) (*int, error) {
	query := `
		UPDATE comments
		SET content = ?
		WHERE id = ? AND user_id = ?
		RETURNING original_reply_to
	`

	row := DB.QueryRowContext(ctx, query, content, id, userID[:])
	var originalReplyTo int
	if err := row.Scan(&originalReplyTo); err != nil {
		return nil, err
	}

	return &originalReplyTo, nil
}

func DeleteComment(ctx context.Context, id int, userID uuid.UUID) (*int, error) {
	query := "DELETE FROM comments WHERE id = ? AND user_id = ? RETURNING original_reply_to"

	row := DB.QueryRowContext(ctx, query, id, userID[:])
	var originalReplyTo int
	if err := row.Scan(&originalReplyTo); err != nil {
		return nil, err
	}

	return &originalReplyTo, nil
}
