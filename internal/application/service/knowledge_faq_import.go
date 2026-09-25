package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/access"
	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/application/service/retriever"
	werrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/tracing/langfuse"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// UpsertFAQEntries imports or appends FAQ entries asynchronously.
// Returns task ID (UUID) for tracking import progress.
func (s *knowledgeService) UpsertFAQEntries(ctx context.Context,
	kbID string, payload *types.FAQBatchUpsertPayload,
) (string, error) {
	if payload == nil || len(payload.Entries) == 0 {
		return "", werrors.NewBadRequestError("FAQ entries cannot be empty")
	}
	if payload.Mode == "" {
		payload.Mode = types.FAQBatchModeAppend
	}
	if payload.Mode != types.FAQBatchModeAppend && payload.Mode != types.FAQBatchModeReplace {
		return "", werrors.NewBadRequestError("Mode only supports append or replace")
	}

	// Verify that the knowledge base exists and is valid
	kb, ctx, err := s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return "", err
	}
	if err := s.validateFAQImportTags(ctx, kb, payload.Entries); err != nil {
		return "", err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	// Use the passed-in TaskID, or generate an enhanced TaskID if none was passed.
	// A client-supplied task_id ends up in file names and Redis keys, so it must be an identifier without path separators.
	taskID := strings.TrimSpace(payload.TaskID)
	if taskID == "" {
		taskID = secutils.GenerateTaskID("faq_import", tenantID, kbID)
	} else if err := secutils.ValidateTaskID(taskID); err != nil {
		return "", werrors.NewBadRequestError("task_id 格式不合法")
	}

	var knowledgeID string

	// Check whether an import task is already in progress (via Redis)
	runningTaskID, err := s.getRunningFAQImportTaskID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to check running import task: %v", err)
		// Check failure doesn't affect the import; continue execution
	} else if runningTaskID != "" {
		logger.Warnf(ctx, "Import task already running for KB %s: %s", kbID, runningTaskID)
		return "", werrors.NewBadRequestError(fmt.Sprintf("This knowledge base already has an import task in progress (task ID: %s); please wait for it to finish", runningTaskID))
	}

	// Ensure the FAQ knowledge exists
	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return "", fmt.Errorf("failed to ensure FAQ knowledge: %w", err)
	}
	knowledgeID = faqKnowledge.ID

	// Record the task enqueue time
	enqueuedAt := time.Now().Unix()
	instanceID := uuid.NewString()
	runningInfoSet := false
	enqueueSucceeded := false

	// Set the KB's running-task info
	if err := s.setRunningFAQImportInfo(ctx, kbID, &runningFAQImportInfo{
		TaskID:     taskID,
		EnqueuedAt: enqueuedAt,
		InstanceID: instanceID,
	}); err != nil {
		logger.Errorf(ctx, "Failed to set running FAQ import task info: %v", err)
		// Doesn't affect task execution; continue
	} else {
		runningInfoSet = true
	}
	defer func() {
		if !runningInfoSet || enqueueSucceeded {
			return
		}
		if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, kbID, taskID, instanceID, enqueuedAt); clearErr != nil {
			logger.Warnf(ctx, "Failed to clear FAQ import running info after setup failure: %v", clearErr)
		}
	}()

	// Initialize the import task status in Redis
	progress := &types.FAQImportProgress{
		TaskID:        taskID,
		KBID:          kbID,
		KnowledgeID:   knowledgeID,
		Status:        types.FAQImportStatusPending,
		Progress:      0,
		Total:         len(payload.Entries),
		Processed:     0,
		SuccessCount:  0,
		FailedCount:   0,
		FailedEntries: make([]types.FAQFailedEntry, 0),
		Message:       "Task created, waiting to process",
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
		DryRun:        payload.DryRun,
	}
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Errorf(ctx, "Failed to initialize FAQ import task status: %v", err)
		return "", fmt.Errorf("failed to initialize task: %w", err)
	}

	logger.Infof(ctx, "FAQ import task initialized: %s, kb_id: %s, total entries: %d, dry_run: %v",
		taskID, kbID, len(payload.Entries), payload.DryRun)

	// Enqueue FAQ import task to Asynq
	logger.Info(ctx, "Enqueuing FAQ import task to Asynq")

	// Build the task payload
	taskPayload := types.FAQImportPayload{
		TenantID:    tenantID,
		TaskID:      taskID,
		KBID:        kbID,
		KnowledgeID: knowledgeID,
		Mode:        payload.Mode,
		DryRun:      payload.DryRun,
		EnqueuedAt:  enqueuedAt,
		InstanceID:  instanceID,
		Initiator:   types.TaskInitiatorFromContext(ctx),
	}

	// Threshold: use object storage when there are more than 200 entries or the serialized size exceeds 50KB
	const (
		entryCountThreshold  = 200
		payloadSizeThreshold = 50 * 1024 // 50KB
	)

	entryCount := len(payload.Entries)
	if entryCount > entryCountThreshold {
		// Data volume is large; upload to object storage
		entriesData, err := json.Marshal(payload.Entries)
		if err != nil {
			logger.Errorf(ctx, "Failed to marshal FAQ entries: %v", err)
			return "", fmt.Errorf("failed to marshal entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries size: %d bytes, uploading to object storage", len(entriesData))

		// Upload to the private bucket (primary bucket); clean up after task processing completes
		fileName, err := faqImportEntriesFileName(taskID, enqueuedAt)
		if err != nil {
			return "", fmt.Errorf("invalid task id for object name: %w", err)
		}
		entriesURL, err := s.fileSvc.SaveBytes(ctx, entriesData, tenantID, fileName, false)
		if err != nil {
			logger.Errorf(ctx, "Failed to upload FAQ entries to object storage: %v", err)
			return "", fmt.Errorf("failed to upload entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries uploaded to: %s", entriesURL)
		taskPayload.EntriesURL = entriesURL
		taskPayload.EntryCount = entryCount
	} else {
		// Data volume is small; store directly in the payload
		taskPayload.Entries = payload.Entries
	}

	langfuse.InjectTracing(ctx, &taskPayload)
	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal FAQ import task payload: %v", err)
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	// Check the payload size again
	if len(payloadBytes) > payloadSizeThreshold && taskPayload.EntriesURL == "" {
		// Payload is too large but not yet uploaded; upload it now
		entriesData, _ := json.Marshal(payload.Entries)
		fileName, nameErr := faqImportEntriesFileName(taskID, enqueuedAt)
		if nameErr != nil {
			return "", fmt.Errorf("invalid task id for object name: %w", nameErr)
		}
		entriesURL, err := s.fileSvc.SaveBytes(ctx, entriesData, tenantID, fileName, false)
		if err != nil {
			logger.Errorf(ctx, "Failed to upload FAQ entries to object storage: %v", err)
			return "", fmt.Errorf("failed to upload entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries uploaded to (size exceeded): %s", entriesURL)
		taskPayload.Entries = nil
		taskPayload.EntriesURL = entriesURL
		taskPayload.EntryCount = entryCount

		payloadBytes, _ = json.Marshal(taskPayload)
	}

	logger.Infof(ctx, "FAQ import task payload size: %d bytes", len(payloadBytes))

	maxRetry := 5
	if payload.DryRun {
		maxRetry = 3 // Fewer retries for dry runs
	}

	// Use taskID:instanceID as asynq's unique task identifier
	asynqTaskID := fmt.Sprintf("%s:%s", taskID, instanceID)

	task := asynq.NewTask(
		types.TypeFAQImport,
		payloadBytes,
		asynq.TaskID(asynqTaskID),
		asynq.Queue(types.QueueMaintenance),
		asynq.MaxRetry(maxRetry),
		asynq.Timeout(2*time.Hour),
	)
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue FAQ import task: %v", err)
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}
	logger.Infof(ctx, "Enqueued FAQ import task: id=%s queue=%s task_id=%s dry_run=%v", info.ID, info.Queue, taskID, payload.DryRun)
	enqueueSucceeded = true

	if !payload.DryRun {
		recordKBActivity(ctx, s.audit, tenantID, kbID, types.AuditActionFAQImportStarted,
			"faq_entry", knowledgeID, types.AuditOutcomeAccepted,
			map[string]any{
				"task_id": taskID, "mode": payload.Mode, "total": len(payload.Entries),
				"trigger": kbActivityTrigger(ctx), "processing_status": "pending",
			})
	}

	return taskID, nil
}

func faqImportEntriesFileName(taskID string, enqueuedAt int64) (string, error) {
	return secutils.SafeFileName(fmt.Sprintf("faq_import_entries_%s_%d.json", taskID, enqueuedAt))
}

// generateFailedEntriesCSV generates a CSV file of failed entries and uploads it
func (s *knowledgeService) generateFailedEntriesCSV(ctx context.Context,
	tenantID uint64, taskID string, failedEntries []types.FAQFailedEntry,
) (string, error) {
	// Generate the CSV content
	var buf strings.Builder

	// Write the BOM to help Excel correctly detect UTF-8
	buf.WriteString("\xEF\xBB\xBF")

	// Write the header row
	buf.WriteString("Error Reason,Category(required),Question(required),Similar Questions(optional-separate multiple with ##),Negative Examples(optional-separate multiple with ##),Bot Answers(required-separate multiple with ##),Reply All(optional-default FALSE),Disabled(optional-default FALSE)\n")

	// Write the data rows
	for _, entry := range failedEntries {
		// CSV escaping: if the content contains commas, quotes, or newlines, wrap it in quotes and escape internal quotes
		reason := csvEscape(entry.Reason)
		tagName := csvEscape(entry.TagName)
		standardQ := csvEscape(entry.StandardQuestion)
		similarQs := ""
		if len(entry.SimilarQuestions) > 0 {
			similarQs = csvEscape(strings.Join(entry.SimilarQuestions, "##"))
		}
		negativeQs := ""
		if len(entry.NegativeQuestions) > 0 {
			negativeQs = csvEscape(strings.Join(entry.NegativeQuestions, "##"))
		}
		answers := ""
		if len(entry.Answers) > 0 {
			answers = csvEscape(strings.Join(entry.Answers, "##"))
		}
		answerAll := "false"
		if entry.AnswerAll {
			answerAll = "true"
		}
		isDisabled := "false"
		if entry.IsDisabled {
			isDisabled = "true"
		}

		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
			reason, tagName, standardQ, similarQs, negativeQs, answers, answerAll, isDisabled))
	}

	// Upload the CSV file to temporary storage (auto-expires)
	fileName, err := secutils.SafeFileName(fmt.Sprintf("faq_dryrun_failed_%s.csv", taskID))
	if err != nil {
		return "", fmt.Errorf("invalid task id for object name: %w", err)
	}
	filePath, err := s.fileSvc.SaveBytes(ctx, []byte(buf.String()), tenantID, fileName, true)
	if err != nil {
		return "", fmt.Errorf("failed to save CSV file: %w", err)
	}

	// Get the download URL
	fileURL, err := s.fileSvc.GetFileURL(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file URL: %w", err)
	}

	logger.Infof(ctx, "Generated failed entries CSV: %s, entries: %d", fileURL, len(failedEntries))
	return fileURL, nil
}

// csvEscape escapes a CSV field
func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		// Replace internal quotes with two quotes and wrap the whole field in quotes
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// saveFAQImportResultToDatabase saves the FAQ import result statistics to the database
func (s *knowledgeService) saveFAQImportResultToDatabase(ctx context.Context,
	payload *types.FAQImportPayload, progress *types.FAQImportProgress, originalTotalEntries int,
) error {
	// Get the FAQ knowledge base instance
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, payload.KnowledgeID)
	if err != nil {
		return fmt.Errorf("failed to get FAQ knowledge: %w", err)
	}

	// Compute the skipped count (total - fully succeeded - partially failed - fully failed)
	skippedCount := originalTotalEntries - progress.SuccessCount - progress.PartialFailedCount - progress.FailedCount
	if skippedCount < 0 {
		skippedCount = 0
	}

	// Create the import result statistics
	importResult := &types.FAQImportResult{
		TotalEntries:       originalTotalEntries,
		SuccessCount:       progress.SuccessCount,
		FailedCount:        progress.FailedCount,
		PartialFailedCount: progress.PartialFailedCount,
		SkippedCount:       skippedCount,
		MergedCount:        progress.MergedCount,
		AddedCount:         progress.AddedCount,
		ImportMode:         payload.Mode,
		ImportedAt:         time.Now(),
		TaskID:             payload.TaskID,
		ProcessingTime:     time.Now().Unix() - progress.CreatedAt, // Processing duration (seconds)
		DisplayStatus:      "open",                                 // Newly imported results are shown by default
	}

	// If there's a failed/partially-failed entries CSV download URL, write it to the result (FailedEntries includes both fully failed and partially failed)
	if progress.FailedEntriesURL != "" {
		importResult.FailedEntriesURL = progress.FailedEntriesURL
	}

	// Set the import result into Knowledge's metadata
	if err := knowledge.SetLastFAQImportResult(importResult); err != nil {
		return fmt.Errorf("failed to set FAQ import result: %w", err)
	}

	// Update the database
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		return fmt.Errorf("failed to update knowledge with import result: %w", err)
	}

	logger.Infof(ctx, "Saved FAQ import result to database: knowledge_id=%s, task_id=%s, total=%d, success=%d, added=%d, merged=%d, failed=%d, partial_failed=%d, skipped=%d",
		payload.KnowledgeID, payload.TaskID, originalTotalEntries, progress.SuccessCount, progress.AddedCount, progress.MergedCount, progress.FailedCount, progress.PartialFailedCount, skippedCount)

	return nil
}

