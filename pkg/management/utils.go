package management

import "github.com/lensesio/lenses-go/pkg/api"

//GroupView the view model for group to be printed
type GroupView struct {
	Name                       string          `json:"name" yaml:"name" header:"name"`
	Namespaces                 []api.Namespace `json:"namespaces,omitempty" yaml:"namespaces" header:"Namespaces,count"`
	ScopedPermissions          []string        `json:"scopedPermissions" yaml:"scopedPermissions" header:"Scoped Permissions"`
	AdminPermissions           []string        `json:"adminPermissions" yaml:"adminPermissions" header:"Admin Permissions"`
	UserAccountsCount          int             `json:"userAccounts" yaml:"userAccounts" header:"User Accounts"`
	ServiceAccountsCount       int             `json:"serviceAccounts" yaml:"serviceAccounts" header:"Service Accounts"`
	ConnectClustersPermissions []string        `json:"connectClustersPermissions" yaml:"connectClustersPermissions" header:"Connect clusters access"`
}

// PrintGroup returns a group for table printing
func PrintGroup(g api.Group) GroupView {
	return GroupView{
		Name:                       g.Name,
		Namespaces:                 g.Namespaces,
		ScopedPermissions:          g.ScopedPermissions,
		AdminPermissions:           g.AdminPermissions,
		ConnectClustersPermissions: g.ConnectClustersPermissions,
	}
}

//TokenView the view model for token to be printed
type TokenView struct {
	Name  string `json:"name" yaml:"name" header:"Service Account"`
	Token string `json:"token" yaml:"token" header:"Token"`
}

// PrintToken returns token for table printing
func PrintToken(name, token string) TokenView {
	return TokenView{
		Name:  name,
		Token: token,
	}
}
