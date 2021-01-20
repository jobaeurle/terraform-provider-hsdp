package hsdp

import (
	"context"
	"encoding/base64"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/philips-software/go-hsdp-api/iam"
)

func resourceIAMEmailTemplate() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CreateContext: resourceIAMEmailTemplateCreate,
		ReadContext:   resourceIAMEmailTemplateRead,
		DeleteContext: resourceIAMEmailTemplateDelete,

		Schema: map[string]*schema.Schema{
			"managing_organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"from": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressDefault,
			},
			"format": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "HTML",
			},
			"subject": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "default",
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"locale": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressDefault,
			},
			"link": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressDefault,
			},
			"message_base64": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceIAMEmailTemplateCreate(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)

	var diags diag.Diagnostics

	client, err := config.IAMClient()
	if err != nil {
		return diag.FromErr(err)
	}
	var template iam.EmailTemplate

	template.Type = d.Get("type").(string)
	template.Format = d.Get("format").(string)
	template.Subject = d.Get("subject").(string)
	template.Message = base64.StdEncoding.EncodeToString([]byte(d.Get("message").(string)))
	template.Link = d.Get("link").(string)
	template.Locale = d.Get("locale").(string)
	template.From = d.Get("from").(string)
	template.ManagingOrganization = d.Get("managing_organization").(string)

	createdTemplate, _, err := client.EmailTemplates.CreateTemplate(template)
	if err != nil {
		return diag.FromErr(err)
	}
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(createdTemplate.ID)
	d.Set("message_base64", createdTemplate.Message)
	return diags
}

func resourceIAMEmailTemplateRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)

	var diags diag.Diagnostics

	client, err := config.IAMClient()
	if err != nil {
		return diag.FromErr(err)
	}

	template, _, err := client.EmailTemplates.GetTemplateByID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("subject", template.Subject)
	if template.Locale != "default" {
		d.Set("locale", template.Locale)
	}
	d.Set("from", template.From)
	d.Set("format", template.Format)
	d.Set("type", template.Type)
	d.Set("link", template.Link)
	// Message is not returned in the read call

	d.SetId(template.ID)
	return diags
}

func resourceIAMEmailTemplateDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)

	var diags diag.Diagnostics

	client, err := config.IAMClient()
	if err != nil {
		return diag.FromErr(err)
	}

	var template iam.EmailTemplate
	template.ID = d.Id()
	ok, _, err := client.EmailTemplates.DeleteTemplate(template)
	if err != nil {
		return diag.FromErr(err)
	}
	if ok {
		d.SetId("")
	}
	return diags
}
