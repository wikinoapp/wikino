# typed: strict

# create_account_use_case_result = AccountService::Create.new.call(
#   email: "user_1@example.com",
#   atname: "user_1",
#   locale: Locale::Ja,
#   password: "password",
#   time_zone: "Asia/Tokyo"
# )
# Current.viewer = create_account_use_case_result.user

# create_topic_use_case_result_1 = TopicService::Create.new.call(
#   name: "公開トピック1",
#   description: "1つ目の公開トピックです",
#   visibility: TopicVisibility::Public.serialize
# )
# public_topic_1 = create_topic_use_case_result_1.topic

# create_topic_use_case_result_2 = TopicService::Create.new.call(
#   name: "非公開トピック1",
#   description: "1つ目の非公開トピックです",
#   visibility: TopicVisibility::Private.serialize
# )
# create_topic_use_case_result_2.topic

# create_blanked_page_use_case_result_1 = PageService::CreateBlanked.new.call(
#   topic: public_topic_1
# )
# page_1 = create_blanked_page_use_case_result_1.page

# PageService::Update.new.call(
#   page: page_1,
#   topic: public_topic_1,
#   title: "公開ページ1",
#   body: "1つ目の公開ページです"
# )
