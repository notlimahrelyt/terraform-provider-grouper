package client

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// This file implements Grouper direct (immediate) membership operations.
// Request/response shapes are taken verbatim from confirmed Grouper WS behavior;
// see docs/api-observations.md. Membership operations target the
// groups/<groupName>/members path (group name URL-encoded, colon -> %3A).

// SubjectLookup identifies a subject (person or group) for membership
// operations. Callers typically set only SubjectID (and optionally
// SubjectSourceID); the client resolves by subject id OR identifier so a value
// like a NetID matches whether the source keys it as the id or an identifier.
type SubjectLookup struct {
	SubjectID         string
	SubjectIdentifier string
	SubjectSourceID   string
}

// Subject is the resolved subject returned by Grouper. Attributes holds any
// additional subject attributes that were explicitly requested (e.g. an email
// address); it is nil for membership operations, which do not request them.
type Subject struct {
	ID         string
	Name       string
	SourceID   string
	Attributes map[string]string
}

type wsSubject struct {
	ID              string   `json:"id,omitempty"`
	Name            string   `json:"name,omitempty"`
	SourceID        string   `json:"sourceId,omitempty"`
	ResultCode      string   `json:"resultCode,omitempty"`
	Success         string   `json:"success,omitempty"`
	AttributeValues []string `json:"attributeValues,omitempty"`
}

func (w *wsSubject) toSubject() *Subject {
	if w == nil {
		return nil
	}
	return &Subject{ID: w.ID, Name: w.Name, SourceID: w.SourceID}
}

type wsSubjectLookup struct {
	SubjectID         string `json:"subjectId,omitempty"`
	SubjectIdentifier string `json:"subjectIdentifier,omitempty"`
	SubjectSourceID   string `json:"subjectSourceId,omitempty"`
}

// toWs builds the wire lookup, filling both subjectId and subjectIdentifier with
// the provided value so Grouper matches by either (documented behavior).
func (l SubjectLookup) toWs() wsSubjectLookup {
	id, ident := l.SubjectID, l.SubjectIdentifier
	if id == "" {
		id = ident
	}
	if ident == "" {
		ident = id
	}
	return wsSubjectLookup{SubjectID: id, SubjectIdentifier: ident, SubjectSourceID: l.SubjectSourceID}
}

// encodeGroupNameForPath URL-encodes a group name for use as a single path
// segment. Grouper expects the colon separators encoded as %3A.
func encodeGroupNameForPath(name string) string {
	return strings.ReplaceAll(url.PathEscape(name), ":", "%3A")
}

func (c *Client) membersPath(groupName string) string {
	return c.wsPath("groups/" + encodeGroupNameForPath(groupName) + "/members")
}

// subjectItemError classifies a per-subject result from an add/delete response.
// A subject that could not be resolved returns ErrNotFound; other non-success
// per-item codes return a classified error. Nil means success.
func subjectItemError(op string, itemMeta ResultMetadata, ws *wsSubject) error {
	if ws != nil && ws.Success != "" && !strings.EqualFold(ws.Success, "T") {
		if strings.Contains(strings.ToUpper(ws.ResultCode), "SUBJECT_NOT_FOUND") {
			return &APIError{Operation: op, GrouperResultCode: ws.ResultCode, Message: "subject not found", Err: ErrNotFound}
		}
		return &APIError{Operation: op, GrouperResultCode: ws.ResultCode, Message: "subject lookup failed", Err: ErrUnexpectedResponse}
	}
	if !itemMeta.Succeeded() {
		return itemMeta.AsError(op)
	}
	return nil
}

// --- Add ---

type wsRestAddMemberRequest struct {
	WsRestAddMemberRequest struct {
		SubjectLookups     []wsSubjectLookup `json:"subjectLookups"`
		ReplaceAllExisting string            `json:"replaceAllExisting"`
	} `json:"WsRestAddMemberRequest"`
}

type wsAddMemberResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		Results        []struct {
			ResultMetadata ResultMetadata `json:"resultMetadata"`
			WsSubject      *wsSubject     `json:"wsSubject"`
		} `json:"results"`
	} `json:"WsAddMemberResults"`
}