// buildFAQFailedEntry builds a FAQFailedEntry
func buildFAQFailedEntry(idx int, reason string, entry *types.FAQEntryPayload) types.FAQFailedEntry {
	answerAll := false
	if entry.AnswerStrategy != nil && *entry.AnswerStrategy == types.AnswerStrategyAll {
		answerAll = true
	}
	isDisabled := false
	if entry.IsEnabled != nil && !*entry.IsEnabled {
		isDisabled = true
	}
	return types.FAQFailedEntry{
		Index:             idx,
		Reason:            reason,
		TagName:           entry.TagName,
		StandardQuestion:  strings.TrimSpace(entry.StandardQuestion),
		SimilarQuestions:  entry.SimilarQuestions,
		NegativeQuestions: entry.NegativeQuestions,
		Answers:           entry.Answers,
		AnswerAll:         answerAll,
		IsDisabled:        isDisabled,
	}
}

func buildFAQPartialFailedEntry(idx int, entry *types.FAQEntryPayload,
	removedSimilarQuestions, removedNegativeQuestions []string,
) types.FAQFailedEntry {
	answerAll := false
	if entry.AnswerStrategy != nil && *entry.AnswerStrategy == types.AnswerStrategyAll {
		answerAll = true
	}
	isDisabled := false
	if entry.IsEnabled != nil && !*entry.IsEnabled {
		isDisabled = true
	}

	// Build the failure reason description: summary + details
	var summary []string
	if len(removedSimilarQuestions) > 0 {
		summary = append(summary, fmt.Sprintf("%d similar questions removed", len(removedSimilarQuestions)))
	}
	if len(removedNegativeQuestions) > 0 {
		summary = append(summary, fmt.Sprintf("%d negative examples removed", len(removedNegativeQuestions)))
	}

	// Full reason: summary | similar-question details | counter-example details
	var reasonParts []string
	reasonParts = append(reasonParts, "Partially succeeded: "+strings.Join(summary, "，"))
	if len(removedSimilarQuestions) > 0 {
		reasonParts = append(reasonParts, strings.Join(removedSimilarQuestions, "; "))
	}
	if len(removedNegativeQuestions) > 0 {
		reasonParts = append(reasonParts, strings.Join(removedNegativeQuestions, "; "))
	}

	return types.FAQFailedEntry{
		Index:                    idx,
		Reason:                   strings.Join(reasonParts, " | "),
		IsPartialFailure:         true,
		TagName:                  entry.TagName,
		StandardQuestion:         strings.TrimSpace(entry.StandardQuestion),
		SimilarQuestions:         entry.SimilarQuestions,
		NegativeQuestions:        entry.NegativeQuestions,
		Answers:                  entry.Answers,
		AnswerAll:                answerAll,
		IsDisabled:               isDisabled,
		RemovedSimilarQuestions:  removedSimilarQuestions,
		RemovedNegativeQuestions: removedNegativeQuestions,
	}
}

// executeFAQDryRunValidation runs FAQ dry-run validation, returning indices of entries that passed validation
func (s *knowledgeService) executeFAQDryRunValidation(ctx context.Context,
	payload *types.FAQImportPayload, progress *types.FAQImportProgress,
) []int {
	entries := payload.Entries

	// Used to record indices of entries that passed basic validation and duplicate checks, for subsequent security checks
	validEntryIndices := make([]int, 0, len(entries))

	// Choose different validation logic based on the mode
	if payload.Mode == types.FAQBatchModeAppend {
		validEntryIndices = s.validateEntriesForAppendModeWithProgress(ctx, payload.TenantID, payload.KBID, entries, progress)
	} else {
		validEntryIndices = s.validateEntriesForReplaceModeWithProgress(ctx, entries, progress)
	}

	return validEntryIndices
}

// validateEntriesForAppendModeWithProgress validates entries in Append mode (with progress updates)
// Note: the validation phase does not update Processed; only the actual import does
// validateEntriesForAppendModeWithProgress validates entries in Append mode (with progress updates)
// Note: the validation phase does not update Processed; only the actual import does
//
// Validation is split into four phases:
// Phase one (pre-validation):
// 1. Standard questions - dedup within the file only; if a matching standard question already exists in the KB, mark as a merge candidate
// 2. Similar questions - compare file + knowledge base (merge candidates exclude their own existing questions) → remove conflicting similar questions individually
// 3. Counter-examples - compare only against the current QA's own standard question + similar questions → remove conflicting counter-examples individually
//
// Phase two (post-validation, merge candidates only):
// 4. Rerun counter-example validation on the fully merged data → on conflict, roll back the entire entry to its pre-merge state
func (s *knowledgeService) validateEntriesForAppendModeWithProgress(ctx context.Context,
	tenantID uint64, kbID string, entries []types.FAQEntryPayload, progress *types.FAQImportProgress,
) []int {
	totalEntries := len(entries)

	// Query the metadata of all existing FAQ chunks in the knowledge base
	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		logger.Warnf(ctx, "Failed to list existing FAQ chunks for dry run: %v", err)
	}

	// Build a mapping of existing standard questions → chunk (used for merge candidate identification)
	existingStdQToChunk := make(map[string]*types.Chunk)
	// Build a mapping of all existing questions → owning chunkID (used for similar-question conflict detection)
	existingQuestionToChunkID := make(map[string]string)
	// Build the set of questions owned by each chunk (used to exclude its own questions during merging)
	existingChunkQuestions := make(map[string]map[string]bool)
	// Build a mapping of chunkID → standard question (used to display conflict failure reasons)
	existingChunkIDToStdQ := make(map[string]string)

	for _, chunk := range existingChunks {
		meta, err := chunk.FAQMetadata()
		if err != nil || meta == nil {
			continue
		}
		qs := make(map[string]bool)
		if meta.StandardQuestion != "" {
			existingStdQToChunk[meta.StandardQuestion] = chunk
			existingQuestionToChunkID[meta.StandardQuestion] = chunk.ID
			qs[meta.StandardQuestion] = true
		}
		for _, q := range meta.SimilarQuestions {
			if q != "" {
				existingQuestionToChunkID[q] = chunk.ID
				qs[q] = true
			}
		}
		existingChunkQuestions[chunk.ID] = qs
		existingChunkIDToStdQ[chunk.ID] = meta.StandardQuestion
	}

	// Merge candidate tracking: entry index → target existing chunk
	mergeChunkMap := make(map[int]*types.Chunk)

	// ==================== First iteration: basic format validation + intra-batch standard question deduplication + merge candidate identification ====================
	batchStandardQuestions := make(map[string]int) // value is the index of first occurrence
	validIndicesAfterStdQ := make([]int, 0, totalEntries)

	for i, entry := range entries {
		if err := validateFAQEntryPayloadBasic(&entry); err != nil {
			progress.FailedCount++
			fe := buildFAQFailedEntry(i, err.Error(), &entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			continue
		}

		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Intra-batch standard question deduplication
		if firstIdx, exists := batchStandardQuestions[standardQ]; exists {
			progress.FailedCount++
			fe := buildFAQFailedEntry(i, fmt.Sprintf("Standard question conflict: duplicates standard question at row %d in the batch", firstIdx+1), &entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			continue
		}

		// Determine whether it's a merge candidate
		if chunk, exists := existingStdQToChunk[standardQ]; exists {
			// Standard question already exists in KB → mark as merge candidate
			mergeChunkMap[i] = chunk
			logger.Infof(ctx, "FAQ entry %d: standard question '%s' exists in KB, marking as merge candidate (chunk_id=%s)", i, standardQ, chunk.ID)
		} else if conflictChunkID, hit := existingQuestionToChunkID[standardQ]; hit {
			// Standard question doesn't duplicate any existing standard question, but collides with a similar question of another entry in the KB
			// → Pre-validation failed, avoiding statistical distortion caused by downstream calculateAppendOperations silently dropping entries
			conflictStdQ := existingChunkIDToStdQ[conflictChunkID]
			progress.FailedCount++
			fe := buildFAQFailedEntry(i,
				fmt.Sprintf(`Standard question conflict: conflicts with similar question "%s" of existing standard question "%s"`, standardQ, conflictStdQ),
				&entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			logger.Infof(ctx,
				"FAQ entry %d: standard question '%s' conflicts with existing similar question of chunk_id=%s (std='%s')",
				i, standardQ, conflictChunkID, conflictStdQ)
			continue
		}

		batchStandardQuestions[standardQ] = i
		validIndicesAfterStdQ = append(validIndicesAfterStdQ, i)

		if (i+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating standard questions %d/%d...", i+1, totalEntries)
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Second iteration: similar question conflict detection ====================
	// Build the set of all standard questions + similar questions within the batch
	batchAllQuestions := make(map[string]int)
	for _, i := range validIndicesAfterStdQ {
		standardQ := strings.TrimSpace(entries[i].StandardQuestion)
		batchAllQuestions[standardQ] = i
		for _, q := range entries[i].SimilarQuestions {
			q = strings.TrimSpace(q)
			if q != "" {
				if _, exists := batchAllQuestions[q]; !exists {
					batchAllQuestions[q] = i
				}
			}
		}
	}

	removedSimilarQuestionsMap := make(map[int][]string)
	removedNegativeQuestionsMap := make(map[int][]string)

	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Merge candidate: get the target chunk's own question set, used for exclusion
		var ownChunkQuestions map[string]bool
		if mergeChunk, isMerge := mergeChunkMap[i]; isMerge {
			ownChunkQuestions = existingChunkQuestions[mergeChunk.ID]
		}

		validSimilarQuestions := make([]string, 0, len(entry.SimilarQuestions))
		var removedSimilarQuestions []string
		for _, q := range entry.SimilarQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// A similar question cannot be identical to its own standard question
			if q == standardQ {
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`Similar-question conflict: "%s" conflicts with this entry's standard question`, q))
				continue
			}
			// Similar question validation: compare against the KB's existing standard questions + similar questions
			if _, exists := existingQuestionToChunkID[q]; exists {
				// Merge candidate: if the question belongs to the merge target's own chunk, allow it (handled during dedup merge)
				if ownChunkQuestions != nil && ownChunkQuestions[q] {
					validSimilarQuestions = append(validSimilarQuestions, q)
					continue
				}
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`Similar-question conflict: "%s" conflicts with an existing knowledge-base standard/similar question`, q))
				continue
			}
			// Similar question validation: compare against the batch's standard questions + similar questions
			if firstIdx, exists := batchAllQuestions[q]; exists && firstIdx != i {
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`Similar-question conflict: "%s" conflicts with the standard/similar question at row %d`, q, firstIdx+1))
				continue
			}
			validSimilarQuestions = append(validSimilarQuestions, q)
		}
		entries[i].SimilarQuestions = validSimilarQuestions

		if len(removedSimilarQuestions) > 0 {
			removedSimilarQuestionsMap[i] = removedSimilarQuestions
		}

		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating similar questions %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Third iteration: counter-example conflict detection (pre-validation, checks only the new entry's own data) ====================
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		currentQAQuestions := make(map[string]bool)
		currentQAQuestions[standardQ] = true
		for _, q := range entry.SimilarQuestions {
			currentQAQuestions[q] = true
		}

		validNegativeQuestions := make([]string, 0, len(entry.NegativeQuestions))
		var removedNegativeQuestions []string
		for _, q := range entry.NegativeQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			if currentQAQuestions[q] {
				removedNegativeQuestions = append(removedNegativeQuestions, fmt.Sprintf(`Negative-example conflict: "%s" conflicts with this entry's standard/similar question`, q))
				continue
			}
			validNegativeQuestions = append(validNegativeQuestions, q)
		}
		entries[i].NegativeQuestions = validNegativeQuestions

		if len(removedNegativeQuestions) > 0 {
			removedNegativeQuestionsMap[i] = removedNegativeQuestions
		}

		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating negative examples %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Fourth iteration: post-validation (merge candidates only) ====================
	// Rerun counter-example validation on the full merged data; on conflict, roll back the whole entry to its pre-merge state
	postValidationFailed := make(map[int]bool)
	mergeCount := 0
	for _, i := range validIndicesAfterStdQ {
		mergeChunk, isMerge := mergeChunkMap[i]
		if !isMerge {
			continue
		}

		existingMeta, err := mergeChunk.FAQMetadata()
		if err != nil || existingMeta == nil {
			logger.Warnf(ctx, "FAQ entry %d: failed to get merge target metadata, skipping post-validation", i)
			continue
		}

		entry := &entries[i]

		// Compute the full merged data
		mergedSimilar := unionStrings(existingMeta.SimilarQuestions, entry.SimilarQuestions)
		mergedNegative := unionStrings(existingMeta.NegativeQuestions, entry.NegativeQuestions)

		// Build the merged conflict set (standard question + all merged similar questions)
		mergedPositiveSet := make(map[string]bool)
		mergedPositiveSet[existingMeta.StandardQuestion] = true
		for _, q := range mergedSimilar {
			mergedPositiveSet[q] = true
		}

		// Check whether each merged counter-example conflicts with the merged standard question / similar questions
		var conflictingNegatives []string
		for _, q := range mergedNegative {
			if mergedPositiveSet[q] {
				conflictingNegatives = append(conflictingNegatives, q)
			}
		}

		if len(conflictingNegatives) > 0 {
			// Post-validation failed → roll back the whole entry to its pre-merge state
			postValidationFailed[i] = true
			delete(mergeChunkMap, i)
			progress.FailedCount++
			reason := fmt.Sprintf("Post-validation failed: merged negative example 「%s」 conflicts with a similar question", strings.Join(conflictingNegatives, "、"))
			fe := buildFAQFailedEntry(i, reason, entry)
			fe.FailureType = "post_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			logger.Infof(ctx, "FAQ entry %d: post-validation failed, conflicting negatives after merge: %v", i, conflictingNegatives)
		} else {
			mergeCount++
		}
	}

	// Remove entries that failed post-validation from the valid indices
	if len(postValidationFailed) > 0 {
		filtered := make([]int, 0, len(validIndicesAfterStdQ))
		for _, i := range validIndicesAfterStdQ {
			if !postValidationFailed[i] {
				filtered = append(filtered, i)
			}
		}
		validIndicesAfterStdQ = filtered
	}

	// Add partial failure info to FailedEntries
	for _, i := range validIndicesAfterStdQ {
		removedSimilar := removedSimilarQuestionsMap[i]
		removedNegative := removedNegativeQuestionsMap[i]
		if len(removedSimilar) > 0 || len(removedNegative) > 0 {
			pf := buildFAQPartialFailedEntry(i, &entries[i], removedSimilar, removedNegative)
			progress.FailedEntries = append(progress.FailedEntries, pf)
			progress.PartialFailedCount++
		}
	}

	// Record merged entry index in progress (for the execution phase and retries)
	mergeIndices := make([]int, 0, mergeCount)
	for _, i := range validIndicesAfterStdQ {
		if _, isMerge := mergeChunkMap[i]; isMerge {
			mergeIndices = append(mergeIndices, i)
		}
	}
	progress.MergeEntryIndices = mergeIndices

	logger.Infof(ctx, "Append mode validation completed: total=%d, valid=%d, merge_candidates=%d, failed=%d, partial_failed=%d",
		totalEntries, len(validIndicesAfterStdQ), mergeCount, progress.FailedCount, progress.PartialFailedCount)

	return validIndicesAfterStdQ
}

