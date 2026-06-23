package provider

import (
	"context"
	"fmt"

	"github.com/ahmedali6/terraform-provider-dokploy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &EnvironmentDataSource{}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

type EnvironmentDataSource struct {
	client *client.DokployClient
}

type EnvironmentDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (d *EnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an existing Dokploy environment within a project. Useful for referencing environments created automatically by Dokploy (such as the default \"production\" environment). Look it up either by name or by id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier of the environment. Either id or name must be set.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the project the environment belongs to.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the environment (e.g. \"production\"). Either name or id must be set.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the environment.",
			},
		},
	}
}

func (d *EnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.DokployClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.DokployClient, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull() && data.ID.ValueString() != ""
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""
	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing environment selector",
			"Either \"id\" or \"name\" must be set to look up an environment.",
		)
		return
	}

	// Environments are returned as part of the parent project.
	project, err := d.client.GetProject(data.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Project", err.Error())
		return
	}

	var match *client.Environment
	for i := range project.Environments {
		env := &project.Environments[i]
		if hasID && env.ID == data.ID.ValueString() {
			match = env
			break
		}
		if !hasID && hasName && env.Name == data.Name.ValueString() {
			match = env
			break
		}
	}

	if match == nil {
		selector := fmt.Sprintf("name %q", data.Name.ValueString())
		if hasID {
			selector = fmt.Sprintf("id %q", data.ID.ValueString())
		}
		resp.Diagnostics.AddError(
			"Environment Not Found",
			fmt.Sprintf("No environment with %s was found in project %q.", selector, data.ProjectID.ValueString()),
		)
		return
	}

	data.ID = types.StringValue(match.ID)
	data.Name = types.StringValue(match.Name)
	data.Description = types.StringValue(match.Description)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
