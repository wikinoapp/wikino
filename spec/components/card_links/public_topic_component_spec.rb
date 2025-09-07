# typed: false
# frozen_string_literal: true

RSpec.describe CardLinks::PublicTopicComponent, type: :view do
  it "トピック名が表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, name: "公開トピック")

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::PublicTopicComponent.new(topic:))

    expect(page).to have_text("公開トピック")
    expect(page).to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}")
  end

  it "説明がある場合に説明が表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, description: "これは公開トピックの説明です")

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::PublicTopicComponent.new(topic:))

    expect(page).to have_text("これは公開トピックの説明です")
  end

  it "説明が空の場合に説明が表示されないこと" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, description: "")

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::PublicTopicComponent.new(topic:))

    expect(page).not_to have_css(".text-muted.line-clamp-2.text-xs")
  end

  it "トピックページへのリンクが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::PublicTopicComponent.new(topic:))

    expect(page).to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}")
  end

  it "ページ作成リンクが表示されないこと" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::PublicTopicComponent.new(topic:))

    expect(page).not_to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/pages/new")
    expect(page).not_to have_css("[data-icon='pencil-simple-line']")
  end

  it "設定リンクが表示されないこと" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::PublicTopicComponent.new(topic:))

    expect(page).not_to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings")
    expect(page).not_to have_css("[data-icon='gear-regular']")
  end
end
