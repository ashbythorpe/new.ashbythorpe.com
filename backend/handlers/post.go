package handlers

import (
	"errors"
	"strconv"
	"unicode/utf8"

	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func SetupCommentRoutes(app *fiber.App) {
	group := app.Group("/post")
	group.Get("/:post/comments", getComments)
	group.Get("/:post/replies/:id", getReplies)
	group.Post("/:post/create-comment", authMiddleware, createComment)
	group.Post("/:post/edit-comment/:id", authMiddleware, editComment)
	group.Delete("/:post/comment/:id", authMiddleware, deleteComment)
}

type CommentsResult struct {
	TotalComments int                  `json:"totalComments"`
	Comments      []db.OriginalComment `json:"comments"`
}

func getComments(c fiber.Ctx) error {
	postName := c.Params("post")
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil {
		return err
	}

	if page <= 0 {
		return errors.New("invalid page")
	}

	var result CommentsResult

	comments, err := db.GetComments(c, postName, page)
	if err != nil {
		return err
	}

	result.Comments = comments

	count, err := db.CountComments(c, postName)
	if err != nil {
		return err
	}

	result.TotalComments = count

	c.Set("Cache-Control", "public, max-age=0, s-maxage=15, must-revalidate")

	return c.JSON(result)
}

func getReplies(c fiber.Ctx) error {
	// time.Sleep(5 * time.Second)
	postName := c.Params("post")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	replies, err := db.GetReplies(c, postName, id)
	if err != nil {
		return err
	}

	c.Set("Cache-Control", "public, max-age=0, s-maxage=15, must-revalidate")

	return c.JSON(replies)
}

type CommentOpts struct {
	Content        string `json:"content"`
	ReplyTo        *int   `json:"replyTo"`
	TurnstileToken string `json:"turnstileToken"`
}

func createComment(c fiber.Ctx) error {
	postName := c.Params("post")
	userID := c.Locals("userID").(uuid.UUID)

	var opts CommentOpts
	if err := c.Bind().WithAutoHandling().JSON(&opts); err != nil {
		return err
	}

	if err := validateTurnstileToken(c, opts.TurnstileToken, true); err != nil {
		return err
	}

	if utf8.RuneCountInString(opts.Content) > 10000 {
		return &utils.AppError{
			Status:  fiber.StatusBadRequest,
			Message: "Comments must be 10,000 characters at most",
		}
	}

	result, err := db.CreateComment(c, postName, userID, opts.Content, opts.ReplyTo)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

type EditOpts struct {
	Content string `json:"content"`
}

func editComment(c fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	var opts EditOpts
	if err := c.Bind().WithAutoHandling().JSON(&opts); err != nil {
		return err
	}

	if utf8.RuneCountInString(opts.Content) > 10000 {
		return &utils.AppError{
			Status:  fiber.StatusBadRequest,
			Message: "Comments must be 10,000 characters at most",
		}
	}

	_, err = db.EditComment(c, id, userID, opts.Content)
	if err != nil {
		return err
	}

	return nil
}

func deleteComment(c fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	_, err = db.DeleteComment(c, id, userID)
	if err != nil {
		return err
	}

	return nil
}
