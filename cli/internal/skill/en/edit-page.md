# Guide: Writing pages

See [SKILL.md](SKILL.md) for the list of commands.
Before using a command for the first time, check its arguments and output fields with `wikino <command> --help`.

Follow this guide only when the user explicitly asks you to write a page.
Creating and updating a page publishes it immediately.
When the user wants to check the content before it is published, or the token only has draft permissions, write a draft instead by following [draft-and-publish.md](draft-and-publish.md).

## Read before you write

- When updating, read the target page with `wikino page get <number> --md`. The `wikino_etag` in the output is the version you read
- When creating, read similar pages in the same topic and match their headings, style, and linking
- Check the number of the topic to create the page in with `wikino topic list`
- Check whether a page with the same title already exists with `wikino page list --topic <number> --title <title>`. If it does, update it instead of creating one

## Add links

- For a wiki link, find the target page with `wikino page search` first, and write its title exactly
- Do not add speculative links to pages that do not exist. A wiki link to a page that does not exist creates an unpublished page with that title
- Point to a page in another topic with `[[Topic name/Title]]`

Before writing, run the command with `--dry-run` and check `wikilinks` in the output.

- A link whose `resolution` is `new_page` will create an unpublished page when written. If that is not intended, fix the title or remove the link
- A link with `topic_not_found` will not become a link. Check the topic name with `wikino topic list`

## Choose how to pass the body

Do not write the body directly in the command's arguments; write it to a file and pass the file.
This avoids quotes, line breaks, and `$` being mangled by the shell.
On Windows, the character encoding can change, so always pass the body as a file, not through a pipe.

### Create a new page

Write the body to a file, and create the page with `wikino page create --topic <number> --title <title> --body-file <file>`.

### Change only part of it

For additions and fixes in a few places, use search and replace with `--edits`.
It does not resend the whole body, so the amount written stays small even for large pages.

- Make `old_str` match the current body exactly, including spaces and line breaks. Copy it from the output of `wikino page get <number> --md`
- If `old_str` matches more than one place, it fails. Include surrounding lines so that it matches exactly one place
- To append at the end, use the last few lines as `old_str`, and those lines plus the new content as `new_str`
- `--edits` cannot change the title

### Rewrite a large part

Edit the file written by `wikino page get <number> --md > page-<number>.md`, and write it back with `wikino page update <number> --md-file page-<number>.md`.

- Do not change the frontmatter keys that start with `wikino_` (only `wikino_title` may be changed, to change the title)
- For a large page, do not load the whole file into your context; read and edit only the parts you need
- `--md-file` reads the version from the file, so do not add `--if-match`

### Change only the title

Change it with `wikino page update <number> --title <new title> --if-match <etag>`.
For now, wiki links on other pages are not rewritten.
`wikino page backlinks` shows the pages linking to it, so tell the user if needed.

## Pass the version

When updating with `--body-file` or `--title`, pass the version you read to `--if-match`.
The version is the `etag` in the output of `wikino page get` (`wikino_etag` with `--md`).

If it fails with exit code 6, someone else changed the page after you read it.
Read it again, reapply your change without discarding the other person's change, and then write.
Use `--force` only when the user says it is fine to overwrite.

## Keep attachment references

Do not remove or rewrite `/attachments/{ID}` references in a body.
New attachments cannot be added from the CLI, so if asked to add an image, tell the user to add it in the web UI.

## Check the result of the write

- On success, show the user the `url` in the output
- If a list of created link targets (unpublished pages) is printed on standard error, tell the user
- When the result is unknown, for example because of a network failure, read the page again to check whether it was applied before resending. Creating a page is not idempotent, and resending can create it twice (when creating, check with `wikino page list --topic <number> --title <title>`)
