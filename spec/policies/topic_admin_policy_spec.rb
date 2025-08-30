# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicAdminPolicy do
  describe "#can_update_topic?" do
    it "自分がAdminであるトピックのみ更新可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      other_topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 自分がAdminであるトピックは更新可能
      expect(policy.can_update_topic?(topic_record:)).to be(true)

      # 他のトピックは更新不可
      expect(policy.can_update_topic?(topic_record: other_topic_record)).to be(false)
    end

    it "非アクティブな場合は更新不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: false)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_delete_topic?" do
    it "自分がAdminであるトピックのみ削除可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      other_topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 自分がAdminであるトピックは削除可能
      expect(policy.can_delete_topic?(topic_record:)).to be(true)

      # 他のトピックは削除不可
      expect(policy.can_delete_topic?(topic_record: other_topic_record)).to be(false)
    end
  end

  describe "#can_manage_topic_members?" do
    it "自分がAdminであるトピックのメンバーのみ管理可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      other_topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 自分がAdminであるトピックのメンバーは管理可能
      expect(policy.can_manage_topic_members?(topic_record:)).to be(true)

      # 他のトピックのメンバーは管理不可
      expect(policy.can_manage_topic_members?(topic_record: other_topic_record)).to be(false)
    end
  end

  describe "#can_update_space?" do
    it "Topic Adminはスペース設定を変更できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_update_space?(space_record:)).to be(false)
    end
  end

  describe "#can_create_topic?" do
    it "アクティブなTopic Adminは新しいトピックを作成可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_create_topic?).to be(true)
    end

    it "非アクティブなTopic Adminは新しいトピックを作成できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: false)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_create_topic?).to be(false)
    end
  end

  describe "#can_export_space?" do
    it "Topic Adminはスペースをエクスポートできないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_export_space?(space_record:)).to be(false)
    end
  end
end
