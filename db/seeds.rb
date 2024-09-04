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

create_list_use_case_result_1 = CreateListUseCase.new.call(
  viewer:,
  name: "公開リスト1",
  description: "1つ目の公開リストです",
  visibility: ListVisibility::Public.serialize
)
public_list_1 = create_list_use_case_result_1.list

create_list_use_case_result_2 = CreateListUseCase.new.call(
  viewer:,
  name: "非公開リスト1",
  description: "1つ目の非公開リストです",
  visibility: ListVisibility::Private.serialize
)
private_list_1 = create_list_use_case_result_2.list

create_initial_note_use_case_result_1 = CreateInitialNoteUseCase.new.call(
  list: public_list_1,
  viewer:
)
note_1 = create_initial_note_use_case_result_1.note

UpdateNoteUseCase.new.call(
  viewer:,
  note: note_1,
  list: public_list_1,
  title: "公開ノート1",
  body: "1つ目の公開ノートです",
)
