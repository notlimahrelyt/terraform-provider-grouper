package provider

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
)

// fakeGrouper is an in-memory stand-in for the Grouper Web Services REST API.
// It implements only the operations the provider calls, using the request and
// response shapes recorded in docs/api-observations.md, so that lifecycle
// tests can drive every resource through real Terraform plans and applies
// without a Grouper server.
//
// It is deliberately strict where Grouper is: requests must be authenticated,
// must use the configured version segment, and must carry the operation's
// top-level request key. Anything unexpected fails the test.
type fakeGrouper struct {
	t      *testing.T
	server *httptest.Server

	mu       sync.Mutex
	stems    map[string]*fakeObject     // by full name
	groups   map[string]*fakeObject     // by full name
	members  map[string]map[string]bool // group name -> member keys ("source/id")
	subjects []fakeSubject
}

type fakeObject struct {
	UUID             string
	Name             string
	DisplayExtension string
	Description      string
}

type fakeSubject struct {
	ID         string
	Identifier string
	SourceID   string
	Name       string
	Attributes map[string]string
}

const (
	fakeUser     = "terraform-svc"
	fakePassword = "fake-password"
	fakeVersion  = "v2_6_000"
)

// newFakeGrouper starts a fake Grouper with an "example" and "example:apps"
// stem and three people (abc123, xyz456, ext789) in an "ldap" source.
func newFakeGrouper(t *testing.T) *fakeGrouper {
	t.Helper()
	f := &fakeGrouper{
		t:       t,
		stems:   map[string]*fakeObject{},
		groups:  map[string]*fakeObject{},
		members: map[string]map[string]bool{},
		subjects: []fakeSubject{
			{ID: "10000001", Identifier: "abc123", SourceID: "ldap", Name: "Ada Example",
				Attributes: map[string]string{"email": "ada@example.edu", "description": "Staff"}},
			{ID: "10000002", Identifier: "xyz456", SourceID: "ldap", Name: "Xavier Example",
				Attributes: map[string]string{"email": "xavier@example.edu", "description": "Student"}},
			{ID: "10000003", Identifier: "ext789", SourceID: "ldap", Name: "Eve External"},
		},
	}
	f.addStem("example", "example", "")
	f.addStem("example:apps", "apps", "")
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	return f
}

// providerConfig returns a provider block pointing at the fake.
func (f *fakeGrouper) providerConfig() string {
	return fmt.Sprintf(`
provider "grouper" {
  base_url = %q
  username = %q
  password = %q
}
`, f.server.URL+"/grouper-ws/servicesRest", fakeUser, fakePassword)
}

// --- state helpers used by tests to inspect or change the "server" ---

func (f *fakeGrouper) addStem(name, displayExtension, description string) *fakeObject {
	f.mu.Lock()
	defer f.mu.Unlock()
	o := &fakeObject{UUID: newUUID(), Name: name, DisplayExtension: displayExtension, Description: description}
	f.stems[name] = o
	return o
}

func (f *fakeGrouper) stem(name string) *fakeObject {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stems[name]
}

func (f *fakeGrouper) group(name string) *fakeObject {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.groups[name]
}

func (f *fakeGrouper) deleteStem(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.stems, name)
}

func (f *fakeGrouper) deleteGroup(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.groups, name)
	delete(f.members, name)
}

// memberIDs returns the subject ids of a group's direct members, sorted.
func (f *fakeGrouper) memberIDs(group string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for key := range f.members[group] {
		ids = append(ids, key[strings.Index(key, "/")+1:])
	}
	sort.Strings(ids)
	return ids
}

// addMember adds a direct member as if someone did it outside Terraform.
func (f *fakeGrouper) addMember(group, subject string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.resolveLocked(subject, "")
	if !ok {
		f.t.Fatalf("fake grouper: unknown subject %q", subject)
	}
	f.addMemberLocked(group, s)
}

func (f *fakeGrouper) removeMember(group, subject string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.resolveLocked(subject, ""); ok {
		delete(f.members[group], s.SourceID+"/"+s.ID)
	}
}