// validateEntriesForReplaceModeWithProgress validates entries in Replace mode (with progress updates)
// Note: Processed is not updated during the validation phase, only during actual import
// Validate in three iterations, ensuring data filtered out earlier is not considered later:
// 1. Standard question - compare against all standard questions → whole QA entry fails "standard question conflict"
// 2. Similar question - compare against all standard questions + similar questions → single question phrasing fails "similar question conflict" (only the conflicting similar question is removed)
// 3. Counter-example - compare against all standard questions + similar questions under the current QA → single question phrasing fails "counter-example conflict" (only the conflicting counter-example is removed)
func (s *knowledgeService) validateEntriesForReplaceModeWithProgress(ctx context.Context,
	entries []types.FAQEntryPayload, progress *types.FAQImportProgress,
) []int {
	totalEntries := len(entries)

	// ==================== First iteration: basic format validation + standard question conflict detection ====================
	// Standard question conflict causes the whole QA entry to fail
	batchStandardQuestions := make(map[string]int) // value is the index of first occurrence
	validIndicesAfterStdQ := make([]int, 0, totalEntries)

	for i, entry := range entries {
		// Validate entry's basic format
		if err := validateFAQEntryPayloadBasic(&entry); err != nil {
			progress.FailedCount++
			progress.FailedEntries = append(progress.FailedEntries, buildFAQFailedEntry(i, err.Error(), &entry))
			continue
		}

		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Standard question validation: compare against all standard questions → whole QA entry fails "standard question conflict"
		if firstIdx, exists := batchStandardQuestions[standardQ]; exists {
			progress.FailedCount++
			progress.FailedEntries = append(progress.FailedEntries, buildFAQFailedEntry(i, fmt.Sprintf("Standard question conflict: duplicates standard question at row %d in the batch", firstIdx+1), &entry))
			continue
		}

		// Record standard question
		batchStandardQuestions[standardQ] = i
		validIndicesAfterStdQ = append(validIndicesAfterStdQ, i)

		// Periodically update progress message
		if (i+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating standard questions %d/%d...", i+1, totalEntries)
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Second iteration: similar question conflict detection ====================
	// Only process entries that passed the first validation; similar question conflicts only remove the conflicting similar question
	// Build the set of all standard questions + similar questions (only including entries that passed the first validation)
	batchAllQuestions := make(map[string]int) // value is the index of first occurrence
	for _, i := range validIndicesAfterStdQ {
		standardQ := strings.TrimSpace(entries[i].StandardQuestion)
		batchAllQuestions[standardQ] = i
		for _, q := range entries[i].SimilarQuestions {
			q = strings.TrimSpace(q)
			if q != "" {
				// Only record the position of the first occurrence
				if _, exists := batchAllQuestions[q]; !exists {
					batchAllQuestions[q] = i
				}
			}
		}
	}

	// Used to collect the similar questions and negative examples removed for each entry
	removedSimilarQuestionsMap := make(map[int][]string)  // key is the entry index
	removedNegativeQuestionsMap := make(map[int][]string) // key is the entry index

	// For each entry that passed the first validation, filter out conflicting similar questions
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		validSimilarQuestions := make([]string, 0, len(entry.SimilarQuestions))
		var removedSimilarQuestions []string
		for _, q := range entry.SimilarQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// Similar question validation: compare against all standard questions + similar questions → a single phrasing failure is a "similar question conflict"
			// If this similar question conflicts with another entry's standard question or similar question (and is not its own standard question), remove it
			if firstIdx, exists := batchAllQuestions[q]; exists && firstIdx != i {
				logger.Infof(ctx, "FAQ entry %d: similar question '%s' conflicts with entry %d, removing", i, q, firstIdx+1)
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`Similar-question conflict: "%s" conflicts with the standard/similar question at row %d`, q, firstIdx+1))
				continue
			}
			// A similar question cannot be identical to its own standard question
			if q == standardQ {
				logger.Infof(ctx, "FAQ entry %d: similar question '%s' same as standard question, removing", i, q)
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`Similar-question conflict: "%s" conflicts with this entry's standard question`, q))
				continue
			}
			validSimilarQuestions = append(validSimilarQuestions, q)
		}
		entries[i].SimilarQuestions = validSimilarQuestions

		// Record the removed similar questions
		if len(removedSimilarQuestions) > 0 {
			removedSimilarQuestionsMap[i] = removedSimilarQuestions
		}

		// Periodically update the progress message
		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating similar questions %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Third iteration: negative example conflict detection ====================
	// Only process entries that passed the first two validations; a negative example conflict only removes the conflicting negative example
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Build the set of all questions for the current QA (standard question + validated similar questions)
		currentQAQuestions := make(map[string]bool)
		currentQAQuestions[standardQ] = true
		for _, q := range entry.SimilarQuestions {
			currentQAQuestions[q] = true
		}

		validNegativeQuestions := make([]string, 0, len(entry.NegativeQuestions))
		var removedNegativeQuestions []string
		for _, q := range entry.NegativeQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// Negative example validation: compare against all standard questions + similar questions under the current QA → a single phrasing failure is a "negative example conflict"
			if currentQAQuestions[q] {
				logger.Infof(ctx, "FAQ entry %d: negative question '%s' conflicts with current QA's questions, removing", i, q)
				removedNegativeQuestions = append(removedNegativeQuestions, fmt.Sprintf(`Negative-example conflict: "%s" conflicts with this entry's standard/similar question`, q))
				continue
			}
			validNegativeQuestions = append(validNegativeQuestions, q)
		}
		entries[i].NegativeQuestions = validNegativeQuestions

		// Record the removed negative examples
		if len(removedNegativeQuestions) > 0 {
			removedNegativeQuestionsMap[i] = removedNegativeQuestions
		}

		// Periodically update the progress message
		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating negative examples %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// Add partial failure information to FailedEntries
	for _, i := range validIndicesAfterStdQ {
		removedSimilar := removedSimilarQuestionsMap[i]
		removedNegative := removedNegativeQuestionsMap[i]
		if len(removedSimilar) > 0 || len(removedNegative) > 0 {
			pf := buildFAQPartialFailedEntry(i, &entries[i], removedSimilar, removedNegative)
			progress.FailedEntries = append(progress.FailedEntries, pf)
			progress.PartialFailedCount++
		}
	}

	return validIndicesAfterStdQ
}

// unionStrings merges two string slices and deduplicates (exact match)
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// validateFAQEntryPayloadBasic validates the basic format of an FAQ entry
func validateFAQEntryPayloadBasic(entry *types.FAQEntryPayload) error {
	if entry == nil {
		return fmt.Errorf("Entries cannot be empty")
	}
	standardQ := strings.TrimSpace(entry.StandardQuestion)
	if standardQ == "" {
		return fmt.Errorf("Standard question cannot be empty")
	}
	if len(entry.Answers) == 0 {
		return fmt.Errorf("Answer cannot be empty")
	}
	hasValidAnswer := false
	for _, a := range entry.Answers {
		if strings.TrimSpace(a) != "" {
			hasValidAnswer = true
			break
		}
	}
	if !hasValidAnswer {
		return fmt.Errorf("Answers cannot all be empty")
	}
	return nil
}

type faqMergeOperation struct {
	Entry         types.FAQEntryPayload
	ExistingChunk *types.Chunk
	OldMeta       *types.FAQChunkMetadata
	MergedMeta    *types.FAQChunkMetadata
	Detail        types.FAQMergeDetail
}

// calculateAppendOperations calculates the operations for Append mode (supports intelligent merging).
// If the entry's standard question already exists in the KB, it is treated as a merge operation (union of similar questions / negative examples,
// answer / strategy follows the new entry); otherwise it is treated as a new creation. Merge targets with no changes (hash matches and the operation bit
// also unchanged) are skipped to avoid useless writes to the DB / index rebuilds.
//
// Internal master behavior, corresponding to the open-source version's simplified logic of just "duplicate = failure". FAQ import's
// common usage pattern is "export, modify, then append again", which requires this merge semantics to correctly layer on
// new similar questions without losing historical data.
func (s *knowledgeService) calculateAppendOperations(ctx context.Context,
	tenantID uint64, kbID string, entries []types.FAQEntryPayload,
) (newEntries []types.FAQEntryPayload, mergeOps []faqMergeOperation, skippedCount int, err error) {
	if len(entries) == 0 {
		return nil, nil, 0, nil
	}

	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to list existing FAQ chunks: %w", err)
	}

	existingStdQToChunk := make(map[string]*types.Chunk)
	existingQuestions := make(map[string]bool)
	for _, chunk := range existingChunks {
		meta, cErr := chunk.FAQMetadata()
		if cErr != nil || meta == nil {
			continue
		}
		if meta.StandardQuestion != "" {
			existingStdQToChunk[meta.StandardQuestion] = chunk
			existingQuestions[meta.StandardQuestion] = true
		}
		for _, q := range meta.SimilarQuestions {
			if q != "" {
				existingQuestions[q] = true
			}
		}
	}

	batchQuestions := make(map[string]bool)
	newEntries = make([]types.FAQEntryPayload, 0, len(entries))
	mergeOps = make([]faqMergeOperation, 0)

	for entryIdx, entry := range entries {
		meta, sErr := sanitizeFAQEntryPayload(&entry)
		if sErr != nil {
			skippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry: %v", sErr)
			continue
		}

		// Check whether the standard question exists in the KB as a standard question → merge candidate
		existingChunk, isMergeCandidate := existingStdQToChunk[meta.StandardQuestion]
		if isMergeCandidate {
			existingMeta, mErr := existingChunk.FAQMetadata()
			if mErr != nil || existingMeta == nil {
				isMergeCandidate = false
			} else {
				mergedSimilar := unionStrings(existingMeta.SimilarQuestions, meta.SimilarQuestions)
				mergedNegative := unionStrings(existingMeta.NegativeQuestions, meta.NegativeQuestions)

				mergedMeta := &types.FAQChunkMetadata{
					StandardQuestion:  existingMeta.StandardQuestion,
					SimilarQuestions:  mergedSimilar,
					NegativeQuestions: mergedNegative,
					Answers:           meta.Answers,
					AnswerStrategy:    meta.AnswerStrategy,
					Version:           existingMeta.Version + 1,
					Source:            existingMeta.Source,
				}

				newHash := types.CalculateFAQContentHash(mergedMeta)
				enabledChanged := entry.IsEnabled != nil && *entry.IsEnabled != existingChunk.IsEnabled
				recommendedChanged := entry.IsRecommended != nil &&
					*entry.IsRecommended != existingChunk.Flags.HasFlag(types.ChunkFlagRecommended)
				answerStrategyChanged := meta.AnswerStrategy != existingMeta.AnswerStrategy
				if existingChunk.ContentHash == newHash && !enabledChanged && !recommendedChanged && !answerStrategyChanged {
					skippedCount++
					logger.Infof(ctx, "Skipping merge for unchanged FAQ entry: %s", meta.StandardQuestion)
					continue
				}

				oldAnswerStr := strings.Join(existingMeta.Answers, "##")
				newAnswerStr := strings.Join(meta.Answers, "##")
				oldSimilarSet := make(map[string]bool, len(existingMeta.SimilarQuestions))
				for _, q := range existingMeta.SimilarQuestions {
					oldSimilarSet[q] = true
				}
				oldNegativeSet := make(map[string]bool, len(existingMeta.NegativeQuestions))
				for _, q := range existingMeta.NegativeQuestions {
					oldNegativeSet[q] = true
				}
				newSimilarCount := 0
				for _, q := range mergedSimilar {
					if !oldSimilarSet[q] {
						newSimilarCount++
					}
				}
				newNegativeCount := 0
				for _, q := range mergedNegative {
					if !oldNegativeSet[q] {
						newNegativeCount++
					}
				}

				mergeOps = append(mergeOps, faqMergeOperation{
					Entry:         entry,
					ExistingChunk: existingChunk,
					OldMeta:       existingMeta,
					MergedMeta:    mergedMeta,
					Detail: types.FAQMergeDetail{
						Index:            entryIdx,
						StandardQuestion: meta.StandardQuestion,
						AnswerChanged:    oldAnswerStr != newAnswerStr,
						NewSimilarCount:  newSimilarCount,
						NewNegativeCount: newNegativeCount,
					},
				})
				continue
			}
		}

		// Reaching here means it is neither a merge candidate nor conflicts with existing KB / current batch standard questions / similar questions.
		// Under normal circumstances these conflicts should already be caught at the executeFAQDryRunValidation stage; this
		// is only a defensive fallback; log at Warn level to help troubleshoot validation-layer misses.
		if existingQuestions[meta.StandardQuestion] || batchQuestions[meta.StandardQuestion] {
			skippedCount++
			logger.Warnf(ctx,
				"calculateAppendOperations: dropping FAQ entry with duplicate standard question %q "+
					"(should have been filtered at validation); entry_idx=%d",
				meta.StandardQuestion, entryIdx)
			continue
		}

		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}
		newEntries = append(newEntries, entry)
	}

	return newEntries, mergeOps, skippedCount, nil
}

