# typed: false
# frozen_string_literal: true

RSpec.describe CardLists::TopicComponent, type: :view do
  it "メンバー向けにトピックカードのリストが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Topic 1")
    topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Topic 2")
    topic_record3 = FactoryBot.create(:topic_record, space_record:, name: "Topic 3")

    topics = [topic_record1, topic_record2, topic_record3].map do |topic_record|
      TopicRepository.new.to_model(
        topic_record:,
        can_update: false,
        can_create_page: false
      )
    end

    render_inline(CardLists::TopicComponent.new(topics:, is_guest: false))

    expect(page).to have_text("Topic 1")
    expect(page).to have_text("Topic 2")
    expect(page).to have_text("Topic 3")
    expect(page).to have_css(".grid")
  end

  it "ゲスト向けに公開トピックカードのリストが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Public Topic 1", description: "Description 1")
    topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Public Topic 2", description: "Description 2")

    topics = [topic_record1, topic_record2].map do |topic_record|
      TopicRepository.new.to_model(
        topic_record:,
        can_update: false,
        can_create_page: false
      )
    end

    render_inline(CardLists::TopicComponent.new(topics:, is_guest: true))

    expect(page).to have_text("Public Topic 1")
    expect(page).to have_text("Description 1")
    expect(page).to have_text("Public Topic 2")
    expect(page).to have_text("Description 2")
    expect(page).to have_css(".grid")
  end

  it "トピックがない場合に空のリストが表示されること" do
    render_inline(CardLists::TopicComponent.new(topics: [], is_guest: false))

    expect(page).to have_css(".grid")
    expect(page).not_to have_css("a")
  end
end