// AddMembership adds a single direct membership. It is idempotent: an existing
// membership returns SUCCESS_ALREADY_EXISTED and is treated as success.
// replaceAllExisting is always "F" so unrelated members are never removed.
func (c *Client) AddMembership(ctx context.Context, groupName string, subj SubjectLookup) (*Subject, error) {
	const op = "add membership"

	var req wsRestAddMemberRequest
	req.WsRestAddMemberRequest.SubjectLookups = []wsSubjectLookup{subj.toWs()}
	req.WsRestAddMemberRequest.ReplaceAllExisting = "F"

	var resp wsAddMemberResults
	if err := c.doJSON(ctx, op, http.MethodPut, c.membersPath(groupName), req, &resp); err != nil {
		return nil, err
	}
	if resp.Results == nil {
		return nil, &APIError{Operation: op, Message: "missing WsAddMemberResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if len(resp.Results.Results) == 0 {
		return nil, &APIError{Operation: op, Message: "no result returned from add", Err: ErrUnexpectedResponse}
	}
	item := resp.Results.Results[0]
	if err := subjectItemError(op, item.ResultMetadata, item.WsSubject); err != nil {
		return nil, err
	}
	return item.WsSubject.toSubject(), nil
}

// --- Has (read) ---

type wsRestHasMemberRequest struct {
	WsRestHasMemberRequest struct {
		SubjectLookups []wsSubjectLookup `json:"subjectLookups"`
		MemberFilter   string            `json:"memberFilter"`
	} `json:"WsRestHasMemberRequest"`
}

type wsHasMemberResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		Results        []struct {
			ResultMetadata ResultMetadata `json:"resultMetadata"`
			WsSubject      *wsSubject     `json:"wsSubject"`
		} `json:"results"`
	} `json:"WsHasMemberResults"`
}

// HasMembership reports whether the subject is a DIRECT (immediate) member of the
// group. It uses memberFilter=Immediate so effective/inherited memberships are
// not reported. If the subject cannot be resolved, it returns ErrNotFound.
func (c *Client) HasMembership(ctx context.Context, groupName string, subj SubjectLookup) (bool, *Subject, error) {
	const op = "read membership"

	var req wsRestHasMemberRequest
	req.WsRestHasMemberRequest.SubjectLookups = []wsSubjectLookup{subj.toWs()}
	req.WsRestHasMemberRequest.MemberFilter = "Immediate"

	var resp wsHasMemberResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.membersPath(groupName), req, &resp); err != nil {
		return false, nil, err
	}
	if resp.Results == nil {
		return false, nil, &APIError{Operation: op, Message: "missing WsHasMemberResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return false, nil, err
	}
	if len(resp.Results.Results) == 0 {
		return false, nil, &APIError{Operation: op, Message: "no result returned from hasMember", Err: ErrUnexpectedResponse}
	}
	item := resp.Results.Results[0]

	// Subject resolution failure => the membership cannot exist.
	if item.WsSubject != nil && item.WsSubject.Success != "" && !strings.EqualFold(item.WsSubject.Success, "T") {
		if strings.Contains(strings.ToUpper(item.WsSubject.ResultCode), "SUBJECT_NOT_FOUND") {
			return false, nil, &APIError{Operation: op, GrouperResultCode: item.WsSubject.ResultCode, Message: "subject not found", Err: ErrNotFound}
		}
		return false, nil, &APIError{Operation: op, GrouperResultCode: item.WsSubject.ResultCode, Message: "subject lookup failed", Err: ErrUnexpectedResponse}
	}

	switch strings.ToUpper(item.ResultMetadata.ResultCode) {
	case "IS_MEMBER":
		return true, item.WsSubject.toSubject(), nil
	case "IS_NOT_MEMBER":
		return false, item.WsSubject.toSubject(), nil
	default:
		if err := item.ResultMetadata.AsError(op); err != nil {
			return false, nil, err
		}
		return false, item.WsSubject.toSubject(), nil
	}
}

// --- Delete ---

type wsRestDeleteMemberRequest struct {
	WsRestDeleteMemberRequest struct {
		SubjectLookups []wsSubjectLookup `json:"subjectLookups"`
	} `json:"WsRestDeleteMemberRequest"`
}

type wsDeleteMemberResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		Results        []struct {
			ResultMetadata ResultMetadata `json:"resultMetadata"`
			WsSubject      *wsSubject     `json:"wsSubject"`
		} `json:"results"`
	} `json:"WsDeleteMemberResults"`
}

// DeleteMembership removes a single direct membership. It removes only the named
// subject; unrelated members are untouched. A subject that is not a member (or
// cannot be resolved) is treated as success, so delete is idempotent.
func (c *Client) DeleteMembership(ctx context.Context, groupName string, subj SubjectLookup) error {
	const op = "delete membership"

	var req wsRestDeleteMemberRequest
	req.WsRestDeleteMemberRequest.SubjectLookups = []wsSubjectLookup{subj.toWs()}

	var resp wsDeleteMemberResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.membersPath(groupName), req, &resp); err != nil {
		return err
	}
	if resp.Results == nil {
		return &APIError{Operation: op, Message: "missing WsDeleteMemberResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return err
	}
	for _, item := range resp.Results.Results {
		// A subject that no longer exists means nothing to delete: success.
		if item.WsSubject != nil && item.WsSubject.Success != "" && !strings.EqualFold(item.WsSubject.Success, "T") {
			if strings.Contains(strings.ToUpper(item.WsSubject.ResultCode), "SUBJECT_NOT_FOUND") {
				continue
			}
			return &APIError{Operation: op, GrouperResultCode: item.WsSubject.ResultCode, Message: "subject lookup failed", Err: ErrUnexpectedResponse}
		}
		if item.ResultMetadata.Succeeded() {
			continue
		}
		return item.ResultMetadata.AsError(op)
	}
	return nil
}
