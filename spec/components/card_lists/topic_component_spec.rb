# typed: false
# frozen_string_literal: true

RSpec.describe CardLists::TopicComponent, type: :view do
  it "トピックカードのリストが表示されること" do
    user_record = FactoryBot.create(:user_record)
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

    render_inline(CardLists::TopicComponent.new(
      topics:,
      current_user_record: user_record
    ))

    expect(page).to have_text("Topic 1")
    expect(page).to have_text("Topic 2")
    expect(page).to have_text("Topic 3")
    expect(page).to have_css(".grid")
  end

  it "トピックがない場合に空のリストが表示されること" do
    user_record = FactoryBot.create(:user_record)

    render_inline(CardLists::TopicComponent.new(
      topics: [],
      current_user_record: user_record
    ))

    expect(page).to have_css(".grid")
    expect(page).not_to have_css("a")
  end
end