func (f *fakeGrouper) addMemberLocked(group string, s fakeSubject) bool {
	if f.members[group] == nil {
		f.members[group] = map[string]bool{}
	}
	key := s.SourceID + "/" + s.ID
	existed := f.members[group][key]
	f.members[group][key] = true
	return existed
}

// resolveLocked finds a person or a group subject by id or identifier. Groups
// are subjects in the "g:gsa" source with their UUID as id and name as
// identifier, as in Grouper.
func (f *fakeGrouper) resolveLocked(value, source string) (fakeSubject, bool) {
	for _, s := range f.subjects {
		if (s.ID == value || s.Identifier == value) && (source == "" || source == s.SourceID) {
			return s, true
		}
	}
	if source == "" || source == "g:gsa" {
		for _, g := range f.groups {
			if g.UUID == value || g.Name == value {
				return fakeSubject{ID: g.UUID, Identifier: g.Name, SourceID: "g:gsa", Name: g.Name}, true
			}
		}
	}
	return fakeSubject{}, false
}

// --- HTTP handling ---

type fakeMeta struct {
	Success       string `json:"success"`
	ResultCode    string `json:"resultCode"`
	ResultMessage string `json:"resultMessage,omitempty"`
}

func ok(code string) fakeMeta   { return fakeMeta{Success: "T", ResultCode: code} }
func fail(code string) fakeMeta { return fakeMeta{Success: "F", ResultCode: code} }

type wireObject struct {
	UUID             string `json:"uuid,omitempty"`
	Name             string `json:"name,omitempty"`
	Extension        string `json:"extension,omitempty"`
	DisplayExtension string `json:"displayExtension,omitempty"`
	DisplayName      string `json:"displayName,omitempty"`
	Description      string `json:"description,omitempty"`
	Enabled          string `json:"enabled,omitempty"`
	TypeOfGroup      string `json:"typeOfGroup,omitempty"`
}

type wireLookup struct {
	StemName  string `json:"stemName"`
	GroupName string `json:"groupName"`
	UUID      string `json:"uuid"`
}

type wireSubjectLookup struct {
	SubjectID         string `json:"subjectId"`
	SubjectIdentifier string `json:"subjectIdentifier"`
	SubjectSourceID   string `json:"subjectSourceId"`
}

type wireSubject struct {
	ID              string   `json:"id,omitempty"`
	Name            string   `json:"name,omitempty"`
	SourceID        string   `json:"sourceId,omitempty"`
	Success         string   `json:"success"`
	ResultCode      string   `json:"resultCode"`
	AttributeValues []string `json:"attributeValues,omitempty"`
}

