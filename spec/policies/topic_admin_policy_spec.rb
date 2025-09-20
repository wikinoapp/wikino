# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicAdminPolicy do
  describe "#can_create_page?" do
    it "Topic Adminはページ作成が許可されること" do
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

      expect(policy.can_create_page?(topic_record:)).to be(true)
    end
  end

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

  describe "#can_create_edit_suggestion?" do
    it "Topic Adminは編集提案を作成できること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: TopicMemberRole::Admin.serialize)

      policy = TopicAdminPolicy.new(user_record:, space_member_record:, topic_member_record:)

      expect(policy.can_create_edit_suggestion?).to be true
    end
  end

  describe "#can_update_edit_suggestion?" do
    it "作成者は自分の編集提案を更新できること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: TopicMemberRole::Admin.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: user_record)

      policy = TopicAdminPolicy.new(user_record:, space_member_record:, topic_member_record:)

      expect(policy.can_update_edit_suggestion?(edit_suggestion_record:)).to be true
    end
  end

  describe "#can_apply_edit_suggestion?" do
    it "Topic Adminは編集提案を反映できること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: TopicMemberRole::Admin.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: user_record)

      policy = TopicAdminPolicy.new(user_record:, space_member_record:, topic_member_record:)

      expect(policy.can_apply_edit_suggestion?(edit_suggestion_record:)).to be true
    end
  end

  describe "#can_close_edit_suggestion?" do
    it "Topic Adminは編集提案をクローズできること" do
      user_record = FactoryBot.create(:user_record)
      other_user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: TopicMemberRole::Admin.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: other_user_record)

      policy = TopicAdminPolicy.new(user_record:, space_member_record:, topic_member_record:)

      expect(policy.can_close_edit_suggestion?(edit_suggestion_record:)).to be true
    end
  end

  describe "#can_comment_on_edit_suggestion?" do
    it "Topic Adminは編集提案にコメントできること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: TopicMemberRole::Admin.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: user_record)

      policy = TopicAdminPolicy.new(user_record:, space_member_record:, topic_member_record:)

      expect(policy.can_comment_on_edit_suggestion?(edit_suggestion_record:)).to be true
    end
  end
end
