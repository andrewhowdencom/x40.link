# Change the CLI account

Use explicit login when the CLI has cached credentials for the wrong account.

1. Start a fresh device authorization flow:

    ```bash
    @ login
    ```

2. Open the verification URL printed by the CLI and enter the displayed code.
3. Sign in to the account that should own newly created links.
4. Wait for the CLI to confirm that login succeeded.

The CLI keeps the previous credentials until the new device flow completes. If
authentication fails or is cancelled, the previous account remains cached and
usable.

The identity provider may reuse an account already signed in to the browser.
Check the selected account on the verification page before approving access.

Use the command help to view OAuth endpoint and client configuration flags:

```bash
@ login --help
```

The CLI does not currently provide logout, remote token revocation, or an
account-inspection command. Running `@ login` again is the supported way to
replace the locally cached account.
