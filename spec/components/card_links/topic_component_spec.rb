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

  it "トピックの説明が存在する場合に表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, description: "これはトピックの説明です")

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    expect(page).to have_text("これはトピックの説明です")
  end

  it "トピックの説明が存在しない場合に表示されないこと" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, description: "")

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    expect(page).not_to have_css(".text-gray-600")
  end

  it "トピックアイコンが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::TopicComponent.new(topic:))

    # Icons::TopicComponentが表示されることを確認
    expect(page).to have_css("svg")
  end

  it "カスタムクラスが適用されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    render_inline(CardLinks::TopicComponent.new(topic:, card_class: "custom-class"))

    expect(page).to have_css(".custom-class")
  end
end
