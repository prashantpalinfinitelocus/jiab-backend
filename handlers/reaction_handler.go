package handlers

import "github.com/gin-gonic/gin"

type ReactionHandler interface {
}

type reactionHandler struct {
}

func NewReactionHandler() ReactionHandler {
	return &reactionHandler{}
}

func (h *reactionHandler) AddUserReelReaction(c *gin.Context) {

}