// calculateReplaceOperations calculates the entries to delete, create, and update in Replace mode
// and also filters out entries with duplicate standard questions or similar questions within the same batch
func (s *knowledgeService) calculateReplaceOperations(ctx context.Context,
	tenantID uint64, knowledgeID string, newEntries []types.FAQEntryPayload,
) ([]types.FAQEntryPayload, []*types.Chunk, int, error) {
	// Get kbID for resolving the tag
	var kbID string
	if len(newEntries) > 0 {
		// Get kbID from knowledgeID
		knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to get knowledge: %w", err)
		}
		if knowledge != nil {
			kbID = knowledge.KnowledgeBaseID
		}
	}

	// Calculate the content hash for all new entries, and build a hash-to-entry mapping at the same time
	type entryWithHash struct {
		entry types.FAQEntryPayload
		hash  string
		meta  *types.FAQChunkMetadata
	}
	entriesWithHash := make([]entryWithHash, 0, len(newEntries))
	newHashSet := make(map[string]bool)
	// Used for deduplicating standard questions and similar questions within the batch
	batchQuestions := make(map[string]bool)
	batchSkippedCount := 0

	for _, entry := range newEntries {
		meta, err := sanitizeFAQEntryPayload(&entry)
		if err != nil {
			batchSkippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry in replace mode: %v", err)
			continue
		}

		// Check whether the standard question is duplicated within the same batch
		if batchQuestions[meta.StandardQuestion] {
			batchSkippedCount++
			logger.Infof(ctx, "Skipping FAQ entry with duplicate standard question in batch: %s", meta.StandardQuestion)
			continue
		}

		// Check whether the similar question is duplicated within the same batch
		hasDuplicateSimilar := false
		for _, q := range meta.SimilarQuestions {
			if batchQuestions[q] {
				hasDuplicateSimilar = true
				logger.Infof(ctx, "Skipping FAQ entry with duplicate similar question in batch: %s (standard: %s)", q, meta.StandardQuestion)
				break
			}
		}
		if hasDuplicateSimilar {
			batchSkippedCount++
			continue
		}

		// Add the current entry's standard question and similar questions to the batch set
		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}

		hash := types.CalculateFAQContentHash(meta)
		if hash != "" {
			entriesWithHash = append(entriesWithHash, entryWithHash{entry: entry, hash: hash, meta: meta})
			newHashSet[hash] = true
		}
	}

	// Query all existing chunks
	allExistingChunks, err := s.chunkRepo.ListAllFAQChunksByKnowledgeID(ctx, tenantID, knowledgeID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to list existing chunks: %w", err)
	}

	// Filter chunks matching the new entries' hash in memory, and build a map
	existingHashMap := make(map[string]*types.Chunk)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash != "" && newHashSet[chunk.ContentHash] {
			existingHashMap[chunk.ContentHash] = chunk
		}
	}

	// Compute the chunks that need to be deleted (present in the database but absent from the new batch, or with a mismatched hash)
	chunksToDelete := make([]*types.Chunk, 0)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash == "" {
			// If there is no hash, it needs to be deleted (may be old data)
			chunksToDelete = append(chunksToDelete, chunk)
		} else if !newHashSet[chunk.ContentHash] {
			// Hash not present in the new entries, needs to be deleted
			chunksToDelete = append(chunksToDelete, chunk)
		}
	}

	// Batch preload tag info to avoid querying the database row by row in the loop
	// Collect all tag_id and tag_name that need to be queried
	tagSeqIDSet := make(map[int64]bool)
	tagNameSet := make(map[string]bool)
	for _, ewh := range entriesWithHash {
		if ewh.entry.TagID != 0 {
			tagSeqIDSet[ewh.entry.TagID] = true
		} else if ewh.entry.TagName != "" {
			tagNameSet[ewh.entry.TagName] = true
		} else {
			tagNameSet[types.UntaggedTagName] = true
		}
	}

	// Batch query tag by seq_id
	tagSeqIDToUUID := make(map[int64]string)
	if len(tagSeqIDSet) > 0 {
		seqIDs := make([]int64, 0, len(tagSeqIDSet))
		for seqID := range tagSeqIDSet {
			seqIDs = append(seqIDs, seqID)
		}
		tags, err := s.tagRepo.GetBySeqIDs(ctx, tenantID, seqIDs)
		if err != nil {
			logger.Warnf(ctx, "Failed to batch load tags by seq_ids: %v", err)
		} else {
			for _, tag := range tags {
				tagSeqIDToUUID[tag.SeqID] = tag.ID
			}
		}
	}

	// Batch query tag by name
	tagNameToUUID := make(map[string]string)
	if len(tagNameSet) > 0 && kbID != "" {
		for name := range tagNameSet {
			if tag, err := s.tagRepo.GetByName(ctx, tenantID, kbID, name); err == nil && tag != nil {
				tagNameToUUID[name] = tag.ID
			}
		}
	}

	logger.Infof(ctx, "Preloaded %d tags by seq_id, %d tags by name for %d entries",
		len(tagSeqIDToUUID), len(tagNameToUUID), len(entriesWithHash))

	// resolveTagIDFromCache resolves the tag ID from the cache, falling back to a database query on a cache miss
	resolveTagIDFromCache := func(entry *types.FAQEntryPayload) (string, error) {
		if entry.TagID != 0 {
			if uuid, ok := tagSeqIDToUUID[entry.TagID]; ok {
				return uuid, nil
			}
			// Cache miss, fall back to a database query (may need to create)
			return s.resolveTagID(ctx, kbID, entry)
		}
		tagName := entry.TagName
		if tagName == "" {
			tagName = types.UntaggedTagName
		}
		if uuid, ok := tagNameToUUID[tagName]; ok {
			return uuid, nil
		}
		// Cache miss, fall back to a database query (may need to create)
		return s.resolveTagID(ctx, kbID, entry)
	}

	// Compute the entries that need to be created (reusing the already computed hash to avoid recomputation)
	entriesToProcess := make([]types.FAQEntryPayload, 0, len(entriesWithHash))
	skippedCount := batchSkippedCount

	for idx, ewh := range entriesWithHash {
		// Print a progress log every 1000 entries processed
		if idx > 0 && idx%1000 == 0 {
			logger.Infof(ctx, "calculateReplaceOperations progress: %d/%d entries processed", idx, len(entriesWithHash))
		}

		existingChunk := existingHashMap[ewh.hash]
		if existingChunk != nil {
			// Hash matches, check whether the tag has changed
			newTagID, err := resolveTagIDFromCache(&ewh.entry)
			if err != nil {
				logger.Warnf(ctx, "Failed to resolve tag for entry, treating as new: %v", err)
				entriesToProcess = append(entriesToProcess, ewh.entry)
				continue
			}

			enabledChanged := ewh.entry.IsEnabled != nil && *ewh.entry.IsEnabled != existingChunk.IsEnabled
			recommendedChanged := ewh.entry.IsRecommended != nil &&
				*ewh.entry.IsRecommended != existingChunk.Flags.HasFlag(types.ChunkFlagRecommended)
			answerStrategyChanged := false
			if existingMeta, metaErr := existingChunk.FAQMetadata(); metaErr == nil && existingMeta != nil {
				answerStrategyChanged = ewh.meta.AnswerStrategy != existingMeta.AnswerStrategy
			}

			if existingChunk.TagID != newTagID || enabledChanged || recommendedChanged || answerStrategyChanged {
				if existingChunk.TagID != newTagID {
					logger.Infof(ctx, "FAQ entry tag changed from %s to %s, will update", existingChunk.TagID, newTagID)
				}
				if enabledChanged || recommendedChanged || answerStrategyChanged {
					logger.Infof(ctx, "FAQ entry operational fields changed (enabled: %v->%v, recommended: %v->%v, answerStrategy changed: %v), will update",
						existingChunk.IsEnabled, ewh.entry.IsEnabled,
						existingChunk.Flags.HasFlag(types.ChunkFlagRecommended), ewh.entry.IsRecommended,
						answerStrategyChanged)
				}
				chunksToDelete = append(chunksToDelete, existingChunk)
				entriesToProcess = append(entriesToProcess, ewh.entry)
			} else {
				// Hash, tag, and operational status are all the same, skip
				skippedCount++
			}
			continue
		}

		// Hash mismatched or missing, needs to be created
		entriesToProcess = append(entriesToProcess, ewh.entry)
	}

	return entriesToProcess, chunksToDelete, skippedCount, nil
}

