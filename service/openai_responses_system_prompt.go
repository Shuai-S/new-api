package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func MergeSystemPromptToResponsesRequest(req *dto.OpenAIResponsesRequest, systemPrompt string, override bool) error {
	if req == nil || strings.TrimSpace(systemPrompt) == "" {
		return nil
	}

	if len(req.Instructions) == 0 {
		instructions, err := common.Marshal(systemPrompt)
		if err != nil {
			return err
		}
		req.Instructions = instructions
		return nil
	}

	if !override {
		return nil
	}

	var existing string
	if err := common.Unmarshal(req.Instructions, &existing); err != nil {
		instructions, marshalErr := common.Marshal(systemPrompt)
		if marshalErr != nil {
			return marshalErr
		}
		req.Instructions = instructions
		return nil
	}

	existing = strings.TrimSpace(existing)
	if existing == "" {
		instructions, err := common.Marshal(systemPrompt)
		if err != nil {
			return err
		}
		req.Instructions = instructions
		return nil
	}

	instructions, err := common.Marshal(systemPrompt + "\n" + existing)
	if err != nil {
		return err
	}
	req.Instructions = instructions
	return nil
}
