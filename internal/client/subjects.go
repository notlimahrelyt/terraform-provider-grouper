package client

import (
	"context"
	"net/http"
	"strings"
)

// This file implements the Grouper subject lookup (getSubjects). Request and
// response shapes are confirmed against the live deployment; see
// docs/api-observations.md. A NetID is a subject identifier (not the subjectId),
// so the dual-match lookup is used.

type wsRestGetSubjectsRequest struct {
	WsRestGetSubjectsRequest struct {
		WsSubjectLookups      []wsSubjectLookup `json:"wsSubjectLookups"`
		SubjectAttributeNames []string          `json:"subjectAttributeNames,omitempty"`
	} `json:"WsRestGetSubjectsRequest"`
}

type wsGetSubjectsResults struct {
	Results *struct {
		ResultMetadata        ResultMetadata `json:"resultMetadata"`
		SubjectAttributeNames []string       `json:"subjectAttributeNames"`
		WsSubjects            []wsSubject    `json:"wsSubjects"`
	} `json:"WsGetSubjectsResults"`
}

// GetSubject resolves a single subject by id or identifier (dual-match), with an
// optional source. attributeNames lists the subject attributes to retrieve
// (e.g. "name", "emailaddress"); those are returned in Subject.Attributes,
// keyed by the attribute name Grouper echoes back. It returns ErrNotFound when
// the subject does not resolve.
func (c *Client) GetSubject(ctx context.Context, lookup SubjectLookup, attributeNames []string) (*Subject, error) {
	const op = "get subject"

	if strings.TrimSpace(lookup.SubjectID) == "" && strings.TrimSpace(lookup.SubjectIdentifier) == "" {
		return nil, &APIError{Operation: op, Message: "a subject id or identifier is required", Err: ErrUnexpectedResponse}
	}

	var req wsRestGetSubjectsRequest
	req.WsRestGetSubjectsRequest.WsSubjectLookups = []wsSubjectLookup{lookup.toWs()}
	req.WsRestGetSubjectsRequest.SubjectAttributeNames = attributeNames

	var resp wsGetSubjectsResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.wsPath("subjects"), req, &resp); err != nil {
		return nil, err
	}
	if resp.Results == nil {
		return nil, &APIError{Operation: op, Message: "missing WsGetSubjectsResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if len(resp.Results.WsSubjects) == 0 {
		return nil, &APIError{Operation: op, Message: "subject not found", Err: ErrNotFound}
	}

	ws := resp.Results.WsSubjects[0]
	if ws.Success != "" && !strings.EqualFold(ws.Success, "T") {
		if strings.Contains(strings.ToUpper(ws.ResultCode), "SUBJECT_NOT_FOUND") {
			return nil, &APIError{Operation: op, GrouperResultCode: ws.ResultCode, Message: "subject not found", Err: ErrNotFound}
		}
		return nil, &APIError{Operation: op, GrouperResultCode: ws.ResultCode, Message: "subject lookup failed", Err: ErrUnexpectedResponse}
	}

	subject := ws.toSubject()
	// Map requested attribute names to their values. Grouper returns
	// attributeValues positionally against the response's subjectAttributeNames.
	names := resp.Results.SubjectAttributeNames
	if len(names) > 0 {
		attrs := make(map[string]string, len(names))
		for i, name := range names {
			if name == "" {
				continue
			}
			if i < len(ws.AttributeValues) {
				attrs[name] = ws.AttributeValues[i]
			} else {
				attrs[name] = ""
			}
		}
		subject.Attributes = attrs
	}
	return subject, nil
}