// executeFAQImport performs the actual FAQ import logic
func (s *knowledgeService) executeFAQImport(ctx context.Context, taskID string, kbID string,
	payload *types.FAQBatchUpsertPayload, tenantID uint64, processedCount int,
	progress *types.FAQImportProgress,
) (err error) {
	// Save the knowledge base and embedding model info, used for index cleanup
	var kb *types.KnowledgeBase
	var embeddingModel embedding.Embedder
	totalEntries := len(payload.Entries) + processedCount

	// Recovery mechanism: if any error or panic occurs, roll back all created chunks and index data
	defer func() {
		// Recover from panic
		if r := recover(); r != nil {
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			stack := string(buf[:n])
			logger.Errorf(ctx, "FAQ import task %s panicked: %v\n%s", taskID, r, stack)
			err = fmt.Errorf("panic during FAQ import: %v", r)
		}
	}()

	kb, ctx, err = s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}

	kb.EnsureDefaults()

	// Get the embedding model, used for subsequent index cleanup
	embeddingModel, err = s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return fmt.Errorf("failed to get embedding model: %w", err)
	}
	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return err
	}

	// Get the index mode
	indexMode := types.FAQIndexModeQuestionOnly
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	// Incremental update logic: compute the entries that need to be processed
	var entriesToProcess []types.FAQEntryPayload
	var chunksToDelete []*types.Chunk
	var skippedCount int

	if payload.Mode == types.FAQBatchModeReplace {
		// Replace mode: compute the entries to delete, create, and update
		entriesToProcess, chunksToDelete, skippedCount, err = s.calculateReplaceOperations(
			ctx,
			tenantID,
			faqKnowledge.ID,
			payload.Entries,
		)
		if err != nil {
			return fmt.Errorf("failed to calculate replace operations: %w", err)
		}

		// Delete the chunks that need to be deleted (including old chunks that need to be updated)
		if len(chunksToDelete) > 0 {
			chunkIDsToDelete := make([]string, 0, len(chunksToDelete))
			for _, chunk := range chunksToDelete {
				chunkIDsToDelete = append(chunkIDsToDelete, chunk.ID)
			}
			if err := s.chunkRepo.DeleteChunks(ctx, tenantID, chunkIDsToDelete); err != nil {
				return fmt.Errorf("failed to delete chunks: %w", err)
			}
			// Delete the index
			if err := s.deleteFAQChunkVectors(ctx, kb, faqKnowledge, chunksToDelete); err != nil {
				return fmt.Errorf("failed to delete chunk vectors: %w", err)
			}
			logger.Infof(ctx, "FAQ import task %s: deleted %d chunks (including updates)", taskID, len(chunksToDelete))
		}
	} else {
		// Append mode (smart merge): entries whose standard question already exists go through merge ops, the rest are treated as new
		var mergeOps []faqMergeOperation
		entriesToProcess, mergeOps, skippedCount, err = s.calculateAppendOperations(ctx, tenantID, kb.ID, payload.Entries)
		if err != nil {
			return fmt.Errorf("failed to calculate append operations: %w", err)
		}

		if len(mergeOps) > 0 {
			mergedCount, mergeErr := s.executeFAQMergeOperations(ctx, taskID, kb, faqKnowledge, embeddingModel, indexMode, mergeOps, progress)
			if mergeErr != nil {
				return fmt.Errorf("failed to execute merge operations: %w", mergeErr)
			}
			logger.Infof(ctx, "FAQ import task %s: merged %d entries", taskID, mergedCount)
			progress.MergedCount = mergedCount
			for _, op := range mergeOps {
				progress.MergeDetails = append(progress.MergeDetails, op.Detail)
			}
		}
	}
	logger.Infof(
		ctx,
		"FAQ import task %s: total entries: %d, new to create: %d, skipped: %d, merged: %d",
		taskID,
		len(payload.Entries),
		len(entriesToProcess),
		skippedCount,
		progress.MergedCount,
	)

	// If there are no entries to process, return directly
	if len(entriesToProcess) == 0 {
		logger.Infof(ctx, "FAQ import task %s: no new entries to create", taskID)
		return nil
	}

	// Process the entries to be created in batches
	remainingEntries := len(entriesToProcess)
	totalStartTime := time.Now()
	actualProcessed := skippedCount + processedCount + progress.MergedCount

	logger.Infof(
		ctx,
		"FAQ import task %s: starting batch processing, remaining entries: %d, total entries: %d, batch size: %d",
		taskID,
		remainingEntries,
		totalEntries,
		faqImportBatchSize,
	)

	for i := 0; i < remainingEntries; i += faqImportBatchSize {
		batchStartTime := time.Now()
		end := i + faqImportBatchSize
		if end > remainingEntries {
			end = remainingEntries
		}

		batch := entriesToProcess[i:end]
		logger.Infof(ctx, "FAQ import task %s: processing batch %d-%d (%d entries)", taskID, i+1, end, len(batch))

		// Build chunks
		buildStartTime := time.Now()
		chunks := make([]*types.Chunk, 0, len(batch))
		chunkIds := make([]string, 0, len(batch))
		for idx, entry := range batch {
			meta, err := sanitizeFAQEntryPayload(&entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to sanitize entry at index %d: %w", i+idx, err)
			}

			// Parse TagID
			tagID, err := s.resolveTagID(ctx, kbID, &entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to resolve tag for entry at index %d: %w", i+idx, err)
			}

			isEnabled := true
			if entry.IsEnabled != nil {
				isEnabled = *entry.IsEnabled
			}
			// ChunkIndex calculation: startChunkIndex + (i+idx) + initialProcessed
			chunk := &types.Chunk{
				ID:              uuid.New().String(),
				TenantID:        tenantID,
				KnowledgeID:     faqKnowledge.ID,
				KnowledgeBaseID: kb.ID,
				Content:         buildFAQChunkContent(meta, indexMode),
				// ChunkIndex:      0,
				IsEnabled: isEnabled,
				ChunkType: types.ChunkTypeFAQ,
				TagID:     tagID,                        // Use the parsed TagID
				Status:    int(types.ChunkStatusStored), // store but not indexed
			}
			// If an ID was specified (for data migration), set SeqID
			if entry.ID != nil && *entry.ID > 0 {
				chunk.SeqID = *entry.ID
			}
			if err := chunk.SetFAQMetadata(meta); err != nil {
				return fmt.Errorf("failed to set FAQ metadata: %w", err)
			}
			chunks = append(chunks, chunk)
			chunkIds = append(chunkIds, chunk.ID)
		}
		buildDuration := time.Since(buildStartTime)
		logger.Debugf(ctx, "FAQ import task %s: batch %d-%d built %d chunks in %v, chunk IDs: %v",
			taskID, i+1, end, len(chunks), buildDuration, chunkIds)
		// Create chunks
		createStartTime := time.Now()
		if err := s.chunkService.CreateChunks(ctx, chunks); err != nil {
			return fmt.Errorf("failed to create chunks: %w", err)
		}
		createDuration := time.Since(createStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d created %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			createDuration,
		)

		// Index chunks
		indexStartTime := time.Now()
		// Note: if indexing fails, the recovery mechanism in defer will automatically roll back the created chunks and index data
		if err := s.indexFAQChunks(ctx, kb, faqKnowledge, chunks, embeddingModel, true, false); err != nil {
			return fmt.Errorf("failed to index chunks: %w", err)
		}
		indexDuration := time.Since(indexStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d indexed %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			indexDuration,
		)

		// Update the chunks' Status to indexed: every row gets the same value, so a single
		// UPDATE ... WHERE id IN is enough, without sending content and other fields back again.
		for _, chunk := range chunks {
			chunk.Status = int(types.ChunkStatusIndexed) // indexed
		}
		if err := s.chunkRepo.UpdateChunkFieldsByIDs(ctx, tenantID, chunkIds, map[string]interface{}{
			"status": int(types.ChunkStatusIndexed),
		}); err != nil {
			return fmt.Errorf("failed to update chunks status: %w", err)
		}

		// Collect successful entry info (tags are loaded once per batch instead of one query per entry)
		tagsByID := s.loadFAQTagsForChunks(ctx, tenantID, chunks)
		for idx, chunk := range chunks {
			entryIdx := i + idx + processedCount // Original entry index
			meta, _ := chunk.FAQMetadata()
			standardQ := ""
			if meta != nil {
				standardQ = meta.StandardQuestion
			}
			tagID, tagName := faqTagInfo(tagsByID, chunk.TagID)
			progress.SuccessEntries = append(progress.SuccessEntries, types.FAQSuccessEntry{
				Index:            entryIdx,
				SeqID:            chunk.SeqID,
				TagID:            tagID,
				TagName:          tagName,
				StandardQuestion: standardQ,
			})
		}

		actualProcessed += len(batch)
		// Update task progress
		progress := int(float64(actualProcessed) / float64(totalEntries) * 100)
		if err := s.updateFAQImportProgressStatus(ctx, taskID, "", 0, types.FAQImportStatusProcessing, progress, totalEntries, actualProcessed, fmt.Sprintf("Processing entry %d/%d", actualProcessed, totalEntries), ""); err != nil {
			logger.Errorf(ctx, "Failed to update task progress: %v", err)
		}

		batchDuration := time.Since(batchStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d completed in %v (build: %v, create: %v, index: %v), total progress: %d/%d (%d%%)",
			taskID,
			i+1,
			end,
			batchDuration,
			buildDuration,
			createDuration,
			indexDuration,
			actualProcessed,
			totalEntries,
			progress,
		)
	}

	totalDuration := time.Since(totalStartTime)
	var avgPerEntry time.Duration
	if actualProcessed > 0 {
		avgPerEntry = totalDuration / time.Duration(actualProcessed)
	}
	logger.Infof(
		ctx,
		"FAQ import task %s: all batches completed, processed: %d entries (skipped: %d) in %v, avg: %v per entry",
		taskID,
		actualProcessed,
		skippedCount,
		totalDuration,
		avgPerEntry,
	)

	return nil
}

// updateFAQImportProgressStatus updates the FAQ import progress in Redis
func (s *knowledgeService) updateFAQImportProgressStatus(
	ctx context.Context,
	taskID string,
	instanceID string,
	enqueuedAt int64,
	status types.FAQImportTaskStatus,
	progress, total, processed int,
	message, errorMsg string,
) error {
	// Get existing progress from Redis
	existingProgress, err := s.GetFAQImportProgress(ctx, taskID)
	if err != nil {
		// If not found, create a new progress entry
		existingProgress = &types.FAQImportProgress{
			TaskID:    taskID,
			CreatedAt: time.Now().Unix(),
		}
	}

	// Update progress fields
	existingProgress.Status = status
	existingProgress.Progress = progress
	existingProgress.Total = total
	existingProgress.Processed = processed
	if message != "" {
		existingProgress.Message = message
	}
	existingProgress.Error = errorMsg
	if status == types.FAQImportStatusCompleted {
		existingProgress.Error = ""
	}

	// Clear running key when task completes or fails
	if status == types.FAQImportStatusCompleted || status == types.FAQImportStatusFailed {
		if existingProgress.KBID != "" {
			if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, existingProgress.KBID, taskID, instanceID, enqueuedAt); clearErr != nil {
				logger.Errorf(ctx, "Failed to clear running FAQ import task ID: %v", clearErr)
			}
		}
	}

	return s.saveFAQImportProgress(ctx, existingProgress)
}

// cleanupFAQEntriesFileOnFinalFailure cleans up the entries file in object storage when the task finally fails
// Only run cleanup when retryCount >= maxRetry; otherwise the file is still needed for retries
func (s *knowledgeService) cleanupFAQEntriesFileOnFinalFailure(ctx context.Context, entriesURL string, retryCount, maxRetry int) {
	if entriesURL == "" || retryCount < maxRetry {
		return
	}
	if err := s.fileSvc.DeleteFile(ctx, entriesURL); err != nil {
		logger.Warnf(ctx, "Failed to delete FAQ entries file from object storage on final failure: %v", err)
	} else {
		logger.Infof(ctx, "Deleted FAQ entries file from object storage on final failure: %s", entriesURL)
	}
}

// runningFAQImportInfo stores the task ID and enqueued timestamp for uniquely identifying a task instance
type runningFAQImportInfo struct {
	TaskID     string `json:"task_id"`
	EnqueuedAt int64  `json:"enqueued_at"`
	InstanceID string `json:"instance_id,omitempty"`
}

// getRunningFAQImportInfo checks if there's a running FAQ import task for the given KB
// Returns the task info if found, nil otherwise
func (s *knowledgeService) getRunningFAQImportInfo(ctx context.Context, kbID string) (*runningFAQImportInfo, error) {
	if s.redisClient == nil {
		if v, ok := s.memFAQRunningImport.Load(kbID); ok {
			return v.(*runningFAQImportInfo), nil
		}
		return nil, nil
	}
	key := getFAQImportRunningKey(kbID)
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get running FAQ import task: %w", err)
	}

	// Try to parse as JSON first (new format)
	var info runningFAQImportInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		// Fallback: old format was just taskID string
		return &runningFAQImportInfo{TaskID: data, EnqueuedAt: 0}, nil
	}
	return &info, nil
}

// getRunningFAQImportTaskID checks if there's a running FAQ import task for the given KB
// Returns the task ID if found, empty string otherwise (for backward compatibility)
func (s *knowledgeService) getRunningFAQImportTaskID(ctx context.Context, kbID string) (string, error) {
	info, err := s.getRunningFAQImportInfo(ctx, kbID)
	if err != nil {
		return "", err
	}
	if info == nil {
		return "", nil
	}
	return info.TaskID, nil
}

// setRunningFAQImportInfo sets the running task info for a KB
func (s *knowledgeService) setRunningFAQImportInfo(ctx context.Context, kbID string, info *runningFAQImportInfo) error {
	if s.redisClient == nil {
		s.memFAQRunningImport.Store(kbID, info)
		return nil
	}
	key := getFAQImportRunningKey(kbID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal running info: %w", err)
	}
	return s.redisClient.Set(ctx, key, data, faqImportProgressTTL).Err()
}

// clearRunningFAQImportTaskID clears the running task ID for a KB
func (s *knowledgeService) clearRunningFAQImportTaskID(ctx context.Context, kbID string) error {
	if s.redisClient == nil {
		s.memFAQRunningImport.Delete(kbID)
		return nil
	}
	key := getFAQImportRunningKey(kbID)
	return s.redisClient.Del(ctx, key).Err()
}

func (s *knowledgeService) clearRunningFAQImportInfoIfMatches(ctx context.Context, kbID, taskID, instanceID string, enqueuedAt int64) error {
	if s.redisClient == nil {
		if v, ok := s.memFAQRunningImport.Load(kbID); ok {
			info, _ := v.(*runningFAQImportInfo)
			if runningFAQImportInfoMatches(info, taskID, instanceID, enqueuedAt) {
				s.memFAQRunningImport.Delete(kbID)
			}
		}
		return nil
	}

	info, err := s.getRunningFAQImportInfo(ctx, kbID)
	if err != nil {
		return err
	}
	if !runningFAQImportInfoMatches(info, taskID, instanceID, enqueuedAt) {
		return nil
	}

	key := getFAQImportRunningKey(kbID)
	return s.redisClient.Del(ctx, key).Err()
}

func runningFAQImportInfoMatches(info *runningFAQImportInfo, taskID, instanceID string, enqueuedAt int64) bool {
	if info == nil || info.TaskID != taskID {
		return false
	}
	if info.InstanceID != "" && instanceID != "" {
		return info.InstanceID == instanceID
	}
	return enqueuedAt == 0 || info.EnqueuedAt == 0 || info.EnqueuedAt == enqueuedAt
}

