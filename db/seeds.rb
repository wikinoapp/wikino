# typed: strict

create_account_use_case_result = CreateAccountUseCase.new.call(
  space_identifier: "nonoto-dev",
  email: "user_1@example.com",
  atname: "user_1",
  locale: UserLocale::Ja,
  password: "password",
  time_zone: "Asia/Tokyo"
)
viewer = create_account_use_case_result.user

create_notebook_use_case_result_1 = CreateNotebookUseCase.new.call(
  viewer:,
  name: "公開ノートブック1",
  description: "1つ目の公開ノートブックです",
  visibility: NotebookVisibility::Public.serialize
)
public_notebook_1 = create_notebook_use_case_result_1.notebook

create_notebook_use_case_result_2 = CreateNotebookUseCase.new.call(
  viewer:,
  name: "非公開ノートブック1",
  description: "1つ目の非公開ノートブックです",
  visibility: NotebookVisibility::Private.serialize
)
create_notebook_use_case_result_2.notebook

create_initial_note_use_case_result_1 = CreateInitialNoteUseCase.new.call(
  notebook: public_notebook_1,
  viewer:
)
note_1 = create_initial_note_use_case_result_1.note

UpdateNoteUseCase.new.call(
  viewer:,
  note: note_1,
  notebook: public_notebook_1,
  title: "公開ノート1",
  body: "1つ目の公開ノートです"
)
