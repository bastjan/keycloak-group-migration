# copy default orgs

Copies default orgs to the control-api in the current kubeconfig context.

```sh
go run ./copyorg \
  -source-host=https://id.appuio.cloud \
  -source-realm=appuio-cloud \
  -source-login-realm=master \
  -source-username=USER \
  -source-password=${EXPORT_PASSWORD}
```
