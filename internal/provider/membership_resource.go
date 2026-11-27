package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/notlimahrelyt/terraform-provider-grouper/internal/client"
)

var (
	_ resource.Resource                = (*membershipResource)(nil)
	_ resource.ResourceWithConfigure   = (*membershipResource)(nil)
	_ resource.ResourceWithImportState = (*membershipResource)(nil)
)

// membershipIDSep separates the components of the deterministic membership ID:
// <group_name>/<subject_source_id>/<subject_id>. Grouper names use colons (never
// slashes) and source ids contain no slashes, so the first two "/" are
// unambiguous; everything after the second "/" is the subject id, which may
// itself contain "/" at some institutions. The source component may be empty.
const membershipIDSep = "/"

// NewMembershipResource constructs the grouper_membership resource.
func NewMembershipResource() resource.Resource {
	return &membershipResource{}
}

type membershipResource struct {
	client *client.Client
}

type membershipResourceModel struct {
	ID              types.String `tfsdk:"id"`
	GroupName       types.String `tfsdk:"group_name"`
	SubjectID       types.String `tfsdk:"subject_id"`
	SubjectSourceID types.String `tfsdk:"subject_source_id"`
	MembershipType  types.String `tfsdk:"membership_type"`
}

func (r *membershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_membership"
}

func (r *membershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single direct (immediate) membership of a subject in a Grouper group. " +
			"Only the one membership represented by this resource is managed; other members of the group " +
			"are never added or removed. All identity attributes are immutable (changing any forces replacement).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Deterministic identifier: `<group_name>/<subject_source_id>/<subject_id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Full colon-delimited name of the group. Changing this forces replacement.",
				Validators:          grouperPathValidators(),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"subject_id": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Identifier of the subject to add. May be the subject's id or a login " +
					"identifier such as a username or NetID (Grouper resolves either). Changing this forces replacement.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"subject_source_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Subject source id (e.g. the LDAP source). Optional; supply it to disambiguate " +
					"when the same id exists in multiple sources. Populated with the resolved source after apply. " +
					"Changing an explicitly-set value forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"membership_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Membership type managed by this resource. Always `immediate` (direct).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *membershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *membershipResource) lookup(m membershipResourceModel) client.SubjectLookup {
	return client.SubjectLookup{
		SubjectID:       m.SubjectID.ValueString(),
		SubjectSourceID: m.SubjectSourceID.ValueString(),
	}
}

func (r *membershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan membershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := plan.GroupName.ValueString()
	subject, err := r.client.AddMembership(ctx, group, r.lookup(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to add membership of %q to group %q", plan.SubjectID.ValueString(), group),
			membershipErrorDetail("add", plan.SubjectID.ValueString(), group, err))
		return
	}

	applyMembershipToState(subject, &plan)
	tflog.Debug(ctx, "added grouper membership", map[string]any{"group": group, "id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state membershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := state.GroupName.ValueString()
	isMember, subject, err := r.client.HasMembership(ctx, group, r.lookup(state))
	if err != nil {
		if client.IsNotFound(err) {
			// The subject no longer resolves; the membership cannot exist.
			tflog.Warn(ctx, "grouper membership subject not found on read; removing from state",
				map[string]any{"group": group, "subject_id": state.SubjectID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read membership of %q in group %q", state.SubjectID.ValueString(), group),
			membershipErrorDetail("read", state.SubjectID.ValueString(), group, err))
		return
	}
	if !isMember {
		// The direct membership was removed outside Terraform.
		tflog.Warn(ctx, "grouper direct membership no longer present; removing from state",
			map[string]any{"group": group, "subject_id": state.SubjectID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	applyMembershipToState(subject, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked: every configurable
// attribute forces replacement. It re-persists the plan defensively.
func (r *membershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan membershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state membershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := state.GroupName.ValueString()
	if err := r.client.DeleteMembership(ctx, group, r.lookup(state)); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete membership of %q from group %q", state.SubjectID.ValueString(), group),
			membershipErrorDetail("delete", state.SubjectID.ValueString(), group, err))
		return
	}
	tflog.Debug(ctx, "deleted grouper membership", map[string]any{"group": group, "subject_id": state.SubjectID.ValueString()})
}

// ImportState parses "<group_name>/<subject_source_id>/<subject_id>". The source
// component may be empty (e.g. "example:a:b//abc123"). Read then verifies the
// membership still exists.
func (r *membershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	group, source, subject, ok := parseMembershipID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected \"<group_name>/<subject_source_id>/<subject_id>\" (source may be empty), got: "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), group)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subject_id"), subject)...)
	if source == "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subject_source_id"), types.StringNull())...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subject_source_id"), source)...)
	}
}

// applyMembershipToState fills computed/normalized fields from a resolved
// subject. subject_id is preserved as the user supplied it; subject_source_id is
// set to the subject's actual source.
func applyMembershipToState(subject *client.Subject, m *membershipResourceModel) {
	source := m.SubjectSourceID.ValueString()
	if subject != nil && subject.SourceID != "" {
		source = subject.SourceID
	}
	m.SubjectSourceID = stringOrNull(source)
	m.MembershipType = types.StringValue("immediate")
	m.ID = types.StringValue(buildMembershipID(m.GroupName.ValueString(), source, m.SubjectID.ValueString()))
}

func buildMembershipID(group, source, subject string) string {
	return group + membershipIDSep + source + membershipIDSep + subject
}

// parseMembershipID splits a membership import ID into its three components.
// Only the first two separators are significant, so a subject id containing "/"
// round-trips intact.
func parseMembershipID(id string) (group, source, subject string, ok bool) {
	parts := strings.SplitN(id, membershipIDSep, 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	if parts[0] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func membershipErrorDetail(action, subject, group string, err error) string {
	switch {
	case client.IsNotFound(err):
		return fmt.Sprintf("Grouper could not resolve subject %q while trying to %s the membership in group %q. "+
			"Confirm the subject id/identifier and (if needed) set subject_source_id.", subject, action, group)
	case client.IsForbidden(err):
		return fmt.Sprintf("Grouper returned a forbidden result while trying to %s membership in group %q. "+
			"Confirm the provider service account has UPDATE privileges on the group.", action, group)
	case client.IsUnauthorized(err):
		return fmt.Sprintf("Grouper authentication failed while trying to %s membership in group %q. "+
			"Check the provider username/password.", action, group)
	default:
		return fmt.Sprintf("Unexpected error while trying to %s membership of %q in group %q: %s",
			action, subject, group, err.Error())
	}
}
