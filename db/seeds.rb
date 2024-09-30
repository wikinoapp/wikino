# typed: strict

create_account_use_case_result = CreateAccountUseCase.new.call(
  space_identifier: "space_1",
  email: "user_1@example.com",
  atname: "user_1",
  locale: UserLocale::Ja,
  password: "password",
  time_zone: "Asia/Tokyo"
)
viewer = create_account_use_case_result.user

create_topic_use_case_result_1 = CreateTopicUseCase.new.call(
  viewer:,
  name: "公開トピック1",
  description: "1つ目の公開トピックです",
  visibility: TopicVisibility::Public.serialize
)
public_topic_1 = create_topic_use_case_result_1.topic

create_topic_use_case_result_2 = CreateTopicUseCase.new.call(
  viewer:,
  name: "非公開トピック1",
  description: "1つ目の非公開トピックです",
  visibility: TopicVisibility::Private.serialize
)
create_topic_use_case_result_2.topic

create_initial_page_use_case_result_1 = CreateInitialPageUseCase.new.call(
  topic: public_topic_1,
  viewer:
)
page_1 = create_initial_page_use_case_result_1.page

UpdatePageUseCase.new.call(
  viewer:,
  page: page_1,
  topic: public_topic_1,
  title: "公開ページ1",
  body: "1つ目の公開ページです"
)