func (f *fakeGrouper) serve(w http.ResponseWriter, r *http.Request) {
	user, pass, hasAuth := r.BasicAuth()
	if !hasAuth || user != fakeUser || pass != fakePassword {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	prefix := "/grouper-ws/servicesRest/" + fakeVersion + "/"
	if !strings.HasPrefix(r.URL.EscapedPath(), prefix) {
		f.t.Errorf("fake grouper: unexpected path %s", r.URL.EscapedPath())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rest := strings.TrimPrefix(r.URL.EscapedPath(), prefix)

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		f.t.Errorf("fake grouper: undecodable request body for %s %s: %v", r.Method, rest, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	var resp any
	switch {
	case rest == "stems" && r.Method == http.MethodPost:
		resp = f.handleStems(body)
	case rest == "groups" && r.Method == http.MethodPost:
		resp = f.handleGroups(body)
	case rest == "subjects" && r.Method == http.MethodPost:
		resp = f.handleSubjects(body)
	case strings.HasPrefix(rest, "groups/") && strings.HasSuffix(rest, "/members"):
		group, err := url.PathUnescape(strings.TrimSuffix(strings.TrimPrefix(rest, "groups/"), "/members"))
		if err != nil {
			f.t.Errorf("fake grouper: bad group path %q", rest)
		}
		resp = f.handleMembers(r.Method, group, body)
	}
	if resp == nil {
		f.t.Errorf("fake grouper: unsupported request %s %s with keys %v", r.Method, rest, keys(body))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		f.t.Errorf("fake grouper: encode response: %v", err)
	}
}

func (f *fakeGrouper) handleStems(body map[string]json.RawMessage) any {
	switch {
	case body["WsRestFindStemsRequest"] != nil:
		var req struct {
			WsStemLookups []wireLookup `json:"wsStemLookups"`
		}
		f.decode(body["WsRestFindStemsRequest"], &req)
		results := []wireObject{}
		for _, lk := range req.WsStemLookups {
			if o := findObject(f.stems, lk.StemName, lk.UUID); o != nil {
				results = append(results, stemWire(o))
			}
		}
		return map[string]any{"WsFindStemsResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "stemResults": results}}

	case body["WsRestStemSaveRequest"] != nil:
		var req struct {
			WsStemToSaves []struct {
				WsStem       wireObject `json:"wsStem"`
				WsStemLookup wireLookup `json:"wsStemLookup"`
			} `json:"wsStemToSaves"`
		}
		f.decode(body["WsRestStemSaveRequest"], &req)
		var results []any
		for _, item := range req.WsStemToSaves {
			o, code := saveObject(f.stems, item.WsStem, item.WsStemLookup.StemName)
			results = append(results, map[string]any{"resultMetadata": ok(code), "wsStem": stemWire(o)})
		}
		return map[string]any{"WsStemSaveResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "results": results}}

	case body["WsRestStemDeleteRequest"] != nil:
		var req struct {
			WsStemLookups []wireLookup `json:"wsStemLookups"`
		}
		f.decode(body["WsRestStemDeleteRequest"], &req)
		var results []any
		for _, lk := range req.WsStemLookups {
			o := findObject(f.stems, lk.StemName, lk.UUID)
			switch {
			case o == nil:
				results = append(results, map[string]any{"resultMetadata": ok("SUCCESS_STEM_NOT_FOUND")})
			case f.hasChildrenLocked(o.Name):
				results = append(results, map[string]any{"resultMetadata": fail("PROBLEM_DELETING_STEMS")})
			default:
				delete(f.stems, o.Name)
				results = append(results, map[string]any{"resultMetadata": ok("SUCCESS")})
			}
		}
		return map[string]any{"WsStemDeleteResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "results": results}}
	}
	return nil
}

func (f *fakeGrouper) handleGroups(body map[string]json.RawMessage) any {
	switch {
	case body["WsRestFindGroupsRequest"] != nil:
		var req struct {
			WsGroupLookups []wireLookup `json:"wsGroupLookups"`
		}
		f.decode(body["WsRestFindGroupsRequest"], &req)
		results := []wireObject{}
		for _, lk := range req.WsGroupLookups {
			if o := findObject(f.groups, lk.GroupName, lk.UUID); o != nil {
				results = append(results, groupWire(o))
			}
		}
		return map[string]any{"WsFindGroupsResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "groupResults": results}}

	case body["WsRestGroupSaveRequest"] != nil:
		var req struct {
			WsGroupToSaves []struct {
				WsGroup       wireObject `json:"wsGroup"`
				WsGroupLookup wireLookup `json:"wsGroupLookup"`
			} `json:"wsGroupToSaves"`
		}
		f.decode(body["WsRestGroupSaveRequest"], &req)
		var results []any
		for _, item := range req.WsGroupToSaves {
			o, code := saveObject(f.groups, item.WsGroup, item.WsGroupLookup.GroupName)
			results = append(results, map[string]any{"resultMetadata": ok(code), "wsGroup": groupWire(o)})
		}
		return map[string]any{"WsGroupSaveResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "results": results}}

	case body["WsRestGroupDeleteRequest"] != nil:
		var req struct {
			WsGroupLookups []wireLookup `json:"wsGroupLookups"`
		}
		f.decode(body["WsRestGroupDeleteRequest"], &req)
		var results []any
		for _, lk := range req.WsGroupLookups {
			if o := findObject(f.groups, lk.GroupName, lk.UUID); o == nil {
				results = append(results, map[string]any{"resultMetadata": ok("SUCCESS_GROUP_NOT_FOUND")})
			} else {
				delete(f.groups, o.Name)
				delete(f.members, o.Name)
				results = append(results, map[string]any{"resultMetadata": ok("SUCCESS")})
			}
		}
		return map[string]any{"WsGroupDeleteResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "results": results}}
	}
	return nil
}

func (f *fakeGrouper) handleMembers(method, group string, body map[string]json.RawMessage) any {
	if f.groups[group] == nil {
		envelope := map[string]any{"resultMetadata": fail("GROUP_NOT_FOUND")}
		switch {
		case body["WsRestAddMemberRequest"] != nil:
			return map[string]any{"WsAddMemberResults": envelope}
		case body["WsRestHasMemberRequest"] != nil:
			return map[string]any{"WsHasMemberResults": envelope}
		case body["WsRestDeleteMemberRequest"] != nil:
			return map[string]any{"WsDeleteMemberResults": envelope}
		}
		return nil
	}

	switch {
	case method == http.MethodPut && body["WsRestAddMemberRequest"] != nil:
		var req struct {
			SubjectLookups     []wireSubjectLookup `json:"subjectLookups"`
			ReplaceAllExisting string              `json:"replaceAllExisting"`
		}
		f.decode(body["WsRestAddMemberRequest"], &req)
		if req.ReplaceAllExisting != "F" {
			f.t.Errorf("fake grouper: addMember sent replaceAllExisting=%q; it must always be \"F\"", req.ReplaceAllExisting)
		}
		var results []any
		for _, lk := range req.SubjectLookups {
			s, found := f.resolveLookup(lk)
			if !found {
				results = append(results, subjectNotFound())
				continue
			}
			code := "SUCCESS"
			if f.addMemberLocked(group, s) {
				code = "SUCCESS_ALREADY_EXISTED"
			}
			results = append(results, map[string]any{"resultMetadata": ok(code), "wsSubject": subjectWire(s, nil)})
		}
		return map[string]any{"WsAddMemberResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "results": results}}

	case method == http.MethodPost && body["WsRestHasMemberRequest"] != nil:
		var req struct {
			SubjectLookups []wireSubjectLookup `json:"subjectLookups"`
			MemberFilter   string              `json:"memberFilter"`
		}
		f.decode(body["WsRestHasMemberRequest"], &req)
		if req.MemberFilter != "Immediate" {
			f.t.Errorf("fake grouper: hasMember sent memberFilter=%q; reads must use \"Immediate\"", req.MemberFilter)
		}
		var results []any
		for _, lk := range req.SubjectLookups {
			s, found := f.resolveLookup(lk)
			if !found {
				results = append(results, subjectNotFound())
				continue
			}
			code := "IS_NOT_MEMBER"
			if f.members[group][s.SourceID+"/"+s.ID] {
				code = "IS_MEMBER"
			}
			results = append(results, map[string]any{"resultMetadata": ok(code), "wsSubject": subjectWire(s, nil)})
		}
		return map[string]any{"WsHasMemberResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "results": results}}

	case method == http.MethodPost && body["WsRestDeleteMemberRequest"] != nil:
		var req struct {
			SubjectLookups []wireSubjectLookup `json:"subjectLookups"`
		}
		f.decode(body["WsRestDeleteMemberRequest"], &req)
		var results []any
		for _, lk := range req.SubjectLookups {
			s, found := f.resolveLookup(lk)
			if !found {
				results = append(results, subjectNotFound())
				continue
			}
			delete(f.members[group], s.SourceID+"/"+s.ID)
			results = append(results, map[string]any{"resultMetadata": ok("SUCCESS"), "wsSubject": subjectWire(s, nil)})
		}
		return map[string]any{"WsDeleteMemberResults": map[string]any{
			"resultMetadata": ok("SUCCESS"), "results": results}}
	}
	return nil
}

func (f *fakeGrouper) handleSubjects(body map[string]json.RawMessage) any {
	if body["WsRestGetSubjectsRequest"] == nil {
		return nil
	}
	var req struct {
		WsSubjectLookups      []wireSubjectLookup `json:"wsSubjectLookups"`
		SubjectAttributeNames []string            `json:"subjectAttributeNames"`
	}
	f.decode(body["WsRestGetSubjectsRequest"], &req)
	subjects := []wireSubject{}
	for _, lk := range req.WsSubjectLookups {
		if s, found := f.resolveLookup(lk); found {
			subjects = append(subjects, subjectWire(s, req.SubjectAttributeNames))
		} else {
			subjects = append(subjects, wireSubject{Success: "F", ResultCode: "SUBJECT_NOT_FOUND"})
		}
	}
	return map[string]any{"WsGetSubjectsResults": map[string]any{
		"resultMetadata":        ok("SUCCESS"),
		"subjectAttributeNames": req.SubjectAttributeNames,
		"wsSubjects":            subjects,
	}}
}

func (f *fakeGrouper) resolveLookup(lk wireSubjectLookup) (fakeSubject, bool) {
	if s, ok := f.resolveLocked(lk.SubjectID, lk.SubjectSourceID); ok {
		return s, true
	}
	return f.resolveLocked(lk.SubjectIdentifier, lk.SubjectSourceID)
}

func (f *fakeGrouper) hasChildrenLocked(stem string) bool {
	for name := range f.stems {
		if strings.HasPrefix(name, stem+":") {
			return true
		}
	}
	for name := range f.groups {
		if strings.HasPrefix(name, stem+":") {
			return true
		}
	}
	return false
}

func (f *fakeGrouper) decode(raw json.RawMessage, v any) {
	if err := json.Unmarshal(raw, v); err != nil {
		f.t.Errorf("fake grouper: decode request: %v", err)
	}
}

// saveObject creates or updates a stem or group. Fields the client omits keep
// their current value, and a new object's display label defaults to its
// extension, as in Grouper.
func saveObject(store map[string]*fakeObject, in wireObject, lookupName string) (*fakeObject, string) {
	name := lookupName
	if name == "" {
		name = in.Name
	}
	o, exists := store[name]
	if !exists {
		o = &fakeObject{UUID: newUUID(), Name: name, DisplayExtension: lastComponent(name)}
		store[name] = o
	}
	before := *o
	if in.DisplayExtension != "" {
		o.DisplayExtension = in.DisplayExtension
	}
	if in.Description != "" {
		o.Description = in.Description
	}
	switch {
	case !exists:
		return o, "SUCCESS_INSERTED"
	case before == *o:
		return o, "SUCCESS_NO_CHANGES_NEEDED"
	default:
		return o, "SUCCESS_UPDATED"
	}
}

func findObject(store map[string]*fakeObject, name, uuid string) *fakeObject {
	if name != "" {
		return store[name]
	}
	for _, o := range store {
		if o.UUID == uuid {
			return o
		}
	}
	return nil
}

func stemWire(o *fakeObject) wireObject {
	return wireObject{
		UUID: o.UUID, Name: o.Name, Extension: lastComponent(o.Name),
		DisplayExtension: o.DisplayExtension, DisplayName: o.Name, Description: o.Description,
	}
}

func groupWire(o *fakeObject) wireObject {
	w := stemWire(o)
	w.Enabled = "T"
	w.TypeOfGroup = "group"
	return w
}

func subjectWire(s fakeSubject, attributeNames []string) wireSubject {
	w := wireSubject{ID: s.ID, Name: s.Name, SourceID: s.SourceID, Success: "T", ResultCode: "SUCCESS"}
	for _, n := range attributeNames {
		w.AttributeValues = append(w.AttributeValues, s.Attributes[n])
	}
	return w
}

func subjectNotFound() map[string]any {
	return map[string]any{
		"resultMetadata": fail("SUBJECT_NOT_FOUND"),
		"wsSubject":      wireSubject{Success: "F", ResultCode: "SUBJECT_NOT_FOUND"},
	}
}

func lastComponent(name string) string {
	return name[strings.LastIndex(name, ":")+1:]
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func keys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
