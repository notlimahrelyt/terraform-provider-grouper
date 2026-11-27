# Grouper Web Services API Observations

This document records **verified** facts about the Grouper Web Services (WS)
REST API, as observed against a live Grouper 4.16.0 deployment. The provider client is implemented strictly against what is
recorded here — no endpoint, request body, or response field is coded until it
is confirmed and captured below.

> **Security:** Never commit credentials, real subject identifiers, or
> production data. All examples below use the sanitized placeholder data that
> ships in the upstream Grouper source samples (`aStem`, `GrouperSystem`,
> `test.subject.0`, source `jdbc`, fake UUIDs). Real captures belong in the
> gitignored `docs/api-captures/` directory.

## Sources

- Live reads and writes against a Grouper 4.16.0 deployment (HTTP Basic auth).
- Internet2 Grouper wiki, [Grouper Web Services](https://spaces.at.internet2.edu/display/Grouper/Grouper+Web+Services).
- Upstream request/response samples in the Grouper source tree:
  `grouper-ws/grouper-ws/doc/samples/*` (addMember, deleteMember, hasMember,
  findGroups, groupSave, groupDelete, findStems, stemSave, stemDelete,
  getSubjects).

## Reference deployment facts

| Fact | Value | Confirmed? |
| ---- | ----- | ---------- |
| Grouper server version | **4.16.0** (from `responseMetadata.serverVersion`) | ✅ |
| Base REST URL shape | `https://<host>/grouper-ws/servicesRest` | ✅ |
| A URL version segment is **required** | yes — version-less requests fail with `EXCEPTION` "Not expecting a blank string for version" | ✅ |
| Which version segment to use | **`v2_6_000`** — confirmed accepted (clientVersion 2.6.0) against the 4.16.0 server via live read | ✅ |
| Authentication method (service account) | **HTTP Basic** — confirmed working with the service account on a live read | ✅ |
| Response format selector | JSON returned with `Content-Type: application/json`; no `/json/` path segment needed | ✅ |
| Subjects require a `sourceId`? | no — optional in `getSubjects` (`subjectSourceId`); source ids are deployment-specific (upstream samples use `jdbc`) | ✅ |
| Grouper UUIDs returned & stable? | ✅ stems and groups return a stable `uuid`; use as the Terraform resource ID. Lookup by uuid supported via `WsStemLookup.uuid` / `FIND_BY_STEM_UUID` | ✅ |
| Service account can CRUD test objects? | ✅ confirmed — create (`SUCCESS_INSERTED`) and delete (`SUCCESS`, "was deleted") of stems under a test stem | ✅ |
| Subject lookup key | a login name (e.g. a NetID) is often a **subjectIdentifier**, not the subjectId; a dual-match lookup resolves either (see below) | ✅ |

## Response envelope conventions (CONFIRMED)

Every operation returns a top-level object keyed by an operation-specific
results name (e.g. `WsStemSaveResults`, `WsFindGroupsResults`). Each carries:

```jsonc
"resultMetadata":   { "success": "T"|"F", "resultCode": "<STRING>", "resultMessage": "<STRING>" },
"responseMetadata": { "millis": "<STRING>", "serverVersion": "<STRING>" }
```

Batch operations (`save`, `delete`, `addMember`, `deleteMember`, `hasMember`)
also return a `results[]` array, each element with its **own** nested
`resultMetadata` plus the affected object (`wsStem` / `wsGroup` / `wsSubject`).

Errors that never reach an operation handler use a different top-level key:

```jsonc
{ "WsRestResultProblem": { "resultMetadata": { "resultCode":"EXCEPTION", "success":"F", ... },
                           "responseMetadata": { "serverVersion":"4.16.0", ... } } }
```

**Application-level failures can be returned with HTTP 200.** The client must
inspect `resultMetadata.success` / `resultCode`, not the HTTP status alone.

### Result codes observed

| Context | resultCode | Meaning | Provider mapping |
| ------- | ---------- | ------- | ---------------- |
| servlet, no version | `EXCEPTION` (success `F`) | request never dispatched | unexpected error |
| save (stem/group) | `SUCCESS_INSERTED` (created) / `SUCCESS_NO_CHANGES_NEEDED` / `SUCCESS_UPDATED` (success `T`) | created or already current | success |
| addMember (per subject) | `SUCCESS` / `SUCCESS_ALREADY_EXISTED` (success `T`) | added or already a member | success (idempotent create) |
| deleteMember (per subject) | `SUCCESS` (success `T`) | removed (also when absent) | success |
| group/stem delete (per item) | `SUCCESS_GROUP_NOT_FOUND` / `SUCCESS_STEM_NOT_FOUND` (success `T`) | delete target already gone | treat as success |
| hasMember (per subject) | `IS_MEMBER` / `IS_NOT_MEMBER` (success `T`) | membership test result | drives Read |

> **Not-found is usually application-level, not HTTP 404.** `find*` returns an
> empty results array; `delete` returns a `*_NOT_FOUND` success code. The client
> must derive not-found from the body, not rely on status codes.

## Object field shapes (CONFIRMED)

**Stem** (`wsStem`): `name`, `extension`, `displayExtension`, `displayName`,
`description`, `idIndex`, `uuid`.

**Group** (`wsGroup`): `name`, `extension`, `displayExtension`, `displayName`,
`description`, `enabled`, `idIndex`, `typeOfGroup`, `uuid`.

**Subject** (`wsSubject`): `id`, `name`, `sourceId`, `resultCode`, `success`,
and (in `getSubjects`) `attributeValues[]` positionally matching the request's
`subjectAttributeNames[]`.

Note: `name` is the full colon-delimited path (`aStem:aGroup`); `extension` is
the final path component; `displayExtension` is the human label for that
component; `displayName` is the full display path.

## Operation captures

All paths are relative to `.../servicesRest/<version>`. Colons in a group name
embedded in a path are URL-encoded (`:` → `%3A`).

### Stems (folders)

**Lookup / find** — `POST /stems`
```jsonc
{"WsRestFindStemsRequest":{"wsStemQueryFilter":{
  "stemQueryFilterType":"FIND_BY_STEM_NAME_APPROXIMATE","stemName":"aSte"}}}
// FIND_BY_STEM_NAME (exact) is the filter to use for a single known stem.
// -> WsFindStemsResults.stemResults[] (empty array => not found)
```

**Create / update (save)** — `POST /stems`
```jsonc
{"WsRestStemSaveRequest":{"wsStemToSaves":[
  {"wsStem":{"name":"aStem:whateverStem","extension":"whateverStem",
             "displayExtension":"disp1","description":"desc1"},
   "wsStemLookup":{"stemName":"aStem:whateverStem"}}]}}
// -> WsStemSaveResults.results[].wsStem (with uuid, idIndex)
```

**Delete** — `POST /stems`
```jsonc
{"WsRestStemDeleteRequest":{"wsStemLookups":[{"stemName":"aStem:whateverStem"}]}}
// -> WsStemDeleteResults.results[].resultMetadata.resultCode
//    SUCCESS or SUCCESS_STEM_NOT_FOUND (both success:T)
```

### Groups

**Lookup / find** — `POST /groups`
```jsonc
{"WsRestFindGroupsRequest":{"wsGroupLookups":[{"groupName":"aStem:aGroup"}]}}
// or by uuid: {"wsGroupLookups":[{"uuid":"<uuid>"}]}
// -> WsFindGroupsResults.groupResults[] (empty array => not found)
```
> **Important (verified):** use `wsGroupLookups` (direct lookup), NOT the
> `wsQueryFilter` `FIND_BY_GROUP_*` form, for read-after-write. During a
> post-write refresh the query-filter path returned **stale** group metadata
> (old displayExtension/description) while the direct lookup returned current
> data. Stems already use the direct `wsStemLookups` form and were unaffected.
> A `wsQueryFilter` with `FIND_BY_GROUP_NAME_APPROXIMATE` is still the right tool
> for fuzzy search, just not for authoritative reads.

**Create / update (save)** — `POST /groups`
```jsonc
{"WsRestGroupSaveRequest":{"wsGroupToSaves":[
  {"wsGroup":{"name":"aStem:whateverGroup","extension":"whateverGroup",
              "displayExtension":"disp1","description":"desc1"},
   "wsGroupLookup":{"groupName":"aStem:whateverGroup"}}]}}
// -> WsGroupSaveResults.results[].wsGroup (with uuid, idIndex, enabled, typeOfGroup)
```

**Delete** — `POST /groups`
```jsonc
{"WsRestGroupDeleteRequest":{"wsGroupLookups":[{"groupName":"aStem:whatever"}]}}
// -> WsGroupDeleteResults.results[].resultMetadata.resultCode
//    SUCCESS or SUCCESS_GROUP_NOT_FOUND (both success:T)
```

### Memberships (direct)

**Add** — `PUT /groups/<groupName%3Aencoded>/members`
```jsonc
{"WsRestAddMemberRequest":{
  "subjectLookups":[{"subjectId":"10021368"}],
  "replaceAllExisting":"F"}}   // MUST be "F" — never replace the whole membership list
// -> WsAddMemberResults.results[].resultMetadata.resultCode
//    SUCCESS or SUCCESS_ALREADY_EXISTED (both success:T) => idempotent create
```

**Read (membership test)** — `POST /groups/<groupName%3Aencoded>/members`
```jsonc
{"WsRestHasMemberRequest":{"subjectLookups":[{"subjectId":"10021368"}]}}
// -> WsHasMemberResults.results[].resultMetadata.resultCode = IS_MEMBER | IS_NOT_MEMBER
// NOTE: to test DIRECT membership only, pass "memberFilter":"Immediate"
//       (default is effective/all). Confirmed; see below.
```

**Delete** — `POST /groups/<groupName%3Aencoded>/members`
```jsonc
{"WsRestDeleteMemberRequest":{"subjectLookups":[{"subjectId":"10021368"}]}}
// -> WsDeleteMemberResults.results[].resultMetadata.resultCode = SUCCESS (also when absent)
// Deletes ONLY the listed subjects; unrelated members are untouched.
```

### Subjects

**Lookup** — `POST /subjects`
```jsonc
{"WsRestGetSubjectsRequest":{
  "wsSubjectLookups":[{"subjectId":"test.subject.0"},
                      {"subjectSourceId":"jdbc","subjectId":"test.subject.1"}],
  "subjectAttributeNames":["description","loginid","name"]}}
// -> WsGetSubjectsResults.wsSubjects[] with id, name, sourceId, resultCode,
//    success, attributeValues[] (positional to subjectAttributeNames[]).
// Only request non-sensitive attributes; do not store arbitrary PII in state.
```

## Design decisions driven by these observations

1. **Terraform IDs** = Grouper `uuid` for stems and groups (stable, returned on
   save). Membership has no single UUID here → use a deterministic composite
   `<group-name>/<source-id>/<subject-id>` (only the first two `/` are
   separators, so a subject id may itself contain `/`).
2. **Idempotent membership create** is native: `SUCCESS_ALREADY_EXISTED` is a
   success, and `replaceAllExisting:"F"` guarantees we never remove members we
   do not manage.
3. **Not-found detection is body-driven** (empty `*Results[]`, `*_NOT_FOUND`
   codes), handled per-operation in the client — HTTP 404 is a secondary signal.
4. **Version segment is deployment-specific** → expose a `ws_version` provider
   attribute (isolating version behavior in the client), defaulting to
   `v2_6_000`, the segment confirmed against Grouper 4.16.
5. **A shared envelope decoder** (`resultMetadata`/`responseMetadata` +
   `WsRestResultProblem`) belongs in the client and is reused by every method.

## Confirmed behaviors

- [x] Exact `ws_version` segment — `v2_6_000`.
- [x] Service-account authentication method — HTTP Basic (dedicated service account).
- [x] JSON selected by `Content-Type`/`Accept` headers; no `/json/` segment needed.
- [x] Service account CREATE/UPDATE/DELETE privileges in the test stem.
- [x] **Subject lookup by login name:** at many institutions a login name (e.g.
      a NetID) is the subject **identifier**, not the subjectId (which may be
      numeric, e.g. `12345678`). A dual-match lookup
      (`subjectId` and `subjectIdentifier` set to the same value) resolves it —
      `success:"T"`, and the response echoes `identifierLookup`. The provider
      uses the dual-match by
      default and auto-populates `subject_source_id` from the resolved value.
- [x] `memberFilter=Immediate` returns direct members only (WsMemberFilter:
      All/Effective/Immediate/Composite/NonImmediate; default All). Used for
      authoritative direct-membership reads.
