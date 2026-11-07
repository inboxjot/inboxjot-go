package inboxjot

import (
	"net/url"
	"time"
)

type DocumentVersionReason string

const (
	VersionCheckpoint      DocumentVersionReason = "checkpoint"
	VersionManual          DocumentVersionReason = "manual"
	VersionBeforeAgentEdit DocumentVersionReason = "before_agent_edit"
	VersionBeforeRestore   DocumentVersionReason = "before_restore"
	VersionRestored        DocumentVersionReason = "restored"
	VersionPublished       DocumentVersionReason = "published"
)

type DocumentVersionEditorKind string

const (
	EditorUser  DocumentVersionEditorKind = "user"
	EditorAgent DocumentVersionEditorKind = "agent"
	EditorAPI   DocumentVersionEditorKind = "api"
)

type ListVersionsParams struct {
	Filter  string // "all" or "saved"
	After   string
	Before  string
	PerPage int
}

type CreateVersionParams struct {
	Name string `json:"name"`
}

type UpdateVersionParams struct {
	Name Nullable[string] `json:"name"`
}

type DocumentVersionEditor struct {
	Kind DocumentVersionEditorKind `json:"kind"`
	ID   string                    `json:"id"`
	Name string                    `json:"name"`
}

type DocumentVersion struct {
	ID        string                  `json:"id"`
	Reason    DocumentVersionReason   `json:"reason"`
	Name      *string                 `json:"name"`
	Editors   []DocumentVersionEditor `json:"editors"`
	CreatedAt time.Time               `json:"created_at"`
}

type DocumentVersionDetail struct {
	DocumentVersion
	From      string         `json:"from"`
	ReplyTo   *string        `json:"reply_to"`
	Subject   string         `json:"subject"`
	Preheader string         `json:"preheader"`
	HTML      string         `json:"html"`
	Text      string         `json:"text"`
	Content   map[string]any `json:"content"`
}

type DocumentVersionPage struct {
	Data       []DocumentVersion `json:"data"`
	Total      int64             `json:"total"`
	NextCursor *string           `json:"next_cursor"`
	PrevCursor *string           `json:"prev_cursor"`
}

func buildVersionsQuery(params ListVersionsParams) url.Values {
	query := url.Values{}
	setQuery(query, "filter", params.Filter)
	setPage(query, params.After, params.Before, params.PerPage)

	return query
}
