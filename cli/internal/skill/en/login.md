# Guide: Logging in and authentication errors

See [SKILL.md](SKILL.md) for the list of commands.
Before using a command for the first time, check its arguments and output fields with `wikino <command> --help`.

## Check the current credentials

First run `wikino auth status`.
For each credential, it prints the host, space, member, the token's scopes (`token.scopes`), and its expiry.

- If there are no credentials, go to "Log in"
- If there are credentials for the space you want but it is not the default, add `--space` to the commands, or with the user's consent switch the default with `wikino auth switch <space>`
- If an item has an `error` with a reason, follow its `hint`

When a token is given in the environment variable `WIKINO_TOKEN`, it takes precedence over saved credentials.
Logging in again has no effect in that case, so ask the user to change the token.

## Log in

Logging in requires the user to open Wikino in a browser and approve.
You cannot complete it alone, so have the user do it in one of the following ways.

### Have the user run it in their terminal (recommended)

Ask the user to run `wikino auth login <space>` in their terminal.
A browser opens, and logging in finishes when they approve.

### Run it yourself and pass the URL and code to the user

Running `wikino auth login <space> --device` prints a verification URL and code on standard error.
Pass the URL and code to the user, and have them open it in a browser, enter the code, and approve.
The command waits until it is approved (the code expires in 10 minutes).
If it is not approved in time it fails, so run it again when the user is ready.

### Use a personal access token

When the user gives you a personal access token issued in the space settings, have them write the token to a file, and log in with `wikino auth login <space> --with-token < <file>`.
Do not copy the token into command arguments or the conversation.

## Authentication failed (exit code 3)

- Not logged in, or the token is invalid or expired
- Check the credentials with `wikino auth status`, and if there are none, follow "Log in". If there are but the token is invalid, have the user log in again

## Insufficient permission (exit code 4)

The token's scopes or the member's permissions are insufficient.
The error message often names the missing scope.

- Check `token.scopes` with `wikino auth status`
- If a scope is missing, ask the user to log in again with that scope (`wikino auth login <space> --scopes <scopes>`)
- If the scopes are sufficient but it still fails, it is a problem with the member's permissions (their role in the space or topic). Tell the user to consult the space's administrator

The lack of permission to write pages may be deliberate on the user's part.
Do not ask to widen permissions on your own; first consider whether you can write a draft instead ([draft-and-publish.md](draft-and-publish.md)).
