package api

import (
	"github.com/Unknwon/macaron"
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/middleware"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/macaron-contrib/binding"
)

// Register adds http routes
func Register(r *macaron.Macaron) {
	reqSignedIn := middleware.Auth(&middleware.AuthOptions{ReqSignedIn: true})
	reqGrafanaAdmin := middleware.Auth(&middleware.AuthOptions{ReqSignedIn: true, ReqGrafanaAdmin: true})
	reqEditorRole := middleware.RoleAuth(m.ROLE_EDITOR, m.ROLE_ADMIN)
	regOrgAdmin := middleware.RoleAuth(m.ROLE_ADMIN)
	quota := middleware.Quota
	bind := binding.Bind

	// not logged in views
	r.Get("/", reqSignedIn, Index)
	r.Get("/logout", Logout)
	r.Post("/login", quota("session"), bind(dtos.LoginCommand{}), wrap(LoginPost))
	r.Get("/login/:name", quota("session"), OAuthLogin)
	r.Get("/login", LoginView)
	r.Get("/invite/:code", Index)

	// authed views
	r.Get("/profile/", reqSignedIn, Index)
	r.Get("/org/", reqSignedIn, Index)
	r.Get("/org/new", reqSignedIn, Index)
	r.Get("/datasources/", reqSignedIn, Index)
	r.Get("/datasources/edit/*", reqSignedIn, Index)
	r.Get("/org/users/", reqSignedIn, Index)
	r.Get("/org/apikeys/", reqSignedIn, Index)
	r.Get("/dashboard/import/", reqSignedIn, Index)
	r.Get("/admin/settings", reqGrafanaAdmin, Index)
	r.Get("/admin/users", reqGrafanaAdmin, Index)
	r.Get("/admin/users/create", reqGrafanaAdmin, Index)
	r.Get("/admin/users/edit/:id", reqGrafanaAdmin, Index)
	r.Get("/admin/orgs", reqGrafanaAdmin, Index)
	r.Get("/admin/orgs/edit/:id", reqGrafanaAdmin, Index)

	r.Get("/dashboard/*", reqSignedIn, Index)
	r.Get("/dashboard-solo/*", reqSignedIn, Index)

	r.Get("/collectors", reqSignedIn, Index)
	r.Get("/collectors/*", reqSignedIn, Index)
	r.Get("/endpoints", reqSignedIn, Index)
	r.Get("/endpoints/*", reqSignedIn, Index)

	// sign up
	r.Get("/signup", Index)
	r.Get("/api/user/signup/options", wrap(GetSignUpOptions))
	r.Post("/api/user/signup", quota("user"), bind(dtos.SignUpForm{}), wrap(SignUp))
	r.Post("/api/user/signup/step2", bind(dtos.SignUpStep2Form{}), wrap(SignUpStep2))

	// invited
	r.Get("/api/user/invite/:code", wrap(GetInviteInfoByCode))
	r.Post("/api/user/invite/complete", bind(dtos.CompleteInviteForm{}), wrap(CompleteInvite))

	// reset password
	r.Get("/user/password/send-reset-email", Index)
	r.Get("/user/password/reset", Index)

	r.Post("/api/user/password/send-reset-email", bind(dtos.SendResetPasswordEmailForm{}), wrap(SendResetPasswordEmail))
	r.Post("/api/user/password/reset", bind(dtos.ResetUserPasswordForm{}), wrap(ResetPassword))

	// dashboard snapshots
	r.Post("/api/snapshots/", bind(m.CreateDashboardSnapshotCommand{}), CreateDashboardSnapshot)
	r.Get("/dashboard/snapshot/*", Index)

	r.Get("/api/snapshots/:key", GetDashboardSnapshot)
	r.Get("/api/snapshots-delete/:key", DeleteDashboardSnapshot)

	// api renew session based on remember cookie
	r.Get("/api/login/ping", quota("session"), LoginApiPing)

	// authed api
	r.Group("/api", func() {

		// user (signed in)
		r.Group("/user", func() {
			r.Get("/", wrap(GetSignedInUser))
			r.Put("/", bind(m.UpdateUserCommand{}), wrap(UpdateSignedInUser))
			r.Post("/using/:id", wrap(UserSetUsingOrg))
			r.Get("/orgs", wrap(GetSignedInUserOrgList))
			r.Post("/stars/dashboard/:id", wrap(StarDashboard))
			r.Delete("/stars/dashboard/:id", wrap(UnstarDashboard))
			r.Put("/password", bind(m.ChangeUserPasswordCommand{}), wrap(ChangeUserPassword))
			r.Get("/quotas", wrap(GetUserQuotas))
		})

		// users (admin permission required)
		r.Group("/users", func() {
			r.Get("/", wrap(SearchUsers))
			r.Get("/:id", wrap(GetUserById))
			r.Get("/:id/orgs", wrap(GetUserOrgList))
			r.Put("/:id", bind(m.UpdateUserCommand{}), wrap(UpdateUser))
		}, reqGrafanaAdmin)

		// org information available to all users.
		r.Group("/org", func() {
			r.Get("/", wrap(GetOrgCurrent))
			r.Get("/quotas", wrap(GetOwnOrgQuotas))
		})

		// current org
		r.Group("/org", func() {
			r.Put("/", bind(dtos.UpdateOrgForm{}), wrap(UpdateOrgCurrent))
			r.Put("/address", bind(dtos.UpdateOrgAddressForm{}), wrap(UpdateOrgAddressCurrent))
			r.Post("/users", quota("user"), bind(m.AddOrgUserCommand{}), wrap(AddOrgUserToCurrentOrg))
			r.Get("/users", wrap(GetOrgUsersForCurrentOrg))
			r.Patch("/users/:userId", bind(m.UpdateOrgUserCommand{}), wrap(UpdateOrgUserForCurrentOrg))
			r.Delete("/users/:userId", wrap(RemoveOrgUserForCurrentOrg))

			// invites
			r.Get("/invites", wrap(GetPendingOrgInvites))
			r.Post("/invites", quota("user"), bind(dtos.AddInviteForm{}), wrap(AddOrgInvite))
			r.Patch("/invites/:code/revoke", wrap(RevokeInvite))
		}, regOrgAdmin)

		// create new org
		r.Post("/orgs", quota("org"), bind(m.CreateOrgCommand{}), wrap(CreateOrg))

		// search all orgs
		r.Get("/orgs", reqGrafanaAdmin, wrap(SearchOrgs))

		// orgs (admin routes)
		r.Group("/orgs/:orgId", func() {
			r.Get("/", wrap(GetOrgById))
			r.Put("/", bind(dtos.UpdateOrgForm{}), wrap(UpdateOrg))
			r.Put("/address", bind(dtos.UpdateOrgAddressForm{}), wrap(UpdateOrgAddress))
			r.Delete("/", wrap(DeleteOrgById))
			r.Get("/users", wrap(GetOrgUsers))
			r.Post("/users", bind(m.AddOrgUserCommand{}), wrap(AddOrgUser))
			r.Patch("/users/:userId", bind(m.UpdateOrgUserCommand{}), wrap(UpdateOrgUser))
			r.Delete("/users/:userId", wrap(RemoveOrgUser))
			r.Get("/quotas", wrap(GetOrgQuotas))
			r.Put("/quotas/:target", bind(m.UpdateOrgQuotaCmd{}), wrap(UpdateOrgQuota))
		}, reqGrafanaAdmin)

		// auth api keys
		r.Group("/auth/keys", func() {
			r.Get("/", wrap(GetApiKeys))
			r.Post("/", quota("api_key"), bind(m.AddApiKeyCommand{}), wrap(AddApiKey))
			r.Delete("/:id", wrap(DeleteApiKey))
		}, regOrgAdmin)

		// Data sources
		r.Group("/datasources", func() {
			r.Get("/", GetDataSources)
			r.Post("/", quota("data_source"), bind(m.AddDataSourceCommand{}), AddDataSource)
			r.Put("/:id", bind(m.UpdateDataSourceCommand{}), UpdateDataSource)
			r.Delete("/:id", DeleteDataSource)
			r.Get("/:id", wrap(GetDataSourceById))
			r.Get("/plugins", GetDataSourcePlugins)
		}, regOrgAdmin)

		r.Get("/frontend/settings/", GetFrontendSettings)
		r.Any("/datasources/proxy/:id/*", reqSignedIn, ProxyDataSourceRequest)
		r.Any("/datasources/proxy/:id", reqSignedIn, ProxyDataSourceRequest)

		// Dashboard
		r.Group("/dashboards", func() {
			r.Combo("/db/:slug").Get(GetDashboard).Delete(DeleteDashboard)
			r.Post("/db", reqEditorRole, bind(m.SaveDashboardCommand{}), PostDashboard)
			r.Get("/file/:file", GetDashboardFromJsonFile)
			r.Get("/home", GetHomeDashboard)
			r.Get("/tags", GetDashboardTags)
		})

		// Search
		r.Get("/search/", Search)

		// metrics
		r.Get("/metrics/test", GetTestMetrics)

		// collectors
		r.Group("/collectors", func() {
			r.Combo("/").
				Get(bind(m.GetCollectorsQuery{}), wrap(GetCollectors)).
				Put(reqEditorRole, quota("collector"), bind(m.AddCollectorCommand{}), wrap(AddCollector)).
				Post(reqEditorRole, bind(m.UpdateCollectorCommand{}), wrap(UpdateCollector))
			r.Get("/:id", wrap(GetCollectorById))
			r.Delete("/:id", reqEditorRole, wrap(DeleteCollector))
		})

		// Monitors
		r.Group("/monitors", func() {
			r.Combo("/").
				Get(bind(m.GetMonitorsQuery{}), wrap(GetMonitors)).
				Put(reqEditorRole, bind(m.AddMonitorCommand{}), wrap(AddMonitor)).
				Post(reqEditorRole, bind(m.UpdateMonitorCommand{}), wrap(UpdateMonitor))
			r.Get("/:id", wrap(GetMonitorById))
			r.Delete("/:id", reqEditorRole, wrap(DeleteMonitor))
		})
		// endpoints
		r.Group("/endpoints", func() {
			r.Combo("/").Get(bind(m.GetEndpointsQuery{}), wrap(GetEndpoints)).
				Put(reqEditorRole, quota("endpoint"), bind(m.AddEndpointCommand{}), wrap(AddEndpoint)).
				Post(reqEditorRole, bind(m.UpdateEndpointCommand{}), wrap(UpdateEndpoint))
			r.Get("/:id", wrap(GetEndpointById))
			r.Delete("/:id", reqEditorRole, wrap(DeleteEndpoint))
			r.Get("/discover", reqEditorRole, bind(m.EndpointDiscoveryCommand{}), wrap(DiscoverEndpoint))
		})

		r.Get("/monitor_types", wrap(GetMonitorTypes))

		//Events
		r.Get("/events", bind(m.GetEventsQuery{}), wrap(GetEvents))

		//Get Graph data from Graphite.
		r.Any("/graphite/*", GraphiteProxy)

	}, reqSignedIn)

	// admin api
	r.Group("/api/admin", func() {
		r.Get("/settings", AdminGetSettings)
		r.Post("/users", bind(dtos.AdminCreateUserForm{}), AdminCreateUser)
		r.Put("/users/:id/password", bind(dtos.AdminUpdateUserPasswordForm{}), AdminUpdateUserPassword)
		r.Put("/users/:id/permissions", bind(dtos.AdminUpdateUserPermissionsForm{}), AdminUpdateUserPermissions)
		r.Delete("/users/:id", AdminDeleteUser)
		r.Get("/users/:id/quotas", wrap(GetUserQuotas))
		r.Put("/users/:id/quotas/:target", bind(m.UpdateUserQuotaCmd{}), wrap(UpdateUserQuota))
	}, reqGrafanaAdmin)

	// rendering
	r.Get("/render/*", reqSignedIn, RenderToPng)

	r.Any("/socket.io/", SocketIO)

	r.NotFound(NotFoundHandler)
}
