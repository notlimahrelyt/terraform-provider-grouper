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

var (
	_ resource.Resource                = (*groupResource)(nil)
	_ resource.ResourceWithConfigure   = (*groupResource)(nil)
	_ resource.ResourceWithImportState = (*groupResource)(nil)
)

// NewGroupResource constructs the grouper_group resource.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client *client.Client
}

type groupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	StemName    types.String `tfsdk:"stem_name"`
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Grouper group. The parent stem must already exist; " +
			"this resource does not create it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Grouper UUID of the group. Used as the stable Terraform identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Full colon-delimited system name of the group, e.g. " +
					"`example:apps:myteam:aws-admins`. Changing this forces replacement.",
				Validators:    grouperPathValidators(),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Human-readable display label for this group component " +
					"(Grouper `displayExtension`). Defaults to the last path component if unset.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the group.",
			},
			"stem_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Full name of the parent stem, derived from `name`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.SaveGroup(ctx, client.SaveGroupRequest{
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to create Grouper group %q", plan.Name.ValueString()),
			groupErrorDetail("create", plan.Name.ValueString(), err))
		return
	}

	applyGroupToState(group, &plan)
	tflog.Debug(ctx, "created grouper group", map[string]any{"uuid": group.UUID, "name": group.Name})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.GetGroup(ctx, client.GroupLookup{UUID: state.ID.ValueString()})
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "grouper group not found on read; removing from state",
				map[string]any{"uuid": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Grouper group %q", state.Name.ValueString()),
			groupErrorDetail("read", state.Name.ValueString(), err))
		return
	}

	applyGroupToState(group, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.SaveGroup(ctx, client.SaveGroupRequest{
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Grouper group %q", plan.Name.ValueString()),
			groupErrorDetail("update", plan.Name.ValueString(), err))
		return
	}

	applyGroupToState(group, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGroup(ctx, client.GroupLookup{Name: state.Name.ValueString()}); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Grouper group %q", state.Name.ValueString()),
			groupErrorDetail("delete", state.Name.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted grouper group", map[string]any{"name": state.Name.ValueString()})
}

// ImportState supports two formats:
//
//	<uuid>            import by Grouper UUID
//	name:<full-name>  import by full group name (resolved to its UUID)
func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if name, ok := strings.CutPrefix(id, "name:"); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected \"name:<full-group-name>\" but the name was empty.")
			return
		}
		group, err := r.client.GetGroup(ctx, client.GroupLookup{Name: name})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to import Grouper group %q", name),
				groupErrorDetail("import", name, err))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), group.UUID)...)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func applyGroupToState(group *client.Group, m *groupResourceModel) {
	m.ID = types.StringValue(group.UUID)
	m.Name = types.StringValue(group.Name)
	m.DisplayName = types.StringValue(group.DisplayExtension)
	m.Description = stringOrNull(group.Description)
	m.StemName = types.StringValue(parentStemName(group.Name))
}

func groupErrorDetail(action, name string, err error) string {
	switch {
	case client.IsForbidden(err):
		return fmt.Sprintf("Grouper returned a forbidden result while trying to %s group %q. "+
			"Confirm the provider service account has the required privileges on the parent stem.", action, name)
	case client.IsUnauthorized(err):
		return fmt.Sprintf("Grouper authentication failed while trying to %s group %q. "+
			"Check the provider username/password.", action, name)
	case client.IsNotFound(err):
		return fmt.Sprintf("Grouper reported the group %q was not found while trying to %s it.", name, action)
	default:
		return fmt.Sprintf("Unexpected error while trying to %s group %q: %s", action, name, err.Error())
	}
}
