# typed: false
# frozen_string_literal: true

RSpec.describe CardLinks::TopicComponent, type: :view do
  it "トピックカードが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    expect(page).to have_text(topic.name)
    expect(page).to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}")
  end

  it "権限がある場合にページ作成リンクが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: true
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    expect(page).to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/pages/new")
  end

  it "権限がない場合にページ作成リンクが非アクティブになること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, visibility: "private")

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    expect(page).not_to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/pages/new")
    expect(page).to have_css(".opacity-50.cursor-not-allowed")
  end

  it "権限がある場合に設定リンクが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: true,
      can_create_page: true
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    expect(page).to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings")
  end

  it "権限がない場合に設定リンクが非アクティブになること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    expect(page).not_to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings")
  end
end