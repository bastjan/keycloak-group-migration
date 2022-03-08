package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/vshn/appuio-keycloak-adapter/keycloak"
)

var (
	sourceHost      string
	sourceRealm     string
	sourceUsername  string
	sourcePassword  string
	sourceRootGroup string

	targetHost      string
	targetRealm     string
	targetUsername  string
	targetPassword  string
	targetRootGroup string
)

func main() {
	flag.StringVar(&sourceHost, "source-host", "", "The keycloak host groups should be copied from.")
	flag.StringVar(&sourceRealm, "source-realm", "", "The keycloak realm groups should be copied from.")
	flag.StringVar(&sourceUsername, "source-username", "", "The keycloak username groups should be copied from.")
	flag.StringVar(&sourcePassword, "source-password", "", "The keycloak password groups should be copied from.")
	flag.StringVar(&sourceRootGroup, "source-root-group", "", "Optional. The keycloak root-group groups should be copied from.")

	flag.StringVar(&targetHost, "target-host", "", "The keycloak host groups should be copied to.")
	flag.StringVar(&targetRealm, "target-realm", "", "The keycloak realm groups should be copied to.")
	flag.StringVar(&targetUsername, "target-username", "", "The keycloak username groups should be copied to.")
	flag.StringVar(&targetPassword, "target-password", "", "The keycloak password groups should be copied to.")
	flag.StringVar(&targetRootGroup, "target-root-group", "", "Optional. The keycloak root-group groups should be copied to.")

	flag.Parse()

	ctx := context.Background()

	sc := keycloak.NewClient(sourceHost, sourceRealm, sourceUsername, sourcePassword)
	sc.RootGroup = sourceRootGroup

	tc := keycloak.NewClient(targetHost, targetRealm, targetUsername, targetPassword)
	tc.RootGroup = targetRootGroup

	sourceGroups, err := sc.ListGroups(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to load groups from source: %w", err))
	}

	fmt.Println("Migration groups", sourceGroups)

	for _, sg := range sourceGroups {
		fmt.Println("Migrating", sg.Path())
		tg, err := tc.PutGroup(ctx, sg)
		if err != nil {
			if _, ok := err.(*keycloak.MembershipSyncErrors); ok {
				fmt.Println(fmt.Errorf("WARNING failed to migrate member: %w", err))
			} else {
				panic(fmt.Errorf("failed to load groups from source: %w", err))
			}
		}
		fmt.Println("Migrated group as", tg)
	}
}