// incrementalIndexFAQEntry incrementally updates the index for a FAQ entry
// Only compute embeddings and update the index for changed content, skipping unchanged parts
func (s *knowledgeService) incrementalIndexFAQEntry(
	ctx context.Context,
	kb *types.KnowledgeBase,
	knowledge *types.Knowledge,
	chunk *types.Chunk,
	embeddingModel embedding.Embedder,
	oldStandardQuestion string,
	oldSimilarQuestions []string,
	oldAnswers []string,
	newMeta *types.FAQChunkMetadata,
) error {
	indexStartTime := time.Now()
	logger.Debugf(ctx, "incrementalIndexFAQEntry: starting for chunk=%s, oldSimilarQuestions=%d, newSimilarQuestions=%d",
		chunk.ID, len(oldSimilarQuestions), len(newMeta.SimilarQuestions))

	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, types.MustTenantIDFromContext(ctx), kb.VectorStoreID)
	if err != nil {
		return err
	}

	indexMode := types.FAQIndexModeQuestionAnswer
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	// Normalize old and new data to ensure consistency with buildFAQIndexInfoList's behavior
	// Normalize old data
	oldStandardQuestion = types.NormalizeQuestion(oldStandardQuestion)
	normalizedOldSimilarQuestions := make([]string, 0, len(oldSimilarQuestions))
	for _, q := range oldSimilarQuestions {
		if nq := types.NormalizeQuestion(q); nq != "" {
			normalizedOldSimilarQuestions = append(normalizedOldSimilarQuestions, nq)
		}
	}
	oldSimilarQuestions = normalizedOldSimilarQuestions
	oldAnswers = types.SanitizeStrings(oldAnswers)
	// Normalize new data
	normalizedNewMeta := newMeta.Normalize()

	// Build index content
	buildContent := func(question string, answers []string) string {
		if indexMode == types.FAQIndexModeQuestionAnswer && len(answers) > 0 {
			var builder strings.Builder
			builder.WriteString(question)
			for _, ans := range answers {
				builder.WriteString("\n")
				builder.WriteString(ans)
			}
			return builder.String()
		}
		return question
	}

	// Check whether the answer changed (only affects the index in QuestionAnswer mode)
	answersChanged := indexMode == types.FAQIndexModeQuestionAnswer && !slices.Equal(oldAnswers, normalizedNewMeta.Answers)
	logger.Debugf(ctx, "incrementalIndexFAQEntry: answersChanged=%v (indexMode=%s), oldAnswers=%d, newAnswers=%d",
		answersChanged, indexMode, len(oldAnswers), len(normalizedNewMeta.Answers))

	// Collect index items that need updating
	var indexInfoToUpdate []*types.IndexInfo

	// 1. Check whether the standard question needs updating
	oldStdContent := buildContent(oldStandardQuestion, oldAnswers)
	newStdContent := buildContent(normalizedNewMeta.StandardQuestion, normalizedNewMeta.Answers)
	stdQuestionChanged := oldStdContent != newStdContent
	if stdQuestionChanged {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: standard question changed, sourceID=%s", chunk.ID)
		indexInfoToUpdate = append(indexInfoToUpdate, &types.IndexInfo{
			Content:         newStdContent,
			SourceID:        chunk.ID,
			SourceType:      types.ChunkSourceType,
			ChunkID:         chunk.ID,
			KnowledgeID:     chunk.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			KnowledgeType:   types.KnowledgeTypeFAQ,
			TagID:           chunk.TagID,
			IsEnabled:       chunk.IsEnabled,
			IsRecommended:   chunk.Flags.HasFlag(types.ChunkFlagRecommended),
		})
	}

	// 2. Handle additions/removals/updates of similar questions based on content hash
	// Build the old question set (question -> exists)
	oldQuestionsSet := make(map[string]struct{}, len(oldSimilarQuestions))
	for _, q := range oldSimilarQuestions {
		oldQuestionsSet[q] = struct{}{}
	}

	// Build the new question set
	newQuestionsSet := make(map[string]struct{}, len(normalizedNewMeta.SimilarQuestions))
	for _, q := range normalizedNewMeta.SimilarQuestions {
		newQuestionsSet[q] = struct{}{}
	}

	// Find questions to delete (in the old set but not in the new set)
	var sourceIDsToDelete []string
	var deletedQuestions []string
	for oldQ := range oldQuestionsSet {
		if _, exists := newQuestionsSet[oldQ]; !exists {
			sourceID := fmt.Sprintf("%s-%s", chunk.ID, hashQuestion(oldQ))
			sourceIDsToDelete = append(sourceIDsToDelete, sourceID)
			deletedQuestions = append(deletedQuestions, oldQ)
		}
	}

	// Find questions to add or update
	var addedQuestions, updatedQuestions []string
	for newQ := range newQuestionsSet {
		_, existedBefore := oldQuestionsSet[newQ]
		// Update conditions:
		// 1. New question (didn't exist before)
		// 2. Answer changed (needs re-embedding)
		if !existedBefore || answersChanged {
			sourceID := fmt.Sprintf("%s-%s", chunk.ID, hashQuestion(newQ))
			indexInfoToUpdate = append(indexInfoToUpdate, &types.IndexInfo{
				Content:         buildContent(newQ, normalizedNewMeta.Answers),
				SourceID:        sourceID,
				SourceType:      types.ChunkSourceType,
				ChunkID:         chunk.ID,
				KnowledgeID:     chunk.KnowledgeID,
				KnowledgeBaseID: chunk.KnowledgeBaseID,
				KnowledgeType:   types.KnowledgeTypeFAQ,
				TagID:           chunk.TagID,
				IsEnabled:       chunk.IsEnabled,
				IsRecommended:   chunk.Flags.HasFlag(types.ChunkFlagRecommended),
			})
			if !existedBefore {
				addedQuestions = append(addedQuestions, newQ)
			} else {
				updatedQuestions = append(updatedQuestions, newQ)
			}
		}
	}

	// Output detailed change logs
	if len(deletedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: deleted similar questions: %v", deletedQuestions)
	}
	if len(addedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: added similar questions: %v", addedQuestions)
	}
	if len(updatedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: updated similar questions (answers changed): %v", updatedQuestions)
	}

	// 3. Delete similar-question indexes that no longer exist
	if len(sourceIDsToDelete) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: deleting %d obsolete sourceIDs: %v", len(sourceIDsToDelete), sourceIDsToDelete)
		if delErr := retrieveEngine.DeleteBySourceIDList(ctx, sourceIDsToDelete, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); delErr != nil {
			logger.Warnf(ctx, "incrementalIndexFAQEntry: failed to delete obsolete source IDs: %v", delErr)
		}
	}

	// 4. Batch-index content that needs updating
	newCount := len(normalizedNewMeta.SimilarQuestions)
	if len(indexInfoToUpdate) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: updating %d index entries (skipped %d unchanged)",
			len(indexInfoToUpdate), 1+newCount-len(indexInfoToUpdate))
		if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoToUpdate); err != nil {
			return err
		}
	} else {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: all %d entries unchanged, skipping index update", 1+newCount)
	}

	// 5. Update the knowledge record
	now := time.Now()
	knowledge.UpdatedAt = now
	knowledge.ProcessedAt = &now
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		return err
	}

	totalDuration := time.Since(indexStartTime)
	logger.Debugf(ctx, "incrementalIndexFAQEntry: completed in %v, updated %d/%d entries",
		totalDuration, len(indexInfoToUpdate), 1+newCount)

	return nil
}

func (s *knowledgeService) indexFAQChunks(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge,
	chunks []*types.Chunk, embeddingModel embedding.Embedder,
	adjustStorage bool, needDelete bool,
) error {
	if len(chunks) == 0 {
		return nil
	}
	indexStartTime := time.Now()
	logger.Debugf(ctx, "indexFAQChunks: starting to index %d chunks", len(chunks))

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
	if err != nil {
		return err
	}

	// Build index info
	buildIndexInfoStartTime := time.Now()
	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}
	buildIndexInfoDuration := time.Since(buildIndexInfoStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: built %d index info entries for %d chunks in %v",
		len(indexInfo),
		len(chunks),
		buildIndexInfoDuration,
	)

	var size int64
	if adjustStorage {
		estimateStartTime := time.Now()
		size = retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
		estimateDuration := time.Since(estimateStartTime)
		logger.Debugf(ctx, "indexFAQChunks: estimated storage size %d bytes in %v", size, estimateDuration)
		if tenantInfo.StorageQuota > 0 && tenantInfo.StorageUsed+size > tenantInfo.StorageQuota {
			return types.NewStorageQuotaExceededError()
		}
	}

	// Delete old vectors
	var deleteDuration time.Duration
	if needDelete {
		deleteStartTime := time.Now()
		if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
			logger.Warnf(ctx, "Delete FAQ vectors failed: %v", err)
		}
		deleteDuration = time.Since(deleteStartTime)
		if deleteDuration > 100*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: deleted old vectors for %d chunks in %v", len(chunkIDs), deleteDuration)
		}
	}

	// Batch index (this could be the performance bottleneck)
	batchIndexStartTime := time.Now()
	if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfo); err != nil {
		return err
	}
	batchIndexDuration := time.Since(batchIndexStartTime)
	var avgPerEntry time.Duration
	if len(indexInfo) > 0 {
		avgPerEntry = batchIndexDuration / time.Duration(len(indexInfo))
	}
	logger.Debugf(ctx, "indexFAQChunks: batch indexed %d index info entries in %v (avg: %v per entry)",
		len(indexInfo), batchIndexDuration, avgPerEntry)

	if adjustStorage && size > 0 {
		adjustStartTime := time.Now()
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, size); err == nil {
			tenantInfo.StorageUsed += size
		}
		knowledge.StorageSize += size
		adjustDuration := time.Since(adjustStartTime)
		if adjustDuration > 50*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: adjusted storage in %v", adjustDuration)
		}
	}

	updateStartTime := time.Now()
	now := time.Now()
	knowledge.UpdatedAt = now
	knowledge.ProcessedAt = &now
	err = s.repo.UpdateKnowledge(ctx, knowledge)
	updateDuration := time.Since(updateStartTime)
	if updateDuration > 50*time.Millisecond {
		logger.Debugf(ctx, "indexFAQChunks: updated knowledge in %v", updateDuration)
	}

	totalDuration := time.Since(indexStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: completed indexing %d chunks in %v (build: %v, delete: %v, batchIndex: %v, update: %v)",
		len(chunks),
		totalDuration,
		buildIndexInfoDuration,
		deleteDuration,
		batchIndexDuration,
		updateDuration,
	)

	return err
}

func (s *knowledgeService) deleteFAQChunkVectors(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge, chunks []*types.Chunk,
) error {
	if len(chunks) == 0 {
		return nil
	}
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return err
	}
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
	if err != nil {
		return err
	}

	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}

	size := retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
	if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
		return err
	}
	if size > 0 {
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, -size); err == nil {
			tenantInfo.StorageUsed -= size
			if tenantInfo.StorageUsed < 0 {
				tenantInfo.StorageUsed = 0
			}
		}
		if knowledge.StorageSize >= size {
			knowledge.StorageSize -= size
		} else {
			knowledge.StorageSize = 0
		}
	}
	knowledge.UpdatedAt = time.Now()
	return s.repo.UpdateKnowledge(ctx, knowledge)
}

func faqImportCompletedOutcome(successCount, failedCount, skippedCount int) types.AuditOutcome {
	if successCount > 0 && (failedCount > 0 || skippedCount > 0) {
		return types.AuditOutcomePartial
	}
	if successCount > 0 {
		return types.AuditOutcomeSuccess
	}
	if failedCount > 0 && skippedCount > 0 {
		return types.AuditOutcomePartial
	}
	if failedCount > 0 || skippedCount > 0 {
		return types.AuditOutcomeFailed
	}
	return types.AuditOutcomeSuccess
}

func faqImportActivityDetails(payload *types.FAQImportPayload, progress *types.FAQImportProgress, totalEntries int) map[string]any {
	details := map[string]any{"mode": payload.Mode}
	if progress == nil {
		return details
	}
	total := totalEntries
	if total <= 0 {
		total = progress.Total
	}
	if total > 0 {
		details["total"] = total
	}
	details["count"] = progress.SuccessCount
	if progress.FailedCount > 0 {
		details["failed"] = progress.FailedCount
	}
	skipped := progress.SkippedCount
	if skipped <= 0 && total > 0 {
		skipped = total - progress.SuccessCount - progress.FailedCount
		if skipped < 0 {
			skipped = 0
		}
	}
	if skipped > 0 {
		details["skipped"] = skipped
	}
	return details
}

func (s *knowledgeService) recordFAQImportKBActivity(
	ctx context.Context,
	payload *types.FAQImportPayload,
	progress *types.FAQImportProgress,
	totalEntries int,
	action types.AuditAction,
	outcome types.AuditOutcome,
) {
	if s == nil || payload == nil || payload.DryRun || payload.KBID == "" {
		return
	}
	recordKBActivity(ctx, s.audit, payload.TenantID, payload.KBID, action,
		"faq_entry", payload.KnowledgeID, outcome, faqImportActivityDetails(payload, progress, totalEntries))
}

