package db

import (
	"context"
	"errors"
)

type Comment struct {
	ID        int    `json:"id"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
	Owned     bool   `json:"owned"`
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

func GetComments(ctx context.Context, postName string, userID int, page int) ([]OriginalComment, error) {
	query := `
	SELECT comments.id, users.name AS author, comments.text, strftime('%Y-%m-%dT%H:%M:%SZ', comments.created_at) as created_at, comments.userID = ? as owned, COUNT(replies.id) AS replies
	FROM comments
	LEFT JOIN users ON comments.userID = users.id
	LEFT JOIN comments AS replies ON comments.id = replies.original_reply_to
	WHERE comments.post_name = ? AND comments.original_reply_to IS NULL
	GROUP BY comments.id
	ORDER BY comments.created_at DESC
	LIMIT 20 OFFSET ?
	`

	rows, err := DB.QueryContext(ctx, query, userID, postName, (page-1)*20)
	if err != nil {
		return nil, err
	}

	comments := []OriginalComment{}

	defer rows.Close()
	for rows.Next() {
		var comment OriginalComment

		err := rows.Scan(&comment.ID, &comment.Author, &comment.Text, &comment.CreatedAt, &comment.Owned, &comment.NumReplies)
		if err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func GetReplies(ctx context.Context, postName string, id int, userID int) ([]Reply, error) {
	query := `
	SELECT comments.id, users.name, comments.text, strftime('%Y-%m-%dT%H:%M:%SZ', comments.created_at) as created_at, comments.userID = ? as owned, comments.reply_to, reply_users.name, comments.original_reply_to
	FROM comments
	LEFT JOIN users ON comments.userID = users.id
	LEFT JOIN comments AS original_comments ON comments.reply_to = original_comments.id
	LEFT JOIN users AS reply_users ON original_comments.userID = reply_users.id
	WHERE comments.post_name = ? AND comments.original_reply_to = ?
	ORDER BY comments.created_at DESC
	`

	rows, err := DB.QueryContext(ctx, query, userID, postName, id)
	if err != nil {
		return nil, err
	}

	comments := []Reply{}

	defer rows.Close()
	for rows.Next() {
		var comment Reply
		var replyToID *int
		var replyToName *string

		err := rows.Scan(&comment.ID, &comment.Author, &comment.Text, &comment.CreatedAt, &comment.Owned, &replyToID, &replyToName, &comment.OriginalReplyTo)
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

func CreateComment(ctx context.Context, postName string, userID int, text string, replyTo *int) (int, error) {
	var originalReplyTo *int

	if replyTo != nil {
		originalReplyToQuery := `SELECT original_reply_to, post_name FROM comments WHERE id = ?`

		row := DB.QueryRowContext(ctx, originalReplyToQuery, replyTo)

		var post string

		if err := row.Scan(&originalReplyTo, &post); err != nil {
			return 0, err
		}

		if postName != post {
			return 0, errors.New("wrong post")
		}

		if originalReplyTo == nil {
			originalReplyTo = replyTo
		}
	}

	query := `
		INSERT INTO comments (post_name, userID, text, reply_to, original_reply_to) VALUES (?, ?, ?, ?, ?) RETURNING id
	`

	var id int
	row := DB.QueryRowContext(ctx, query, postName, userID, text, replyTo, originalReplyTo)
	err := row.Scan(&id)

	return id, err
}

func EditComment(ctx context.Context, id int, userID int, text string) error {
	query := `
		UPDATE comments
		SET text = ?
		WHERE id = ? AND userID = ?
	`

	res, err := DB.ExecContext(ctx, query, text, id, userID)
	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return errors.New("invalid comment")
	}

	return nil
}

func DeleteComment(ctx context.Context, id int, userID int) error {
	query := "DELETE FROM comments WHERE id = ? AND userID = ?"

	res, err := DB.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return errors.New("invalid comment")
	}

	return nil
}
