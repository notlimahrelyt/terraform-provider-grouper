package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/notlimahrelyt/terraform-provider-grouper/internal/client"
)

// Interface assertions.
var (
	_ resource.Resource                = (*stemResource)(nil)
	_ resource.ResourceWithConfigure   = (*stemResource)(nil)
	_ resource.ResourceWithImportState = (*stemResource)(nil)
)

// NewStemResource constructs the grouper_stem resource.
func NewStemResource() resource.Resource {
	return &stemResource{}
}

type stemResource struct {
	client *client.Client
}

// stemResourceModel is the Terraform state model for a stem. It is distinct from
// the client DTO so API changes stay isolated in the client package.
type stemResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
	ParentStemName types.String `tfsdk:"parent_stem_name"`
}

func (r *stemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stem"
}

func (r *stemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grouper folder (stem). The parent stem must already exist; " +
			"this resource does not create missing parents.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Grouper UUID of the stem. Used as the stable Terraform identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Grouper UUID of the stem. Used as the stable Terraform identifier.",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Full colon-delimited system name of the stem, e.g. " +
					"\"example:apps:myteam\". Changing this forces replacement.",
				Validators:    grouperPathValidators(),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Human-readable display label for this stem component " +
					"(Grouper displayExtension). Defaults to the last path component if unset.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the stem.",
			},
			"parent_stem_name": schema.StringAttribute{
				Computed:      true,
				Description:   "Full name of the parent stem, derived from name. Empty for a top-level stem.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *stemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *stemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stem, err := r.client.SaveStem(ctx, client.SaveStemRequest{
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to create Grouper stem %q", plan.Name.ValueString()),
			stemErrorDetail("create", plan.Name.ValueString(), err))
		return
	}

	applyStemToState(stem, &plan)
	tflog.Debug(ctx, "created grouper stem", map[string]any{"uuid": stem.UUID, "name": stem.Name})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *stemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stem, err := r.client.GetStem(ctx, client.StemLookup{UUID: state.ID.ValueString()})
	if err != nil {
		if client.IsNotFound(err) {
			// The stem was deleted outside Terraform; drop it from state.
			tflog.Warn(ctx, "grouper stem not found on read; removing from state",
				map[string]any{"uuid": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Grouper stem %q", state.Name.ValueString()),
			stemErrorDetail("read", state.Name.ValueString(), err))
		return
	}

	applyStemToState(stem, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *stemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stem, err := r.client.SaveStem(ctx, client.SaveStemRequest{
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Grouper stem %q", plan.Name.ValueString()),
			stemErrorDetail("update", plan.Name.ValueString(), err))
		return
	}

	applyStemToState(stem, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *stemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete by full name. An already-absent stem is treated as success by the
	// client. A non-empty stem returns an error, which is surfaced.
	if err := r.client.DeleteStem(ctx, client.StemLookup{Name: state.Name.ValueString()}); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Grouper stem %q", state.Name.ValueString()),
			stemErrorDetail("delete", state.Name.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted grouper stem", map[string]any{"name": state.Name.ValueString()})
}

// ImportState supports two formats:
//
//	<uuid>            import by Grouper UUID
//	name:<full-name>  import by full stem name (resolved to its UUID)
func (r *stemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if name, ok := strings.CutPrefix(id, "name:"); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected \"name:<full-stem-name>\" but the name was empty.")
			return
		}
		stem, err := r.client.GetStem(ctx, client.StemLookup{Name: name})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to import Grouper stem %q", name),
				stemErrorDetail("import", name, err))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), stem.UUID)...)
		return
	}
	// Treat the bare ID as a UUID; Read populates the remaining fields.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// applyStemToState maps a client Stem onto the Terraform model, normalizing
// server-side values and deriving computed fields.
func applyStemToState(stem *client.Stem, m *stemResourceModel) {
	m.ID = types.StringValue(stem.UUID)
	m.Name = types.StringValue(stem.Name)
	m.DisplayName = types.StringValue(stem.DisplayExtension)
	m.Description = stringOrNull(stem.Description)
	m.ParentStemName = types.StringValue(parentStemName(stem.Name))
}

// stringOrNull maps an empty string to a null value so that an unset optional
// attribute and a server-returned empty string do not produce a permanent diff.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// parentStemName returns everything before the final colon in a Grouper path,
// or an empty string for a top-level stem.
func parentStemName(name string) string {
	if i := strings.LastIndex(name, ":"); i >= 0 {
		return name[:i]
	}
	return ""
}

// stemErrorDetail builds a sanitized, actionable diagnostic detail string.
func stemErrorDetail(action, name string, err error) string {
	switch {
	case client.IsForbidden(err):
		return fmt.Sprintf("Grouper returned a forbidden result while trying to %s stem %q. "+
			"Confirm the provider service account has the required privileges on the parent stem.", action, name)
	case client.IsUnauthorized(err):
		return fmt.Sprintf("Grouper authentication failed while trying to %s stem %q. "+
			"Check the provider username/password.", action, name)
	case client.IsNotFound(err):
		return fmt.Sprintf("Grouper reported the stem %q was not found while trying to %s it.", name, action)
	default:
		return fmt.Sprintf("Unexpected error while trying to %s stem %q: %s", action, name, err.Error())
	}
}