// ProcessFAQImport handles Asynq FAQ import tasks (including dry run mode)
func (s *knowledgeService) ProcessFAQImport(ctx context.Context, t *asynq.Task) error {
	var payload types.FAQImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "failed to unmarshal FAQ import task payload: %v", err)
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}
	ctx = payload.Initiator.Apply(ctx)
	ctx = withKBActivityTask(ctx, payload.TaskID, kbActivityTrigger(ctx))

	ctx = logger.WithRequestID(ctx, uuid.New().String())
	ctx = logger.WithField(ctx, "faq_import", payload.TaskID)
	ctx = types.WithExecutionTenant(ctx, payload.TenantID)
	kb, err := s.validateFAQKnowledgeBase(ctx, payload.KBID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeBaseNotFound) {
			return fmt.Errorf("%w: FAQ task KB no longer exists", asynq.SkipRetry)
		}
		return err
	}
	ctx, err = access.WithKBTaskWrite(ctx, kb, payload.TenantID)
	if err != nil {
		return fmt.Errorf("%w: FAQ task KB does not belong to its tenant", asynq.SkipRetry)
	}
	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeNotFound) {
			return fmt.Errorf("%w: FAQ task document no longer exists", asynq.SkipRetry)
		}
		return err
	}
	if knowledge == nil || knowledge.TenantID != payload.TenantID || knowledge.KnowledgeBaseID != payload.KBID ||
		knowledge.Type != types.KnowledgeTypeFAQ {
		return fmt.Errorf("%w: FAQ task document does not belong to its KB", asynq.SkipRetry)
	}

	// Get task retry info, used to determine whether this is the last retry
	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	isLastRetry := retryCount >= maxRetry

	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil {
		logger.Errorf(ctx, "failed to get tenant: %v", err)
		return nil
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	// If entries are stored in object storage, download first
	if payload.EntriesURL != "" && len(payload.Entries) == 0 {
		logger.Infof(ctx, "Downloading FAQ entries from object storage: %s", payload.EntriesURL)
		reader, err := s.fileSvc.GetFile(ctx, payload.EntriesURL)
		if err != nil {
			logger.Errorf(ctx, "Failed to download FAQ entries from object storage: %v", err)
			return fmt.Errorf("failed to download entries: %w", err)
		}
		defer reader.Close()

		entriesData, err := io.ReadAll(reader)
		if err != nil {
			logger.Errorf(ctx, "Failed to read FAQ entries data: %v", err)
			return fmt.Errorf("failed to read entries data: %w", err)
		}

		var entries []types.FAQEntryPayload
		if err := json.Unmarshal(entriesData, &entries); err != nil {
			logger.Errorf(ctx, "Failed to unmarshal FAQ entries: %v", err)
			return fmt.Errorf("failed to unmarshal entries: %w", err)
		}

		payload.Entries = entries
		logger.Infof(ctx, "Downloaded %d FAQ entries from object storage", len(entries))
	}

	logger.Infof(ctx, "Processing FAQ import task: task_id=%s, kb_id=%s, total_entries=%d, dry_run=%v, retry=%d/%d",
		payload.TaskID, payload.KBID, len(payload.Entries), payload.DryRun, retryCount, maxRetry)
	if err := s.validateFAQImportTags(ctx, kb, payload.Entries); err != nil {
		return err
	}

	// Save the original total count
	originalTotalEntries := len(payload.Entries)

	// Initialize progress
	// Check whether validation results already exist (used to skip validation on retry)
	// Note: this must be queried before saving new progress, otherwise it will be overwritten
	existingProgress, _ := s.GetFAQImportProgress(ctx, payload.TaskID)

	progress := &types.FAQImportProgress{
		TaskID:         payload.TaskID,
		KBID:           payload.KBID,
		KnowledgeID:    payload.KnowledgeID,
		Status:         types.FAQImportStatusProcessing,
		Progress:       0,
		Total:          originalTotalEntries,
		Processed:      0,
		SuccessCount:   0,
		FailedCount:    0,
		FailedEntries:  make([]types.FAQFailedEntry, 0),
		SuccessEntries: make([]types.FAQSuccessEntry, 0),
		Message:        "Validating entries...",
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
		DryRun:         payload.DryRun,
	}
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to save initial FAQ import progress: %v", err)
	}

	var validEntryIndices []int
	if existingProgress != nil && len(existingProgress.ValidEntryIndices) > 0 {
		// Reuse the previous validation results on retry
		validEntryIndices = existingProgress.ValidEntryIndices
		progress.FailedCount = existingProgress.FailedCount
		progress.FailedEntries = existingProgress.FailedEntries
		logger.Infof(ctx, "Reusing previous validation result: valid=%d, failed=%d",
			len(validEntryIndices), progress.FailedCount)
	} else {
		// Step 1: run validation (needed for both dry run and import mode)
		validEntryIndices = s.executeFAQDryRunValidation(ctx, &payload, progress)
		// Save the validated index for reuse on retry
		progress.ValidEntryIndices = validEntryIndices
		if err := s.saveFAQImportProgress(ctx, progress); err != nil {
			logger.Warnf(ctx, "Failed to save validation result: %v", err)
		}
		logger.Infof(ctx, "FAQ validation completed: total=%d, valid=%d, failed=%d",
			originalTotalEntries, len(validEntryIndices), progress.FailedCount)
	}

	// Dry run mode: return the result directly after validation completes
	if payload.DryRun {
		return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
	}

	// Import mode: check whether there are valid entries to import
	if len(validEntryIndices) == 0 {
		// No valid entries, complete immediately
		return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
	}

	// Extract valid entries
	validEntries := make([]types.FAQEntryPayload, 0, len(validEntryIndices))
	for _, idx := range validEntryIndices {
		validEntries = append(validEntries, payload.Entries[idx])
	}

	// Update progress message
	progress.Message = fmt.Sprintf("Validation complete; importing %d valid entries...", len(validEntries))
	progress.UpdatedAt = time.Now().Unix()
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to update FAQ import progress: %v", err)
	}

	// Check task status - idempotency handling (reuse the previously fetched existingProgress)
	var processedCount int
	if existingProgress != nil {
		if existingProgress.Status == types.FAQImportStatusCompleted {
			logger.Infof(ctx, "FAQ import already completed, skipping: %s", payload.TaskID)
			if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, payload.KBID, payload.TaskID, payload.InstanceID, payload.EnqueuedAt); clearErr != nil {
				logger.Warnf(ctx, "Failed to clear running FAQ import info for completed task: %v", clearErr)
			}
			return nil // Idempotent: already-completed tasks return immediately
		}
		// Get the processed count (note: this is an index relative to validEntries)
		processedCount = existingProgress.Processed - progress.FailedCount // Processed count - Verification Failed count = number of valid entries already imported
		if processedCount < 0 {
			processedCount = 0
		}
		logger.Infof(ctx, "Resuming FAQ import from progress: %d/%d (valid entries)", processedCount, len(validEntries))
	}

	// Idempotency handling: clean up chunks and index data that may have been partially processed
	chunksDeleted, err := s.chunkRepo.DeleteUnindexedChunks(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to delete unindexed chunks: %v", err)
		// If this is the last retry, update the status to failed
		if isLastRetry {
			if updateErr := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusFailed, 0, originalTotalEntries, 0, "Failed to clean unindexed data", err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
			s.recordFAQImportKBActivity(ctx, &payload, progress, originalTotalEntries, types.AuditActionFAQImportFailed, types.AuditOutcomeFailed)
		}
		s.cleanupFAQEntriesFileOnFinalFailure(ctx, payload.EntriesURL, retryCount, maxRetry)
		return fmt.Errorf("failed to delete unindexed chunks: %w", err)
	}
	if len(chunksDeleted) > 0 {
		logger.Infof(ctx, "Deleted unindexed chunks: %d", len(chunksDeleted))

		// Delete index data
		embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
		if err == nil {
			retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
				ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
			if err == nil {
				chunkIDs := make([]string, 0, len(chunksDeleted))
				for _, chunk := range chunksDeleted {
					chunkIDs = append(chunkIDs, chunk.ID)
				}
				if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
					logger.Warnf(ctx, "Failed to delete index data for chunks (may not exist): %v", err)
				} else {
					logger.Infof(ctx, "Successfully deleted index data for %d chunks", len(chunksDeleted))
				}
			}
		}
	}

	// If some valid entries have already been processed, resume from that position
	entriesToImport := validEntries
	importMode := payload.Mode
	if processedCount > 0 && processedCount < len(validEntries) {
		entriesToImport = validEntries[processedCount:]
		// In retry scenarios, if some data was already processed previously, need to switch to Append mode
		// Because the delete operation of Replace mode was already executed on the first run
		// If Replace mode continues to be used, calculateReplaceOperations will mark previously successfully imported data for deletion
		// Causing data loss
		if payload.Mode == types.FAQBatchModeReplace {
			importMode = types.FAQBatchModeAppend
			logger.Infof(ctx, "Switching to Append mode for retry, original mode was Replace")
		}
		logger.Infof(ctx, "Continuing FAQ import from entry %d, remaining: %d entries", processedCount, len(entriesToImport))
	}

	// Build FAQBatchUpsertPayload (using validated valid entries)
	faqPayload := &types.FAQBatchUpsertPayload{
		Entries: entriesToImport,
		Mode:    importMode,
	}

	// Execute FAQ import (passing the processed offset, used for progress calculation)
	if err := s.executeFAQImport(ctx, payload.TaskID, payload.KBID, faqPayload, payload.TenantID, progress.FailedCount+processedCount, progress); err != nil {
		logger.Errorf(ctx, "FAQ import task failed: %s, error: %v", payload.TaskID, err)
		// If this is the last retry, update the status to failed
		if isLastRetry {
			if updateErr := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusFailed, 0, originalTotalEntries, len(validEntries), "Import failed", err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
			s.recordFAQImportKBActivity(ctx, &payload, progress, originalTotalEntries, types.AuditActionFAQImportFailed, types.AuditOutcomeFailed)
		}
		s.cleanupFAQEntriesFileOnFinalFailure(ctx, payload.EntriesURL, retryCount, maxRetry)
		return fmt.Errorf("FAQ import failed: %w", err)
	}

	// Task completed successfully
	logger.Infof(ctx, "FAQ import task completed: %s, imported: %d, failed: %d",
		payload.TaskID, len(progress.SuccessEntries), progress.FailedCount)

	// Final completion handling (generate failed-entries CSV, etc.)
	return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
}

// finalizeFAQValidation completes the FAQ validation/import task, generating a failed-entries CSV (if any)
func (s *knowledgeService) finalizeFAQValidation(ctx context.Context, payload *types.FAQImportPayload,
	progress *types.FAQImportProgress, originalTotalEntries int,
) error {
	// Clean up the entries file in object storage (if any)
	if payload.EntriesURL != "" {
		if err := s.fileSvc.DeleteFile(ctx, payload.EntriesURL); err != nil {
			logger.Warnf(ctx, "Failed to delete FAQ entries file from object storage: %v", err)
		} else {
			logger.Infof(ctx, "Deleted FAQ entries file from object storage: %s", payload.EntriesURL)
		}
	}
	progress.UpdatedAt = time.Now().Unix()

	// If there are failed entries, generate a CSV file
	if len(progress.FailedEntries) > 0 {
		csvURL, err := s.generateFailedEntriesCSV(ctx, payload.TenantID, payload.TaskID, progress.FailedEntries)
		if err != nil {
			logger.Warnf(ctx, "Failed to generate failed entries CSV: %v", err)
		} else {
			progress.FailedEntriesURL = csvURL
			progress.FailedEntries = nil // Clear inline data, use URL instead
			progress.Message += " (failed records exported as CSV)"
		}
	}

	// Final statistics must be calculated before saveFAQImportResultToDatabase.
	progress.Status = types.FAQImportStatusCompleted
	progress.Progress = 100
	progress.Processed = originalTotalEntries

	if len(progress.ValidEntryIndices) > 0 {
		progress.SuccessCount = len(progress.ValidEntryIndices) - progress.PartialFailedCount
	} else if len(progress.SuccessEntries) > 0 {
		progress.SuccessCount = len(progress.SuccessEntries) - progress.PartialFailedCount
	} else {
		progress.SuccessCount = originalTotalEntries - progress.FailedCount - progress.PartialFailedCount
	}
	if progress.SuccessCount < 0 {
		progress.SuccessCount = 0
	}

	if progress.AddedCount == 0 && progress.MergedCount > 0 {
		progress.AddedCount = progress.SuccessCount - progress.MergedCount
		if progress.AddedCount < 0 {
			progress.AddedCount = 0
		}
	} else if progress.AddedCount == 0 {
		progress.AddedCount = progress.SuccessCount
	}

	skippedCount := originalTotalEntries - progress.SuccessCount - progress.PartialFailedCount - progress.FailedCount
	if skippedCount < 0 {
		skippedCount = 0
	}
	progress.SkippedCount = skippedCount

	if payload.DryRun {
		progress.Message = s.buildFAQImportResultMessage("Validation complete", progress)
	} else {
		progress.Message = s.buildFAQImportResultMessage("Import complete", progress)
	}

	// If not in dry-run mode, save the import result statistics to the database
	if !payload.DryRun {
		if err := s.saveFAQImportResultToDatabase(ctx, payload, progress, originalTotalEntries); err != nil {
			logger.Warnf(ctx, "Failed to save FAQ import result to database: %v", err)
		}

		// Only replace mode cleans up unused Tags
		// Append mode should not delete empty tags pre-created by the user
		if payload.Mode == types.FAQBatchModeReplace {
			deletedTags, err := s.tagRepo.DeleteUnusedTags(ctx, payload.TenantID, payload.KBID)
			if err != nil {
				logger.Warnf(ctx, "FAQ import task %s: failed to cleanup unused tags: %v", payload.TaskID, err)
			} else if deletedTags > 0 {
				logger.Infof(ctx, "FAQ import task %s: cleaned up %d unused tags after replace import", payload.TaskID, deletedTags)
			}
		}
	}

	// Use updateFAQImportProgressStatus to ensure the running key is cleaned up correctly
	// But other fields must be saved first, because updateFAQImportProgressStatus does not save all fields
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to save final FAQ import progress: %v", err)
	}

	// Then call the status update to clean up the running key
	if err := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusCompleted,
		100, originalTotalEntries, originalTotalEntries, progress.Message, ""); err != nil {
		logger.Warnf(ctx, "Failed to update final FAQ import status: %v", err)
	}

	logger.Infof(ctx, "FAQ task completed: %s, dry_run=%v, success: %d, added: %d, merged: %d, failed: %d, partial_failed: %d",
		payload.TaskID, payload.DryRun, progress.SuccessCount, progress.AddedCount, progress.MergedCount, progress.FailedCount, progress.PartialFailedCount)

	if !payload.DryRun {
		outcome := faqImportCompletedOutcome(progress.SuccessCount, progress.FailedCount, progress.SkippedCount)
		s.recordFAQImportKBActivity(ctx, payload, progress, originalTotalEntries,
			types.AuditActionFAQImportCompleted, outcome)
	}

	return nil
}

