# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "Topic Policies" do
  describe "#can_create_page?" do
    it "トピックメンバーの場合、ページ作成が許可されること" do
      # テストデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      # ポリシー実行
      policy = TopicPolicyFactory.build(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 検証
      expect(policy.can_create_page?(topic_record:)).to be(true)
    end

    it "トピック管理者の場合、ページ作成が許可されること" do
      # テストデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # ポリシー実行
      policy = TopicPolicyFactory.build(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 検証
      expect(policy.can_create_page?(topic_record:)).to be(true)
    end

    it "トピックメンバーでない場合、ページ作成が許可されないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # ポリシー実行（topic_member_record: nil）
      policy = TopicPolicyFactory.build(
        user_record:,
        space_member_record:,
        topic_member_record: nil
      )

      # 検証
      # TopicGuestPolicyのcan_create_page?はtopic_recordを必要とするが、常にfalseを返す
      expect(policy.can_create_page?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_topic?" do
    it "トピック管理者の場合、更新が許可されること" do
      # テストデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # ポリシー実行
      policy = TopicPolicyFactory.build(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 検証
      expect(policy.can_update_topic?(topic_record:)).to be(true)
    end

    it "トピック一般メンバーの場合、更新が許可されないこと" do
      # テストデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      # ポリシー実行
      policy = TopicPolicyFactory.build(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 検証
      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end

    it "トピックメンバーでない場合、更新が許可されないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # ポリシー実行（topic_member_record: nil）
      policy = TopicPolicyFactory.build(
        user_record:,
        space_member_record:,
        topic_member_record: nil
      )

      # 検証
      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end
end
