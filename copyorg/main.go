package main

import (
	"context"
	"flag"
	"fmt"

	controlv1 "github.com/appuio/control-api/apis/v1"
	"github.com/vshn/appuio-keycloak-adapter/keycloak"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	sourceHost       string
	sourceRealm      string
	sourceLoginRealm string
	sourceUsername   string
	sourcePassword   string
	sourceRootGroup  string
)

func main() {
	flag.StringVar(&sourceHost, "source-host", "", "The keycloak host groups should be copied from.")
	flag.StringVar(&sourceRealm, "source-realm", "", "The keycloak realm groups should be copied from.")
	flag.StringVar(&sourceLoginRealm, "source-login-realm", "", "The realm to log in to the Keycloak server. `source-realm` is used if not set.")
	flag.StringVar(&sourceUsername, "source-username", "", "The keycloak username groups should be copied from.")
	flag.StringVar(&sourcePassword, "source-password", "", "The keycloak password groups should be copied from.")
	flag.StringVar(&sourceRootGroup, "source-root-group", "", "Optional. The keycloak root-group groups should be copied from.")

	flag.Parse()

	ctx := context.Background()

	sc := keycloak.NewClient(sourceHost, sourceRealm, sourceUsername, sourcePassword)
	sc.LoginRealm = sourceLoginRealm
	sc.RootGroup = sourceRootGroup

	sourceGroups, err := sc.ListGroups(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to load groups from source: %w", err))
	}

	fmt.Println("Gattering users", sourceGroups)

	kcUsers := []keycloak.User{}

	for _, sg := range sourceGroups {
		fmt.Println("Gattering from", sg.Path())
		kcUsers = append(kcUsers, sg.Members...)
	}

	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(controlv1.AddToScheme(scheme))

	cl, err := client.New(config.GetConfigOrDie(), client.Options{
		Scheme: scheme,
	})
	if err != nil {
		panic(fmt.Errorf("failed to create client: %w", err))
	}

	userList := controlv1.UserList{}
	err = cl.List(ctx, &userList)
	if err != nil {
		panic(fmt.Errorf("failed to load users: %w", err))
	}

	userMap := map[string]controlv1.User{}
	for _, user := range userList.Items {
		userMap[user.Name] = user
	}

	seen := map[string]struct{}{}

	for _, u := range kcUsers {
		cau, ok := userMap[u.Username]
		if !ok {
			fmt.Printf("WARNING user %q not found k8s\n", u.Username)
			continue
		}

		if _, ok := seen[u.Username]; ok {
			fmt.Printf("SKIPPED user %q already seen\n", u.Username)
			continue
		}

		fmt.Printf("Setting default org for user %q...\n", u.Username)
		toUpdate := &cau
		_, err := controllerutil.CreateOrUpdate(ctx, cl, toUpdate, func() error {
			toUpdate.Spec.Preferences.DefaultOrganizationRef = u.DefaultOrganizationRef
			return nil
		})
		if err != nil {
			fmt.Printf("ERROR updating %q: %s\n", u.Username, err.Error())
			continue
		}
		seen[u.Username] = struct{}{}
		fmt.Printf("OK did set default org for user %q\n", u.Username)
	}
}
