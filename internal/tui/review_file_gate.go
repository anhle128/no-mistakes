package tui

import "github.com/kunchenguid/no-mistakes/internal/types"

func (m Model) isReviewFileGate(step types.StepName) bool {
	return step == types.StepReview && m.reviewFilePaths[step] != ""
}

func (m Model) reviewFilePath(step types.StepName) string {
	return m.reviewFilePaths[step]
}

func (m Model) reviewFileAbsPath(step types.StepName) string {
	return m.reviewFileAbsPaths[step]
}

func (m Model) reviewValidationError(step types.StepName) string {
	return m.reviewValidationErrors[step]
}