// executeFAQMergeOperations batch-executes append-mode merge operations: updates existing chunks'
// metadata / content / index. Uses ListChunksByID batch loading + SaveChunks transactional
// batch save to reduce DB round trips. If any batch fails, return immediately; whether to roll back
// the entire import is decided by executeFAQImport's defer recovery.
//
// Fan out index updates one by one here when needed (EFPutDocument), instead of merging chunks in bulk
// rebuild: because the underlying index overwrites by SourceID, put directly using the final merged content
// Done.
func (s *knowledgeService) executeFAQMergeOperations(
	ctx context.Context,
	taskID string,
	kb *types.KnowledgeBase,
	faqKnowledge *types.Knowledge,
	embeddingModel embedding.Embedder,
	indexMode types.FAQIndexMode,
	mergeOps []faqMergeOperation,
	progress *types.FAQImportProgress,
) (int, error) {
	if len(mergeOps) == 0 {
		return 0, nil
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	mergedCount := 0

	for batchStart := 0; batchStart < len(mergeOps); batchStart += faqImportBatchSize {
		batchEnd := batchStart + faqImportBatchSize
		if batchEnd > len(mergeOps) {
			batchEnd = len(mergeOps)
		}
		batch := mergeOps[batchStart:batchEnd]

		// 1. Bulk-load the full chunk (calculateAppendOperations only loads a subset of fields,
		// missing status/is_enabled/flags/seq_id, etc. — a direct update would overwrite these fields with zero values)
		chunkIDs := make([]string, len(batch))
		for i, op := range batch {
			chunkIDs[i] = op.ExistingChunk.ID
		}
		fullChunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, chunkIDs)
		if err != nil {
			logger.Errorf(ctx, "FAQ import task %s: failed to batch reload chunks for merge: %v", taskID, err)
			return mergedCount, fmt.Errorf("failed to batch reload chunks for merge: %w", err)
		}
		chunkMap := make(map[string]*types.Chunk, len(fullChunks))
		for _, c := range fullChunks {
			chunkMap[c.ID] = c
		}

		// 2. Apply merged data item by item
		mergedChunks := make([]*types.Chunk, 0, len(batch))
		for _, op := range batch {
			fullChunk, ok := chunkMap[op.ExistingChunk.ID]
			if !ok {
				logger.Errorf(ctx, "FAQ import task %s: chunk %s not found during batch reload", taskID, op.ExistingChunk.ID)
				return mergedCount, fmt.Errorf("chunk %s not found during batch reload", op.ExistingChunk.ID)
			}

			if err := fullChunk.SetFAQMetadata(op.MergedMeta); err != nil {
				logger.Errorf(ctx, "FAQ import task %s: failed to set merged metadata for chunk %s: %v", taskID, fullChunk.ID, err)
				return mergedCount, fmt.Errorf("failed to set merged FAQ metadata: %w", err)
			}

			fullChunk.Content = buildFAQChunkContent(op.MergedMeta, indexMode)
			fullChunk.ContentHash = types.CalculateFAQContentHash(op.MergedMeta)
			fullChunk.UpdatedAt = time.Now()

			// Overwrite the operational status with the new value
			if op.Entry.IsEnabled != nil {
				fullChunk.IsEnabled = *op.Entry.IsEnabled
			}
			if op.Entry.IsRecommended != nil {
				if *op.Entry.IsRecommended {
					fullChunk.Flags = fullChunk.Flags.SetFlag(types.ChunkFlagRecommended)
				} else {
					fullChunk.Flags = fullChunk.Flags.ClearFlag(types.ChunkFlagRecommended)
				}
			}

			mergedChunks = append(mergedChunks, fullChunk)
		}

		// 3. Batch-save in a transaction (GORM Save updates all fields, ensuring metadata/content_hash are persisted)
		if err := s.chunkRepo.SaveChunks(ctx, mergedChunks); err != nil {
			logger.Errorf(ctx, "FAQ import task %s: failed to batch save merged chunks: %v", taskID, err)
			return mergedCount, fmt.Errorf("failed to batch save merged chunks: %w", err)
		}

		// 4. Rebuild the index (EFPutDocument automatically overwrites the same SourceID)
		if err := s.indexFAQChunks(ctx, kb, faqKnowledge, mergedChunks, embeddingModel, false, false); err != nil {
			return mergedCount, fmt.Errorf("failed to re-index merged chunks: %w", err)
		}

		// 5. Collect info on successful entries
		tagsByID := s.loadFAQTagsForChunks(ctx, tenantID, mergedChunks)
		for i, op := range batch {
			chunk := mergedChunks[i]
			meta := op.MergedMeta
			tagID, tagName := faqTagInfo(tagsByID, chunk.TagID)
			progress.SuccessEntries = append(progress.SuccessEntries, types.FAQSuccessEntry{
				Index:            op.Detail.Index,
				SeqID:            chunk.SeqID,
				TagID:            tagID,
				TagName:          tagName,
				StandardQuestion: meta.StandardQuestion,
			})
		}

		mergedCount += len(batch)

		logger.Infof(ctx, "FAQ import task %s: merged batch %d-%d (%d chunks)", taskID, batchStart+1, batchEnd, len(mergedChunks))
	}

	return mergedCount, nil
}

// loadFAQTagsForChunks resolves every distinct tag referenced by chunks with a
// single query. Lookup failures are logged and yield an empty map so the
// import result degrades to "no tag info" instead of aborting the batch.
func (s *knowledgeService) loadFAQTagsForChunks(
	ctx context.Context, tenantID uint64, chunks []*types.Chunk,
) map[string]*types.KnowledgeTag {
	tagsByID := make(map[string]*types.KnowledgeTag)
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, chunk := range chunks {
		if chunk == nil || chunk.TagID == "" {
			continue
		}
		if _, ok := seen[chunk.TagID]; ok {
			continue
		}
		seen[chunk.TagID] = struct{}{}
		ids = append(ids, chunk.TagID)
	}
	if len(ids) == 0 {
		return tagsByID
	}
	tags, err := s.tagRepo.GetByIDs(ctx, tenantID, ids)
	if err != nil {
		logger.Warnf(ctx, "Failed to load FAQ tags for import result: %v", err)
		return tagsByID
	}
	for _, tag := range tags {
		if tag != nil {
			tagsByID[tag.ID] = tag
		}
	}
	return tagsByID
}

// faqTagInfo returns the external (seq_id, name) pair for tagID, or zero values
// when the chunk has no tag or the tag could not be loaded.
func faqTagInfo(tagsByID map[string]*types.KnowledgeTag, tagID string) (int64, string) {
	if tagID == "" {
		return 0, ""
	}
	if tag, ok := tagsByID[tagID]; ok && tag != nil {
		return tag.SeqID, tag.Name
	}
	return 0, ""
}

// buildFAQImportResultMessage builds a human-readable message for the final FAQ import/validation result.
// The frontend displays it directly in the toast / task list, so keep it simple and clear:
//   - Default form: "Import complete / N uploaded / X succeeded [/ Y failed] [/ Z partially failed]"
//   - When MergedCount > 0, switch to the split form: "/ X added / Y merged and updated",
//     so users can see how many historical FAQs were merged in append mode, instead of just the total success count.
//
// Original internal master implementation; not present before HEAD — all completion messages previously read "Processing item N/M".
func (s *knowledgeService) buildFAQImportResultMessage(prefix string, progress *types.FAQImportProgress) string {
	parts := []string{prefix}
	parts = append(parts, fmt.Sprintf("Uploaded %d entries", progress.Total))

	if progress.MergedCount > 0 {
		parts = append(parts, fmt.Sprintf("Added %d entries", progress.AddedCount))
		parts = append(parts, fmt.Sprintf("Merged and updated %d entries", progress.MergedCount))
	} else {
		parts = append(parts, fmt.Sprintf("Succeeded %d entries", progress.SuccessCount))
	}

	if progress.FailedCount > 0 {
		parts = append(parts, fmt.Sprintf("Failed %d entries", progress.FailedCount))
	}
	if progress.PartialFailedCount > 0 {
		parts = append(parts, fmt.Sprintf("Partially failed %d entries", progress.PartialFailedCount))
	}

	return strings.Join(parts, " / ")
}

const (
	faqImportProgressKeyPrefix = "faq_import_progress:"
	faqImportRunningKeyPrefix  = "faq_import_running:"
	faqImportProgressTTL       = 3 * time.Hour
)

// getFAQImportProgressKey returns the Redis key for storing FAQ import progress
func getFAQImportProgressKey(taskID string) string {
	return faqImportProgressKeyPrefix + taskID
}

// getFAQImportRunningKey returns the Redis key for storing running task ID by KB ID
func getFAQImportRunningKey(kbID string) string {
	return faqImportRunningKeyPrefix + kbID
}

// saveFAQImportProgress saves the FAQ import progress to Redis
func (s *knowledgeService) saveFAQImportProgress(ctx context.Context, progress *types.FAQImportProgress) error {
	if s.redisClient == nil {
		progress.UpdatedAt = time.Now().Unix()
		s.memFAQProgress.Store(progress.TaskID, progress)
		return nil
	}
	key := getFAQImportProgressKey(progress.TaskID)
	progress.UpdatedAt = time.Now().Unix()
	data, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("failed to marshal FAQ import progress: %w", err)
	}
	return s.redisClient.Set(ctx, key, data, faqImportProgressTTL).Err()
}

// GetFAQImportProgress retrieves the progress of an FAQ import task
func (s *knowledgeService) GetFAQImportProgress(ctx context.Context, taskID string) (*types.FAQImportProgress, error) {
	if s.redisClient == nil {
		if v, ok := s.memFAQProgress.Load(taskID); ok {
			return v.(*types.FAQImportProgress), nil
		}
		return nil, werrors.NewNotFoundError("FAQ import task not found")
	}
	key := getFAQImportProgressKey(taskID)
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, werrors.NewNotFoundError("FAQ import task not found")
		}
		return nil, fmt.Errorf("failed to get FAQ import progress from Redis: %w", err)
	}

	var progress types.FAQImportProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FAQ import progress: %w", err)
	}

	// If task is completed, enrich with persisted result fields from database
	if progress.Status == types.FAQImportStatusCompleted && progress.KnowledgeID != "" {
		tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
		knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, progress.KnowledgeID)
		if err == nil && knowledge != nil {
			if result, err := knowledge.GetLastFAQImportResult(); err == nil && result != nil {
				progress.SuccessCount = result.SuccessCount
				progress.FailedCount = result.FailedCount
				progress.PartialFailedCount = result.PartialFailedCount
				progress.SkippedCount = result.SkippedCount
				progress.MergedCount = result.MergedCount
				progress.AddedCount = result.AddedCount
				progress.ImportMode = result.ImportMode
				progress.ImportedAt = result.ImportedAt
				progress.DisplayStatus = result.DisplayStatus
				progress.ProcessingTime = result.ProcessingTime
				if result.FailedEntriesURL != "" {
					progress.FailedEntriesURL = result.FailedEntriesURL
				}
			}
		}
	}

	return &progress, nil
}

// UpdateLastFAQImportResultDisplayStatus updates the display status of FAQ import result
func (s *knowledgeService) UpdateLastFAQImportResultDisplayStatus(ctx context.Context, kbID string, displayStatus string) error {
	// Validate the displayStatus parameter
	if displayStatus != "open" && displayStatus != "close" {
		return werrors.NewBadRequestError("invalid display status, must be 'open' or 'close'")
	}

	kb, ctx, err := s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	tenantID := kb.TenantID

	// Look up knowledge of type FAQ
	knowledgeList, err := s.repo.ListKnowledgeByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return fmt.Errorf("failed to list knowledge: %w", err)
	}

	// Look up knowledge of type FAQ
	var faqKnowledge *types.Knowledge
	for _, k := range knowledgeList {
		if k.Type == types.KnowledgeTypeFAQ {
			faqKnowledge = k
			break
		}
	}

	if faqKnowledge == nil {
		return werrors.NewNotFoundError("FAQ knowledge not found in this knowledge base")
	}

	// Parse the current import result
	result, err := faqKnowledge.GetLastFAQImportResult()
	if err != nil {
		return fmt.Errorf("failed to parse FAQ import result: %w", err)
	}

	if result == nil {
		return werrors.NewNotFoundError("no FAQ import result found")
	}

	// Update the display status
	result.DisplayStatus = displayStatus

	// Save the updated result
	if err := faqKnowledge.SetLastFAQImportResult(result); err != nil {
		return fmt.Errorf("failed to set FAQ import result: %w", err)
	}

	// Update the database
	if err := s.repo.UpdateKnowledge(ctx, faqKnowledge); err != nil {
		return fmt.Errorf("failed to update knowledge: %w", err)
	}

	return nil
}
