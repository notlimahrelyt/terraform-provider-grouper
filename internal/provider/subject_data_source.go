package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/notlimahrelyt/terraform-provider-grouper/internal/client"
)

var (
	_ datasource.DataSource              = (*subjectDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*subjectDataSource)(nil)
)

// NewSubjectDataSource constructs the grouper_subject data source.
func NewSubjectDataSource() datasource.DataSource {
	return &subjectDataSource{}
}

type subjectDataSource struct {
	client *client.Client
}

type subjectDataSourceModel struct {
	SubjectID             types.String `tfsdk:"subject_id"`
	SubjectSourceID       types.String `tfsdk:"subject_source_id"`
	SubjectAttributeNames types.List   `tfsdk:"subject_attribute_names"`
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	SourceID              types.String `tfsdk:"source_id"`
	Attributes            types.Map    `tfsdk:"attributes"`
}

func (d *subjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subject"
}

func (d *subjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Grouper subject by id or identifier (e.g. a login name). Returns only " +
			"non-sensitive reference attributes.",
		Attributes: map[string]schema.Attribute{
			"subject_id": schema.StringAttribute{
				Required: true,
				Description: "Subject id or login identifier (e.g. a username or NetID) to look up. " +
					"Grouper resolves either.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"subject_source_id": schema.StringAttribute{
				Optional:    true,
				Description: "Optional subject source id to disambiguate when the same id exists in multiple sources.",
			},
			"subject_attribute_names": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Subject attributes to retrieve, by their Grouper attribute names. Names depend " +
					"on your subject source configuration (e.g. \"email\", \"name\", \"description\"). " +
					"Requested values are returned in `attributes`. Only request what you need: these values are " +
					"written to Terraform state, which should be treated as sensitive.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical Grouper subject id (may differ from the supplied identifier).",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Subject display name.",
			},
			"source_id": schema.StringAttribute{
				Computed:    true,
				Description: "Resolved subject source id.",
			},
			"attributes": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Map of the requested subject attributes (from `subject_attribute_names`) to " +
					"their values. Empty when no attributes are requested.",
			},
		},
	}
}

func (d *subjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *subjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config subjectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var attributeNames []string
	if !config.SubjectAttributeNames.IsNull() && !config.SubjectAttributeNames.IsUnknown() {
		resp.Diagnostics.Append(config.SubjectAttributeNames.ElementsAs(ctx, &attributeNames, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	subject, err := d.client.GetSubject(ctx, client.SubjectLookup{
		SubjectID:       config.SubjectID.ValueString(),
		SubjectSourceID: config.SubjectSourceID.ValueString(),
	}, attributeNames)
	if err != nil {
		detail := fmt.Sprintf("Grouper could not resolve subject %q.", config.SubjectID.ValueString())
		if client.IsNotFound(err) {
			detail += " Confirm the subject id/identifier and (if needed) set subject_source_id."
		} else {
			detail = fmt.Sprintf("Unexpected error resolving subject %q: %s", config.SubjectID.ValueString(), err.Error())
		}
		resp.Diagnostics.AddError("Unable to read Grouper subject", detail)
		return
	}

	config.ID = types.StringValue(subject.ID)
	config.Name = stringOrNull(subject.Name)
	config.SourceID = types.StringValue(subject.SourceID)

	attrValues := subject.Attributes
	if attrValues == nil {
		attrValues = map[string]string{}
	}
	attrs, diags := types.MapValueFrom(ctx, types.StringType, attrValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Attributes = attrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
