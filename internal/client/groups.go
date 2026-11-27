package client

import (
	"context"
	"net/http"
	"strings"
)

// This file implements the Grouper group operations. Request/response shapes are
// taken verbatim from confirmed Grouper WS behavior; see docs/api-observations.md.
// All group operations POST to the "groups" path.

// Group is the provider-facing representation of a Grouper group.
type Group struct {
	UUID             string
	Name             string
	DisplayName      string
	DisplayExtension string
	Extension        string
	Description      string
	Enabled          bool
	TypeOfGroup      string
	IDIndex          string
}

// GroupLookup identifies a group for read/delete by exactly one of UUID or Name.
type GroupLookup struct {
	UUID string
	Name string
}

// SaveGroupRequest carries the mutable fields for creating or updating a group.
type SaveGroupRequest struct {
	Name        string
	DisplayName string
	Description string
}

// wsGroup is the on-the-wire Grouper group DTO. Enabled is a "T"/"F" string.
type wsGroup struct {
	UUID             string `json:"uuid,omitempty"`
	Name             string `json:"name,omitempty"`
	DisplayName      string `json:"displayName,omitempty"`
	DisplayExtension string `json:"displayExtension,omitempty"`
	Extension        string `json:"extension,omitempty"`
	Description      string `json:"description,omitempty"`
	Enabled          string `json:"enabled,omitempty"`
	TypeOfGroup      string `json:"typeOfGroup,omitempty"`
	IDIndex          string `json:"idIndex,omitempty"`
}

func (w *wsGroup) toGroup() *Group {
	if w == nil {
		return nil
	}
	return &Group{
		UUID:             w.UUID,
		Name:             w.Name,
		DisplayName:      w.DisplayName,
		DisplayExtension: w.DisplayExtension,
		Extension:        w.Extension,
		Description:      w.Description,
		Enabled:          strings.EqualFold(w.Enabled, "T"),
		TypeOfGroup:      w.TypeOfGroup,
		IDIndex:          w.IDIndex,
	}
}

type wsGroupLookup struct {
	GroupName string `json:"groupName,omitempty"`
	UUID      string `json:"uuid,omitempty"`
}

// --- Find ---

// wsRestFindGroupsRequest uses wsGroupLookups (direct lookup by uuid/name)
// rather than a wsQueryFilter. The query filter was observed to return stale
// group metadata during a post-write refresh, whereas the direct lookup returns
// current data (matching the stem lookup behavior). See docs/api-observations.md.
type wsRestFindGroupsRequest struct {
	WsRestFindGroupsRequest struct {
		WsGroupLookups []wsGroupLookup `json:"wsGroupLookups"`
	} `json:"WsRestFindGroupsRequest"`
}

type wsFindGroupsResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		GroupResults   []wsGroup      `json:"groupResults"`
	} `json:"WsFindGroupsResults"`
}

// GetGroup looks up a single group by UUID or full name. It returns ErrNotFound
// when the group does not exist (Grouper returns an empty groupResults array).
func (c *Client) GetGroup(ctx context.Context, lookup GroupLookup) (*Group, error) {
	const op = "get group"

	var lk wsGroupLookup
	switch {
	case lookup.UUID != "":
		lk.UUID = lookup.UUID
	case lookup.Name != "":
		lk.GroupName = lookup.Name
	default:
		return nil, &APIError{Operation: op, Message: "a group UUID or name is required", Err: ErrUnexpectedResponse}
	}

	var req wsRestFindGroupsRequest
	req.WsRestFindGroupsRequest.WsGroupLookups = []wsGroupLookup{lk}

	var resp wsFindGroupsResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.wsPath("groups"), req, &resp); err != nil {
		return nil, err
	}
	if resp.Results == nil {
		return nil, &APIError{Operation: op, Message: "missing WsFindGroupsResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if len(resp.Results.GroupResults) == 0 {
		return nil, &APIError{Operation: op, Message: "group not found", Err: ErrNotFound}
	}
	return resp.Results.GroupResults[0].toGroup(), nil
}

// --- Save (create/update) ---

type wsGroupToSave struct {
	WsGroup       wsGroup       `json:"wsGroup"`
	WsGroupLookup wsGroupLookup `json:"wsGroupLookup"`
}

type wsRestGroupSaveRequest struct {
	WsRestGroupSaveRequest struct {
		WsGroupToSaves []wsGroupToSave `json:"wsGroupToSaves"`
	} `json:"WsRestGroupSaveRequest"`
}

type wsGroupSaveResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		Results        []struct {
			ResultMetadata ResultMetadata `json:"resultMetadata"`
			WsGroup        *wsGroup       `json:"wsGroup"`
		} `json:"results"`
	} `json:"WsGroupSaveResults"`
}

// SaveGroup creates or updates a group. The parent stem must already exist.
func (c *Client) SaveGroup(ctx context.Context, in SaveGroupRequest) (*Group, error) {
	const op = "save group"

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, &APIError{Operation: op, Message: "group name is required", Err: ErrUnexpectedResponse}
	}

	var req wsRestGroupSaveRequest
	req.WsRestGroupSaveRequest.WsGroupToSaves = []wsGroupToSave{
		{
			WsGroup: wsGroup{
				Name:             name,
				Extension:        lastPathComponent(name),
				DisplayExtension: in.DisplayName,
				Description:      in.Description,
			},
			WsGroupLookup: wsGroupLookup{GroupName: name},
		},
	}

	var resp wsGroupSaveResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.wsPath("groups"), req, &resp); err != nil {
		return nil, err
	}
	if resp.Results == nil {
		return nil, &APIError{Operation: op, Message: "missing WsGroupSaveResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if len(resp.Results.Results) == 0 {
		return nil, &APIError{Operation: op, Message: "no group returned from save", Err: ErrUnexpectedResponse}
	}
	item := resp.Results.Results[0]
	if err := item.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if item.WsGroup == nil {
		return nil, &APIError{Operation: op, Message: "save result missing group", Err: ErrUnexpectedResponse}
	}
	return item.WsGroup.toGroup(), nil
}

// --- Delete ---

type wsRestGroupDeleteRequest struct {
	WsRestGroupDeleteRequest struct {
		WsGroupLookups []wsGroupLookup `json:"wsGroupLookups"`
	} `json:"WsRestGroupDeleteRequest"`
}

type wsGroupDeleteResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		Results        []struct {
			ResultMetadata ResultMetadata `json:"resultMetadata"`
		} `json:"results"`
	} `json:"WsGroupDeleteResults"`
}

// DeleteGroup deletes a group by full name. A group that is already absent is
// treated as success (Grouper returns SUCCESS_GROUP_NOT_FOUND with success=T).
func (c *Client) DeleteGroup(ctx context.Context, lookup GroupLookup) error {
	const op = "delete group"

	if lookup.Name == "" {
		return &APIError{Operation: op, Message: "a group name is required for delete", Err: ErrUnexpectedResponse}
	}

	var req wsRestGroupDeleteRequest
	req.WsRestGroupDeleteRequest.WsGroupLookups = []wsGroupLookup{{GroupName: lookup.Name}}

	var resp wsGroupDeleteResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.wsPath("groups"), req, &resp); err != nil {
		return err
	}
	if resp.Results == nil {
		return &APIError{Operation: op, Message: "missing WsGroupDeleteResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return err
	}
	for _, item := range resp.Results.Results {
		if item.ResultMetadata.Succeeded() {
			continue
		}
		return item.ResultMetadata.AsError(op)
	}
	return nil
}
