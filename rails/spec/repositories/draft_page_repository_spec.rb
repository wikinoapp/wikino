# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe DraftPageRepository do
  describe "#find_for_sidebar" do
    it "ユーザーの下書きページをmodified_atの降順で返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      page_record1 = FactoryBot.create(:page_record, space_record:, topic_record:, title: "Page 1")
      page_record2 = FactoryBot.create(:page_record, space_record:, topic_record:, title: "Page 2")
      page_record3 = FactoryBot.create(:page_record, space_record:, topic_record:, title: "Page 3")

      FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record: page_record1, space_member_record:, modified_at: 3.days.ago)
      FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record: page_record2, space_member_record:, modified_at: 1.day.ago)
      FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record: page_record3, space_member_record:, modified_at: 2.days.ago)

      repository = DraftPageRepository.new
      result = repository.find_for_sidebar(user_record:, limit: 5)

      expect(result[:draft_pages].size).to eq(3)
      expect(result[:draft_pages][0].page.title).to eq("Page 2")
      expect(result[:draft_pages][1].page.title).to eq("Page 3")
      expect(result[:draft_pages][2].page.title).to eq("Page 1")
      expect(result[:has_more]).to be(false)
    end

    it "limitを超える場合has_moreがtrueになること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      3.times do |i|
        page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "Page #{i + 1}")
        FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record:, space_member_record:, modified_at: (3 - i).days.ago)
      end

      repository = DraftPageRepository.new
      result = repository.find_for_sidebar(user_record:, limit: 2)

      expect(result[:draft_pages].size).to eq(2)
      expect(result[:has_more]).to be(true)
    end

    it "limit以下の場合has_moreがfalseになること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      page_record = FactoryBot.create(:page_record, space_record:, topic_record:)
      FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record:, space_member_record:)

      repository = DraftPageRepository.new
      result = repository.find_for_sidebar(user_record:, limit: 5)

      expect(result[:draft_pages].size).to eq(1)
      expect(result[:has_more]).to be(false)
    end

    it "非アクティブなスペースメンバーの下書きは含まれないこと" do
      user_record = FactoryBot.create(:user_record)
      active_space_record = FactoryBot.create(:space_record)
      inactive_space_record = FactoryBot.create(:space_record)

      active_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record: active_space_record, active: true)
      inactive_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record: inactive_space_record, active: false)

      active_topic_record = FactoryBot.create(:topic_record, space_record: active_space_record)
      inactive_topic_record = FactoryBot.create(:topic_record, space_record: inactive_space_record)

      active_page_record = FactoryBot.create(:page_record, space_record: active_space_record, topic_record: active_topic_record, title: "Active")
      inactive_page_record = FactoryBot.create(:page_record, space_record: inactive_space_record, topic_record: inactive_topic_record, title: "Inactive")

      FactoryBot.create(:draft_page_record, space_record: active_space_record, topic_record: active_topic_record, page_record: active_page_record, space_member_record: active_member_record)
      FactoryBot.create(:draft_page_record, space_record: inactive_space_record, topic_record: inactive_topic_record, page_record: inactive_page_record, space_member_record: inactive_member_record)

      repository = DraftPageRepository.new
      result = repository.find_for_sidebar(user_record:, limit: 5)

      expect(result[:draft_pages].size).to eq(1)
      expect(result[:draft_pages][0].page.title).to eq("Active")
    end

    it "他のユーザーの下書きは含まれないこと" do
      user_record = FactoryBot.create(:user_record)
      other_user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)

      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      other_space_member_record = FactoryBot.create(:space_member_record, :member, user_record: other_user_record, space_record:)

      topic_record = FactoryBot.create(:topic_record, space_record:)

      my_page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "My Page")
      other_page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "Other Page")

      FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record: my_page_record, space_member_record:)
      FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record: other_page_record, space_member_record: other_space_member_record)

      repository = DraftPageRepository.new
      result = repository.find_for_sidebar(user_record:, limit: 5)

      expect(result[:draft_pages].size).to eq(1)
      expect(result[:draft_pages][0].page.title).to eq("My Page")
    end

    it "下書きがない場合空配列を返すこと" do
      user_record = FactoryBot.create(:user_record)

      repository = DraftPageRepository.new
      result = repository.find_for_sidebar(user_record:, limit: 5)

      expect(result[:draft_pages]).to be_empty
      expect(result[:has_more]).to be(false)
    end
  end
end
