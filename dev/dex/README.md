# Local Dex IDP

Run a local OIDC provider for the API auth flow:

```sh
make idp-up
```

Start the API with Dex endpoints:

```sh
make run-api-with-idp
```

Open the browser flow:

```text
http://localhost:8090/login
```

Test user:

```text
alice@example.com / password
```
