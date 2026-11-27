package client

import (
	"context"
	"net/http"
	"strings"
)

// This file implements the Grouper stem (folder) operations. Request/response
// shapes are taken verbatim from confirmed Grouper WS behavior; see
// docs/api-observations.md. All stem operations POST to the "stems" path.

// Stem is the provider-facing representation of a Grouper folder. It is
// deliberately decoupled from the wire DTO (wsStem) so future Grouper changes
// stay contained in this package.
type Stem struct {
	UUID             string
	Name             string
	DisplayName      string
	DisplayExtension string
	Extension        string
	Description      string
	IDIndex          string
}

// StemLookup identifies a stem for read/delete by exactly one of UUID or Name.
type StemLookup struct {
	UUID string
	Name string
}

// SaveStemRequest carries the mutable fields for creating or updating a stem.
// Name is the full colon-delimited path and is immutable at the Terraform layer
// (a change forces replacement).
type SaveStemRequest struct {
	Name        string
	DisplayName string
	Description string
}

// wsStem is the on-the-wire Grouper stem DTO.
type wsStem struct {
	UUID             string `json:"uuid,omitempty"`
	Name             string `json:"name,omitempty"`
	DisplayName      string `json:"displayName,omitempty"`
	DisplayExtension string `json:"displayExtension,omitempty"`
	Extension        string `json:"extension,omitempty"`
	Description      string `json:"description,omitempty"`
	IDIndex          string `json:"idIndex,omitempty"`
}

func (w *wsStem) toStem() *Stem {
	if w == nil {
		return nil
	}
	return &Stem{
		UUID:             w.UUID,
		Name:             w.Name,
		DisplayName:      w.DisplayName,
		DisplayExtension: w.DisplayExtension,
		Extension:        w.Extension,
		Description:      w.Description,
		IDIndex:          w.IDIndex,
	}
}

type wsStemLookup struct {
	StemName string `json:"stemName,omitempty"`
	UUID     string `json:"uuid,omitempty"`
}

// --- Find ---

type wsRestFindStemsRequest struct {
	WsRestFindStemsRequest struct {
		WsStemLookups []wsStemLookup `json:"wsStemLookups,omitempty"`
	} `json:"WsRestFindStemsRequest"`
}

type wsFindStemsResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		StemResults    []wsStem       `json:"stemResults"`
	} `json:"WsFindStemsResults"`
}

// GetStem looks up a single stem by UUID or full name. It returns ErrNotFound
// when the stem does not exist (Grouper returns an empty stemResults array).
func (c *Client) GetStem(ctx context.Context, lookup StemLookup) (*Stem, error) {
	const op = "get stem"

	var lk wsStemLookup
	switch {
	case lookup.UUID != "":
		lk.UUID = lookup.UUID
	case lookup.Name != "":
		lk.StemName = lookup.Name
	default:
		return nil, &APIError{Operation: op, Message: "a stem UUID or name is required", Err: ErrUnexpectedResponse}
	}

	var req wsRestFindStemsRequest
	req.WsRestFindStemsRequest.WsStemLookups = []wsStemLookup{lk}

	var resp wsFindStemsResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.wsPath("stems"), req, &resp); err != nil {
		return nil, err
	}
	if resp.Results == nil {
		return nil, &APIError{Operation: op, Message: "missing WsFindStemsResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if len(resp.Results.StemResults) == 0 {
		return nil, &APIError{Operation: op, Message: "stem not found", Err: ErrNotFound}
	}
	return resp.Results.StemResults[0].toStem(), nil
}

// --- Save (create/update) ---

type wsStemToSave struct {
	WsStem       wsStem       `json:"wsStem"`
	WsStemLookup wsStemLookup `json:"wsStemLookup"`
}

type wsRestStemSaveRequest struct {
	WsRestStemSaveRequest struct {
		WsStemToSaves []wsStemToSave `json:"wsStemToSaves"`
	} `json:"WsRestStemSaveRequest"`
}

type wsStemSaveResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		Results        []struct {
			ResultMetadata ResultMetadata `json:"resultMetadata"`
			WsStem         *wsStem        `json:"wsStem"`
		} `json:"results"`
	} `json:"WsStemSaveResults"`
}

// SaveStem creates or updates a stem. Grouper's stemSave is idempotent: an
// unchanged stem returns SUCCESS_NO_CHANGES_NEEDED. The stem name is used as the
// lookup key.
func (c *Client) SaveStem(ctx context.Context, in SaveStemRequest) (*Stem, error) {
	const op = "save stem"

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, &APIError{Operation: op, Message: "stem name is required", Err: ErrUnexpectedResponse}
	}

	var req wsRestStemSaveRequest
	req.WsRestStemSaveRequest.WsStemToSaves = []wsStemToSave{
		{
			WsStem: wsStem{
				Name:             name,
				Extension:        lastPathComponent(name),
				DisplayExtension: in.DisplayName,
				Description:      in.Description,
			},
			WsStemLookup: wsStemLookup{StemName: name},
		},
	}

	var resp wsStemSaveResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.wsPath("stems"), req, &resp); err != nil {
		return nil, err
	}
	if resp.Results == nil {
		return nil, &APIError{Operation: op, Message: "missing WsStemSaveResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if len(resp.Results.Results) == 0 {
		return nil, &APIError{Operation: op, Message: "no stem returned from save", Err: ErrUnexpectedResponse}
	}
	item := resp.Results.Results[0]
	if err := item.ResultMetadata.AsError(op); err != nil {
		return nil, err
	}
	if item.WsStem == nil {
		return nil, &APIError{Operation: op, Message: "save result missing stem", Err: ErrUnexpectedResponse}
	}
	return item.WsStem.toStem(), nil
}

// --- Delete ---

type wsRestStemDeleteRequest struct {
	WsRestStemDeleteRequest struct {
		WsStemLookups []wsStemLookup `json:"wsStemLookups"`
	} `json:"WsRestStemDeleteRequest"`
}

type wsStemDeleteResults struct {
	Results *struct {
		ResultMetadata ResultMetadata `json:"resultMetadata"`
		Results        []struct {
			ResultMetadata ResultMetadata `json:"resultMetadata"`
		} `json:"results"`
	} `json:"WsStemDeleteResults"`
}

// DeleteStem deletes a stem by full name. A stem that is already absent is
// treated as success (Grouper returns SUCCESS_STEM_NOT_FOUND with success=T).
func (c *Client) DeleteStem(ctx context.Context, lookup StemLookup) error {
	const op = "delete stem"

	if lookup.Name == "" {
		return &APIError{Operation: op, Message: "a stem name is required for delete", Err: ErrUnexpectedResponse}
	}

	var req wsRestStemDeleteRequest
	req.WsRestStemDeleteRequest.WsStemLookups = []wsStemLookup{{StemName: lookup.Name}}

	var resp wsStemDeleteResults
	if err := c.doJSON(ctx, op, http.MethodPost, c.wsPath("stems"), req, &resp); err != nil {
		return err
	}
	if resp.Results == nil {
		return &APIError{Operation: op, Message: "missing WsStemDeleteResults in response", Err: ErrUnexpectedResponse}
	}
	if err := resp.Results.ResultMetadata.AsError(op); err != nil {
		return err
	}
	// Per-item: an already-absent stem is a successful no-op. Any other
	// non-success per-item code is surfaced.
	for _, item := range resp.Results.Results {
		if item.ResultMetadata.Succeeded() {
			continue
		}
		return item.ResultMetadata.AsError(op)
	}
	return nil
}

// lastPathComponent returns the final colon-delimited component of a Grouper
// path, used to derive a stem/group extension from its full name.
func lastPathComponent(name string) string {
	if i := strings.LastIndex(name, ":"); i >= 0 {
		return name[i+1:]
	}
	return name
}
