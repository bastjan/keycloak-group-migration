# keycloak-group-migration

Migrate Keycloak groups including hierarchy and membership between keycloaks.


## Preparation

### Access to commodore

```sh
# For example: https://api.syn.vshn.net
# IMPORTANT: do NOT add a trailing `/`. Commands below will fail.
export COMMODORE_API_URL=<lieutenant-api-endpoint>
export COMMODORE_API_TOKEN=<lieutenant-api-token>

# Set Project Syn cluster and tenant ID
export CLUSTER_ID=<lieutenant-cluster-id> # Looks like: c-<something>
export TENANT_ID=$(curl -sH "Authorization: Bearer ${COMMODORE_API_TOKEN}" ${COMMODORE_API_URL}/clusters/${CLUSTER_ID} | jq -r .tenant)
```

### Connect to vault

```sh
export VAULT_ADDR=https://vault-int.syn.vshn.net
vault login -method=ldap username=<your.name>
```

## Copy groups

Create group "organizations" in https://id.test.vshn.net/auth/admin/master/console/#/realms/VSHN-main-dev-realm/groups.

Create a user for importing groups on https://id.test.vshn.net and a user for exporting groups on https://id.dev.appuio.cloud.

Copy groups from https://id.dev.appuio.cloud 's root group to the newly created "organizations" group.

```sh
go run . \
  -source-host=https://id.dev.appuio.cloud \
  -source-realm=appuio-cloud-dev \
  -source-username=group-export \
  -source-password=.... \
  -target-host=https://id.test.vshn.net \
  -target-realm=VSHN-main-dev-realm \
  -target-username=group-import \
  -target-password=.... \
  -target-root-group=organizations
```

## Connect cluster to new keycloak

Create new client with $CLUSTER_ID as name in the target realm.
Set "Access Type" to "confidential".
Copy the secret from the "Credentials" tab.
Copy "Valid Redirect URIs" and "Web Origins" over.

Create new client with "appuio-control-api" as name in the target realm.
Copy "Valid Redirect URIs" and "Web Origins" over.

Create the user "appuio-keycloak-sync" in the "master" realm.
In the tab "Role Mappings", select the target realm under "Client Roles" and assign
the role `view-users`.

```sh
vault kv put clusters/kv/${TENANT_ID}/${CLUSTER_ID}/oidc/appuio-keycloak \
  clientSecret=<SECRET_FROM_CREDENTIALS_TAB>

vault kv put clusters/kv/${TENANT_ID}/${CLUSTER_ID}/oidc/appuio-keycloak-sync \
  username=appuio-keycloak-sync \
  password=<PW>
```

### Update cluster catalog

```sh
KEYCLOAK_HOST=id.test.vshn.net
KEYCLOAK_REALM=VSHN-main-dev-realm
ORGANIZATIONS_ROOT_GROUP=organizations

yq eval -i ".parameters.openshift4_authentication.identityProviders.appuio_keycloak.openID.issuer = \"https://${KEYCLOAK_HOST}/auth/realms/${KEYCLOAK_REALM}\"" ${CLUSTER_ID}.yml

yq eval -i ".parameters.keycloak_attribute_sync_controller.sync_configurations.sync-default-org.url   = \"https://${KEYCLOAK_HOST}\"" ${CLUSTER_ID}.yml
yq eval -i ".parameters.keycloak_attribute_sync_controller.sync_configurations.sync-default-org.realm = \"${KEYCLOAK_REALM}\"" ${CLUSTER_ID}.yml

yq eval -i ".parameters.group_sync_operator.sync.sync-keycloak-groups.providers.keycloak.keycloak.url   = \"https://${KEYCLOAK_HOST}\"" ${CLUSTER_ID}.yml
yq eval -i ".parameters.group_sync_operator.sync.sync-keycloak-groups.providers.keycloak.keycloak.realm = \"${KEYCLOAK_REALM}\"" ${CLUSTER_ID}.yml
yq eval -i ".parameters.group_sync_operator.sync.sync-keycloak-groups.providers.keycloak.keycloak.groups += [\"${ORGANIZATIONS_ROOT_GROUP}\"]" ${CLUSTER_ID}.yml
yq eval -i ".parameters.group_sync_operator.sync.sync-keycloak-groups.providers.keycloak.keycloak.subGroupJoinStripRootGroups += [\"${ORGANIZATIONS_ROOT_GROUP}\"]" ${CLUSTER_ID}.yml

yq eval -i ".parameters.cloud_portal.helm_values.portal.config.issuer = \"https://${KEYCLOAK_HOST}/auth/realms/${KEYCLOAK_REALM}\"" ${CLUSTER_ID}.yml



git commit -am "Switch cluster to ${KEYCLOAK_HOST}"

commodore catalog compile -i --push ${CLUSTER_ID}
```
