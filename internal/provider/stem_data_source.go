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
	_ datasource.DataSource                     = (*stemDataSource)(nil)
	_ datasource.DataSourceWithConfigure        = (*stemDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*stemDataSource)(nil)
)

// NewStemDataSource constructs the grouper_stem data source.
func NewStemDataSource() datasource.DataSource {
	return &stemDataSource{}
}

type stemDataSource struct {
	client *client.Client
}

type stemDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
	ParentStemName types.String `tfsdk:"parent_stem_name"`
}

func (d *stemDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stem"
}

func (d *stemDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing Grouper stem (folder) by exactly one of `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Grouper UUID of the stem. Provide exactly one of `id` or `name`.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Full colon-delimited name of the stem. Provide exactly one of `id` or `name`.",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Display label for this stem component (Grouper `displayExtension`).",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the stem.",
			},
			"parent_stem_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Full name of the parent stem, derived from `name`.",
			},
		},
	}
}

func (d *stemDataSource) ConfigValidators(context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *stemDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *stemDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config stemDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lookup := client.StemLookup{UUID: config.ID.ValueString(), Name: config.Name.ValueString()}
	stem, err := d.client.GetStem(ctx, lookup)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Grouper stem", stemErrorDetail("read", lookupLabel(config.ID, config.Name), err))
		return
	}

	config.ID = types.StringValue(stem.UUID)
	config.Name = types.StringValue(stem.Name)
	config.DisplayName = types.StringValue(stem.DisplayExtension)
	config.Description = stringOrNull(stem.Description)
	config.ParentStemName = types.StringValue(parentStemName(stem.Name))
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// lookupLabel returns a human-friendly label for whichever lookup key was set.
func lookupLabel(id, name types.String) string {
	if !name.IsNull() && name.ValueString() != "" {
		return name.ValueString()
	}
	return id.ValueString()
}
