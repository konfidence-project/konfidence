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
http://localhost:5173/api/login?returnTo=/landscape
```

Test users:

```text
alice@example.com / password  ADMIN
ada@example.com   / password  DEV
paul@example.com  / password  PM
nina@example.com  / password  no role
```
