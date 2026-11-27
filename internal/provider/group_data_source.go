package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/notlimahrelyt/terraform-provider-grouper/internal/client"
)

var (
	_ datasource.DataSource                     = (*groupDataSource)(nil)
	_ datasource.DataSourceWithConfigure        = (*groupDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*groupDataSource)(nil)
)

// NewGroupDataSource constructs the grouper_group data source.
func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	client *client.Client
}

type groupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	StemName    types.String `tfsdk:"stem_name"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	TypeOfGroup types.String `tfsdk:"type_of_group"`
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing Grouper group by exactly one of `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Grouper UUID of the group. Provide exactly one of `id` or `name`.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Full colon-delimited name of the group. Provide exactly one of `id` or `name`.",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Display label for this group component (Grouper `displayExtension`).",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the group.",
			},
			"stem_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Full name of the parent stem, derived from `name`.",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the group is enabled.",
			},
			"type_of_group": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Grouper type of group (e.g. `group`).",
			},
		},
	}
}

func (d *groupDataSource) ConfigValidators(context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config groupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lookup := client.GroupLookup{UUID: config.ID.ValueString(), Name: config.Name.ValueString()}
	group, err := d.client.GetGroup(ctx, lookup)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Grouper group", groupErrorDetail("read", lookupLabel(config.ID, config.Name), err))
		return
	}

	config.ID = types.StringValue(group.UUID)
	config.Name = types.StringValue(group.Name)
	config.DisplayName = types.StringValue(group.DisplayExtension)
	config.Description = stringOrNull(group.Description)
	config.StemName = types.StringValue(parentStemName(group.Name))
	config.Enabled = types.BoolValue(group.Enabled)
	config.TypeOfGroup = types.StringValue(group.TypeOfGroup)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
