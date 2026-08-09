package tools

// maxFunctionNameLength is the maximum length for a tool/function name
// imposed by the OpenAI API.
const maxFunctionNameLength = 64

// Tool names constants
const (
	ToolThinking            = "thinking"
	ToolTodoWrite           = "todo_write"
	ToolGrepChunks          = "grep_chunks"
	ToolKnowledgeSearch     = "knowledge_search"
	ToolListKnowledgeChunks = "list_knowledge_chunks"
	ToolQueryKnowledgeGraph = "query_knowledge_graph"
	ToolGetDocumentInfo     = "get_document_info"
	ToolDatabaseQuery       = "database_query"
	ToolDataAnalysis        = "data_analysis"
	ToolDataSchema          = "data_schema"
	ToolWebSearch           = "web_search"
	ToolWebFetch            = "web_fetch"
	// Skills-related tools (only available when skills are enabled)
	ToolExecuteSkillScript = "execute_skill_script"
	ToolReadSkill          = "read_skill"
	// Wiki-related tools (only available when wiki KBs are in scope)
	ToolWikiReadPage      = "wiki_read_page"
	ToolWikiWritePage     = "wiki_write_page"
	ToolWikiReplaceText   = "wiki_replace_text"
	ToolWikiRenamePage    = "wiki_rename_page"
	ToolWikiDeletePage    = "wiki_delete_page"
	ToolWikiSearch        = "wiki_search"
	ToolWikiReadSourceDoc = "wiki_read_source_doc"
	ToolWikiFlagIssue     = "wiki_flag_issue"
	ToolWikiReadIssue     = "wiki_read_issue"
	ToolWikiUpdateIssue   = "wiki_update_issue"
)

// AvailableTool defines a simple tool metadata used by settings APIs.
type AvailableTool struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AvailableToolDefinitions returns the list of tools exposed to the UI.
// Keep this in sync with registered tools in this package.
func AvailableToolDefinitions() []AvailableTool {
	return []AvailableTool{
		{Name: ToolThinking, Label: "Thinking", Description: "Dynamic, reflective problem-solving thought tool"},
		{Name: ToolTodoWrite, Label: "Plan", Description: "Create a structured research plan"},
		{Name: ToolGrepChunks, Label: "Keyword Search", Description: "Quickly locate documents and chunks that contain specific keywords"},
		{Name: ToolKnowledgeSearch, Label: "Semantic Search", Description: "Understand the question and find semantically related content"},
		{Name: ToolListKnowledgeChunks, Label: "List Document Chunks", Description: "Retrieve the full chunk contents of a document"},
		{Name: ToolQueryKnowledgeGraph, Label: "Query Knowledge Graph", Description: "Query relationships from the knowledge graph"},
		{Name: ToolGetDocumentInfo, Label: "Get Document Info", Description: "View document metadata"},
		{Name: ToolDatabaseQuery, Label: "Query Database", Description: "Query information from the database"},
		{Name: ToolDataAnalysis, Label: "Data Analysis", Description: "Understand data files and perform data analysis"},
		{Name: ToolDataSchema, Label: "View Data Schema", Description: "Get metadata for tabular files"},
		{Name: ToolReadSkill, Label: "Read Skill", Description: "Read skill content on demand to learn specialized capabilities"},
		{Name: ToolExecuteSkillScript, Label: "Execute Skill Script", Description: "Run a skill script in a sandboxed environment"},
		{Name: ToolWikiReadPage, Label: "Read Wiki Page", Description: "Read the content of a specific Wiki page"},
		{Name: ToolWikiSearch, Label: "Search Wiki", Description: "Search pages in the Wiki"},
		{Name: ToolWikiReadSourceDoc, Label: "Deep-read Source Doc", Description: "Deep-read a specific source document using knowledge points"},
		{Name: ToolWikiFlagIssue, Label: "Flag Wiki Issue", Description: "Flag factual errors or merge conflicts on a page"},
		{Name: ToolWikiWritePage, Label: "Create/Overwrite Wiki", Description: "Create a new page or fully overwrite an existing page"},
		{Name: ToolWikiReplaceText, Label: "Replace Wiki Text", Description: "Replace specific text in a Wiki page"},
		{Name: ToolWikiRenamePage, Label: "Rename Wiki", Description: "Rename a Wiki page and automatically update related links"},
		{Name: ToolWikiDeletePage, Label: "Delete Wiki", Description: "Delete a Wiki page and automatically clean up related dead links"},
		{Name: ToolWikiReadIssue, Label: "View Wiki Issue", Description: "View details of a specific Wiki page issue"},
		{Name: ToolWikiUpdateIssue, Label: "Update Wiki Issue Status", Description: "Update the status of a specific Wiki page issue"},
	}
}

// DefaultAllowedTools returns the default allowed tools list.
func DefaultAllowedTools() []string {
	return []string{
		ToolThinking,
		ToolTodoWrite,
		ToolKnowledgeSearch,
		ToolGrepChunks,
		ToolListKnowledgeChunks,
		ToolQueryKnowledgeGraph,
		ToolGetDocumentInfo,
		ToolDatabaseQuery,
		ToolDataAnalysis,
		ToolDataSchema,
	}
}
