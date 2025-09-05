# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicRepository do
  describe "#find_topics_by_space" do
    it "スペースに参加しているトピックをlast_page_modified_atの降順で返すこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)

      # 3つのトピックを作成
      topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Topic 1")
      topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Topic 2")
      topic_record3 = FactoryBot.create(:topic_record, space_record:, name: "Topic 3")

      # トピックメンバーを作成（last_page_modified_atを設定）
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize,
        last_page_modified_at: 3.days.ago)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record2,
        space_member_record:,
        role: TopicMemberRole::Member.serialize,
        last_page_modified_at: 1.day.ago)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record3,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize,
        last_page_modified_at: 2.days.ago)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, current_user_record: user_record)

      # 検証
      expect(topics.size).to eq(3)
      expect(topics[0].name).to eq("Topic 2") # 1日前（最新）
      expect(topics[1].name).to eq("Topic 3") # 2日前
      expect(topics[2].name).to eq("Topic 1") # 3日前（最古）
    end

    it "current_user_recordがnilの場合、空配列を返すこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_record:, topic_record:, space_member_record:)

      # メソッド実行（current_user_record: nil）
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, current_user_record: nil)

      # 検証
      expect(topics).to be_empty
    end

    it "ユーザーが参加していないトピックは返さないこと" do
      # テスト用のデータ作成
      user_record1 = FactoryBot.create(:user_record)
      user_record2 = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record1 = FactoryBot.create(:space_member_record, :member, user_record: user_record1, space_record:)
      space_member_record2 = FactoryBot.create(:space_member_record, :member, user_record: user_record2, space_record:)

      # 3つのトピックを作成
      topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Topic 1")
      topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Topic 2")
      topic_record3 = FactoryBot.create(:topic_record, space_record:, name: "Topic 3")

      # user_record1は topic1とtopic2に参加
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record: space_member_record1)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record2,
        space_member_record: space_member_record1)

      # user_record2はtopic3のみに参加
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record3,
        space_member_record: space_member_record2)

      # user_record1でメソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, current_user_record: user_record1)

      # 検証: user_record1が参加している2つのトピックのみが返される
      expect(topics.size).to eq(2)
      expect(topics.map(&:name)).to contain_exactly("Topic 1", "Topic 2")
    end

    it "ユーザーがスペースメンバーでない場合、空配列を返すこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:topic_record, space_record:)

      # user_recordはspace_memberではない

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, current_user_record: user_record)

      # 検証
      expect(topics).to be_empty
    end

    it "トピック管理者の場合、can_updateがtrueになること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, current_user_record: user_record)

      # 検証
      expect(topics.size).to eq(1)
      topic = topics.first
      expect(topic.can_update).to be(true)
      expect(topic.can_create_page).to be(true)
    end

    it "トピック一般メンバーの場合、can_updateがfalse、can_create_pageがtrueになること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, current_user_record: user_record)

      # 検証
      expect(topics.size).to eq(1)
      topic = topics.first
      expect(topic.can_update).to be(false)
      expect(topic.can_create_page).to be(true)
    end

    it "last_page_modified_atがnilのトピックは最後に表示されること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)

      topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Topic with nil")
      topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Topic with date")

      # 1つ目のトピックメンバーはlast_page_modified_atがnil
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record:,
        last_page_modified_at: nil)
      # 2つ目のトピックメンバーはlast_page_modified_atが設定されている
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record2,
        space_member_record:,
        last_page_modified_at: 1.day.ago)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, current_user_record: user_record)

      # 検証
      expect(topics.size).to eq(2)
      expect(topics[0].name).to eq("Topic with date")
      expect(topics[1].name).to eq("Topic with nil")
    end
  end
end
