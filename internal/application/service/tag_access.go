package service

import (
	"context"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
)

func (s *knowledgeTagService) requireTagWrite(
	ctx context.Context,
	tag *types.KnowledgeTag,
) (*types.KnowledgeBase, context.Context, error) {
	if tag == nil {
		return nil, ctx, apperrors.NewNotFoundError("Tag not found")
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, tag.KnowledgeBaseID)
	if err != nil {
		return nil, ctx, err
	}
	if kb == nil || kb.ID != tag.KnowledgeBaseID || kb.TenantID != tag.TenantID {
		return nil, ctx, apperrors.NewForbiddenError("Tag does not belong to the current knowledge base")
	}
	ctx, err = requireKBWrite(ctx, kb)
	return kb, ctx, err
}

func (s *knowledgeTagService) validateTagDeleteExclusions(
	ctx context.Context,
	kb *types.KnowledgeBase,
	ids []string,
) error {
	if len(ids) == 0 {
		return nil
	}
	if kb.Type != types.KnowledgeBaseTypeFAQ {
		return apperrors.NewBadRequestError("Excluded entries are only supported when deleting FAQ entries")
	}
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	chunks, err := s.chunkRepo.ListChunksByID(ctx, kb.TenantID, ids)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		if chunk == nil || !wanted[chunk.ID] {
			continue
		}
		if chunk.TenantID != kb.TenantID || chunk.KnowledgeBaseID != kb.ID || chunk.ChunkType != types.ChunkTypeFAQ {
			return apperrors.NewForbiddenError("Excluded entries do not belong to this knowledge base")
		}
		delete(wanted, chunk.ID)
	}
	if len(wanted) != 0 {
		return apperrors.NewNotFoundError("Excluded entries do not exist")
	}
	return nil
}
