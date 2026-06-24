package provider

import (
	"context"
	"fmt"

	"github.com/ahmedali6/terraform-provider-dokploy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

type ProjectDataSource struct {
	client *client.DokployClient
}

type ProjectEnvironmentModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type ProjectDataSourceModel struct {
	ID           types.String              `tfsdk:"id"`
	Name         types.String              `tfsdk:"name"`
	Description  types.String              `tfsdk:"description"`
	Environments []ProjectEnvironmentModel `tfsdk:"environments"`
}

func (d *ProjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an existing Dokploy project by its id or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier of the project. Either id or name must be set.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The display name of the project. Either name or id must be set.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the project.",
			},
			"environments": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The environments belonging to the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The unique identifier of the environment.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the environment.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The description of the environment.",
						},
					},
				},
			},
		},
	}
}

func (d *ProjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull() && data.ID.ValueString() != ""
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""
	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing project selector",
			"Either \"id\" or \"name\" must be set to look up a project.",
		)
		return
	}

	var project *client.Project
	if hasID {
		p, err := d.client.GetProject(data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Project", err.Error())
			return
		}
		project = p
	} else {
		projects, err := d.client.ListProjects()
		if err != nil {
			resp.Diagnostics.AddError("Unable to List Projects", err.Error())
			return
		}
		for i := range projects {
			if projects[i].Name == data.Name.ValueString() {
				project = &projects[i]
				break
			}
		}
		if project == nil {
			resp.Diagnostics.AddError(
				"Project Not Found",
				fmt.Sprintf("No project with name %q was found.", data.Name.ValueString()),
			)
			return
		}
	}

	data.ID = types.StringValue(project.ID)
	data.Name = types.StringValue(project.Name)
	data.Description = types.StringValue(project.Description)

	data.Environments = make([]ProjectEnvironmentModel, 0, len(project.Environments))
	for _, env := range project.Environments {
		data.Environments = append(data.Environments, ProjectEnvironmentModel{
			ID:          types.StringValue(env.ID),
			Name:        types.StringValue(env.Name),
			Description: types.StringValue(env.Description),
		})
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
