package iam

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/philips-software/terraform-provider-hsdp/internal/config"
	"github.com/philips-software/terraform-provider-hsdp/internal/tools"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/philips-software/go-hsdp-api/iam"
)

func ResourceIAMUser() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CreateContext: resourceIAMUserCreate,
		ReadContext:   resourceIAMUserRead,
		UpdateContext: resourceIAMUserUpdate,
		DeleteContext: resourceIAMUserDelete,

		SchemaVersion: 2,
		Schema: map[string]*schema.Schema{
			"username": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "use login field instead",
			},
			"login": {
				Type:             schema.TypeString,
				DiffSuppressFunc: tools.SuppressCaseDiffs,
				Required:         true,
			},
			"email": {
				Type:             schema.TypeString,
				DiffSuppressFunc: tools.SuppressCaseDiffs,
				Required:         true,
			},
			"password": {
				Type:      schema.TypeString,
				Sensitive: true,
				Optional:  true,
			},
			"first_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"last_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"mobile": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"organization_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"preferred_language": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: tools.SuppressEmptyPreferredLanguage,
			},
			"preferred_communication_channel": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: tools.SuppressDefaultCommunicationChannel,
			},
		},
	}
}

func resourceIAMUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*config.Config)
	client, err := c.IAMClient()
	if err != nil {
		return diag.FromErr(err)
	}

	last := d.Get("last_name").(string)
	first := d.Get("first_name").(string)
	email := d.Get("username").(string) // Deprecated
	mobile := d.Get("mobile").(string)
	login := d.Get("login").(string)
	password := d.Get("password").(string)
	if login == "" {
		login = email
	}
	email = d.Get("email").(string)
	organization := d.Get("organization_id").(string)
	preferredLanguage := d.Get("preferred_language").(string)
	preferredCommunicationChannel := d.Get("preferred_communication_channel").(string)

	// First check if this user already exists
	foundUser, _, err := client.Users.GetUserByID(login)
	if err == nil && (foundUser != nil && foundUser.ID != "") {
		if foundUser.ManagingOrganization != organization {
			return diag.FromErr(fmt.Errorf("user '%s' already exists but is managed by a different IAM organization", login))
		}
		d.SetId(foundUser.ID)
		return resourceIAMUserRead(ctx, d, m)
	}
	person := iam.Person{
		ResourceType: "Person",
		Name: iam.Name{
			Family: last,
			Given:  first,
		},
		LoginID:  login,
		Password: password,
		Telecom: []iam.TelecomEntry{
			{
				System: "email",
				Value:  email,
			},
		},
		ManagingOrganization:          organization,
		PreferredLanguage:             preferredLanguage,
		PreferredCommunicationChannel: preferredCommunicationChannel,
		IsAgeValidated:                "true",
	}
	if mobile != "" {
		person.Telecom = append(person.Telecom,
			iam.TelecomEntry{
				System: "mobile",
				Value:  mobile,
			})
	}
	user, resp, err := client.Users.CreateUser(person)
	if err != nil {
		return diag.FromErr(err)
	}
	if user == nil {
		return diag.FromErr(fmt.Errorf("error creating user '%s': %v %w", login, resp, err))
	}
	d.SetId(user.ID)
	return resourceIAMUserRead(ctx, d, m)
}

func resourceIAMUserRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	c := m.(*config.Config)
	client, err := c.IAMClient()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()

	user, _, err := client.Users.GetUserByID(id)
	if err != nil {
		if errors.Is(err, iam.ErrEmptyResults) {
			// Means the user was cleared, probably due to not activating their account
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}
	_ = d.Set("login", user.LoginID)
	_ = d.Set("last_name", user.Name.Family)
	_ = d.Set("first_name", user.Name.Given)
	_ = d.Set("email", user.EmailAddress)
	_ = d.Set("login", user.LoginID)
	_ = d.Set("organization_id", user.ManagingOrganization)
	_ = d.Set("preferred_communication_channel", user.PreferredCommunicationChannel)
	_ = d.Set("preferred_language", user.PreferredLanguage)
	return diags
}

func resourceIAMUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	c := m.(*config.Config)
	client, err := c.IAMClient()
	if err != nil {
		return diag.FromErr(err)
	}

	var p iam.Person
	p.ID = d.Id()

	if d.HasChange("login") {
		newLogin := d.Get("login").(string)
		_, _, err := client.Users.ChangeLoginID(p, newLogin)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("last_name") || d.HasChange("first_name") || d.HasChange("email") ||
		d.HasChange("mobile") || d.HasChange("preferred_language") || d.HasChange("preferred_communication_channel") {
		profile, _, err := client.Users.LegacyGetUserByUUID(d.Id())
		if err != nil {
			return diag.FromErr(fmt.Errorf("resourceIAMUserUpdate LegacyGetUserByUUID: %w", err))
		}
		profile.FamilyName = d.Get("last_name").(string)
		profile.GivenName = d.Get("first_name").(string)
		profile.PreferredLanguage = d.Get("preferred_language").(string)
		profile.PreferredCommunicationChannel = d.Get("preferred_communication_channel").(string)
		profile.Contact.EmailAddress = d.Get("email").(string)
		if profile.MiddleName == "" {
			profile.MiddleName = " "
		}
		profile.Contact.MobilePhone = d.Get("mobile").(string)
		profile.ID = d.Id()
		_, _, err = client.Users.LegacyUpdateUser(*profile)
		if err != nil {
			return diag.FromErr(fmt.Errorf("resourceIAMUserUpdate LegacyUpdateUser: %w", err))
		}
	}
	if d.HasChange("password") {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "password change not propagated",
			Detail:   "changing the password after a user is created has no effect",
		})
	}
	readDiags := resourceIAMUserRead(ctx, d, m)
	return append(diags, readDiags...)
}

func resourceIAMUserDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	c := m.(*config.Config)
	client, err := c.IAMClient()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()

	user, _, err := client.Users.GetUserByID(id)
	if err != nil {
		if _, ok := err.(*iam.UserError); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}
	if user == nil {
		return diags
	}
	var person iam.Person
	person.ID = user.ID
	_, resp, err := client.Users.DeleteUser(person)
	if err != nil {
		return diag.FromErr(fmt.Errorf("DeleteUser error: %w", err))
	}
	if resp != nil && resp.StatusCode == http.StatusConflict {
		return diag.FromErr(fmt.Errorf("DeleteUser return HTTP 409 Conflict: %w", err))
	}
	if resp != nil && resp.StatusCode != http.StatusNoContent {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "DeleteUser returned unexpected result",
			Detail:   fmt.Sprintf("DeleteUser returned status '%d', which is unexpected: %v", resp.StatusCode, err),
		})
	}
	d.SetId("")
	return diags
}
