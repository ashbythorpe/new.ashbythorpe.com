package handlers

import (
	"errors"
	"strconv"

	"ashbythorpe.com/website/db"
	"github.com/gofiber/fiber/v3"
)

func SetupCommentRoutes(app *fiber.App) {
	group := app.Group("/post")
	group.Get("/:post/comments", userIDmiddleware, getComments)
	group.Get("/:post/replies/:id", userIDmiddleware, getReplies)
	group.Post("/:post/create-comment", authMiddleware, createComment)
	group.Post("/:post/edit-comment/:id", authMiddleware, editComment)
	group.Post("/:post/delete-comment/:id", authMiddleware, deleteComment)
}

type CommentsResult struct {
	TotalComments int                  `json:"totalComments"`
	Comments      []db.OriginalComment `json:"comments"`
}

func getComments(c fiber.Ctx) error {
	postName := c.Params("post")
	userID := c.Locals("userID", 0).(int)
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil {
		return err
	}

	if page <= 0 {
		return errors.New("invalid page")
	}

	var result CommentsResult

	comments, err := db.GetComments(c, postName, userID, page)
	if err != nil {
		return err
	}

	result.Comments = comments

	count, err := db.CountComments(c, postName)
	if err != nil {
		return err
	}

	result.TotalComments = count

	return c.JSON(result)
}

func getReplies(c fiber.Ctx) error {
	// time.Sleep(5 * time.Second)
	postName := c.Params("post")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	userID := c.Locals("userID", 0).(int)

	replies, err := db.GetReplies(c.RequestCtx(), postName, id, userID)
	if err != nil {
		return err
	}

	return c.JSON(replies)
}

type CommentOpts struct {
	Text    string `json:"text"`
	ReplyTo *int   `json:"replyTo"`
}

type CommentResult struct {
	ID int `json:"id"`
}

func createComment(c fiber.Ctx) error {
	postName := c.Params("post")
	userID := c.Locals("userID", 0).(int)

	var opts CommentOpts
	if err := c.Bind().Body(&opts); err != nil {
		return err
	}

	id, err := db.CreateComment(c.RequestCtx(), postName, userID, opts.Text, opts.ReplyTo)
	if err != nil {
		return err
	}

	return c.JSON(CommentResult{id})
}

type EditOpts struct {
	Text string `json:"text"`
}

func editComment(c fiber.Ctx) error {
	userID := c.Locals("userID", 0).(int)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	var opts EditOpts
	if err := c.Bind().Body(&opts); err != nil {
		return err
	}

	return db.EditComment(c.RequestCtx(), id, userID, opts.Text)
}

func deleteComment(c fiber.Ctx) error {
	userID := c.Locals("userID", 0).(int)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	return db.DeleteComment(c.RequestCtx(), id, userID)
}
