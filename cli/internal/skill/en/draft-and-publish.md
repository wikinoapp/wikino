# Guide: Writing and publishing drafts

See [SKILL.md](SKILL.md) for the list of commands.
Before using a command for the first time, check its arguments and output fields with `wikino <command> --help`.

A draft is a way to rewrite a page without publishing it.
When the user asks you to "make it a draft" or "show me before publishing", or when the token only has draft permissions, write a draft instead of updating the page directly.
You can check the token's scopes with `wikino auth status`.

## Create a draft

- Create a draft of an existing page with `wikino draft create <page number>`. Without a body, the page's current content becomes the draft
- Create a draft of a new page with `wikino draft create --topic <number> --title <title> --body-file <file>`. The `page_number` in the output is the new page's number
- If the page already has your draft, creating one fails with exit code 6. Read the current draft with `wikino draft get <page number> --md` and update it instead

Creating a draft of a new page is not idempotent.
When the result is unknown, check whether it was created with `wikino draft list` before resending.

## Rewrite a draft

How to write (passing the body, adding links, checking with `--dry-run`) is the same as in [edit-page.md](edit-page.md).
Use the `wikino draft` commands instead of `wikino page`.

- Read with `wikino draft get <page number> --md`, and write back with `wikino draft update <page number> --md-file <file>`
- To change only part of it, use `wikino draft update <page number> --edits <file>`

Drafts are also updated by autosave in the web UI.
If the user has the same draft open in the web UI, writes tend to fail with exit code 6.
In that case, read it again and reapply your change; if it keeps happening, ask the user to stop editing in the web UI.

## Check the revision history

`wikino draft revision list <page number>` lists the records of each creation and update of the draft, newest first.
To go back to past content, pass the body read with `wikino draft revision get <page number> <revision ID> --md` to `wikino draft update`.
The `--md` output of a revision has no version, so it cannot be passed to `--md-file` as is.

## Publish or delete

Publishing (`wikino draft publish`) and deleting (`wikino draft delete`) take the version of the draft you read in `--if-match`.
The version is the `etag` in the output of `wikino draft get`.

- Publishing overwrites the page with the draft's content, and the draft and its revisions are removed. Even if the page was updated after the draft was created, those changes are overwritten by the draft's content
- Deleting removes the draft and its revisions, and cannot be undone. The page itself remains
- A draft without a title cannot be published. Give it a title first with `wikino draft update <page number> --title <title>`

### When confirmation is required

Publishing and deleting without `--yes` do nothing and exit with code 9.
The JSON on standard output contains what will happen (`changes`) and the command to run after approval (`confirm_command`).

1. Show the user the `changes` as they are
2. Ask the user for approval and wait for the answer
3. If approved, run `confirm_command` as is. Do not rewrite it or put in a different version
4. If not approved, do not run it

Do not add `--yes` yourself to skip the confirmation.
Even if the user said in advance that you may publish without asking, show the `changes` before running it.

If running `confirm_command` fails with exit code 6, the draft was updated after the confirmation.
Read it again, tell the user what it now contains, and ask for confirmation again.

## Use a token that only has draft permissions

A token without the page write scope (page:write) cannot create or update pages, publish drafts, or create drafts of new pages.
It can create and update drafts of existing pages.
In this setup, after writing a draft, show the user the draft's URL (the `url` in the output) and have them check and publish it in the web UI.
