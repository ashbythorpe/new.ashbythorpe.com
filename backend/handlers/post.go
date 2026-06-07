package handlers

import (
	"errors"
	"fmt"
	"strconv"

	"ashbythorpe.com/website/config"
	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
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

	if page > 1 {
		c.Set("Cache-Control", "public, s-maxage=30, stale-while-revalidate=120")
	} else {
		c.Set("Cache-Control", "public, s-maxage=2592000")
	}

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

	c.Set("Cache-Control", "public, s-maxage=2592000")

	return c.JSON(replies)
}

type CommentOpts struct {
	Content string `json:"content"`
	ReplyTo *int   `json:"replyTo"`
}

func createComment(c fiber.Ctx) error {
	postName := c.Params("post")
	userID := c.Locals("userID").(uuid.UUID)

	var opts CommentOpts
	if err := c.Bind().WithAutoHandling().JSON(&opts); err != nil {
		return err
	}

	result, err := db.CreateComment(c, postName, userID, opts.Content, opts.ReplyTo)
	if err != nil {
		return err
	}

	go purgeCommentsCache(c, postName, result.OriginalReplyTo)

	return c.JSON(result)
}

type EditOpts struct {
	Content string `json:"content"`
}

func editComment(c fiber.Ctx) error {
	postName := c.Params("post")
	userID := c.Locals("userID").(uuid.UUID)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	var opts EditOpts
	if err := c.Bind().WithAutoHandling().JSON(&opts); err != nil {
		return err
	}

	log.Info(opts.Content)
	originalReplyTo, err := db.EditComment(c, id, userID, opts.Content)
	if err != nil {
		return err
	}

	go purgeCommentsCache(c, postName, originalReplyTo)

	return nil
}

func deleteComment(c fiber.Ctx) error {
	postName := c.Params("post")
	userID := c.Locals("userID").(uuid.UUID)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return err
	}

	originalReplyTo, err := db.DeleteComment(c, id, userID)
	if err != nil {
		return err
	}

	go purgeCommentsCache(c, postName, originalReplyTo)

	return nil
}

func purgeCommentsCache(c fiber.Ctx, post string, replyTo *int) {
	if replyTo != nil {
		utils.PurgeCloudflareCache(c, fmt.Sprintf("%s/api/%s/replies/%d", config.Origin, post, *replyTo))
	} else {
		utils.PurgeCloudflareCache(c, fmt.Sprintf("%s/api/%s/comments?page=1", config.Origin, post))
	}
}
