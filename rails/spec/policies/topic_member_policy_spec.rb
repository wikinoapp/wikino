# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicMemberPolicy do
  describe "#can_update_topic?" do
    it "Topic Memberはトピックの基本情報を更新できないこと" do
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
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_delete_topic?" do
    it "Topic Memberはトピックを削除できないこと" do
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
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_delete_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_manage_topic_members?" do
    it "Topic Memberはトピックメンバーを管理できないこと" do
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
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_manage_topic_members?(topic_record:)).to be(false)
    end
  end

  describe "#can_create_page?" do
    it "Topic Memberはページ作成が許可されること" do
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
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_create_page?(topic_record:)).to be(true)
    end

    it "自分が参加しているトピックにページを作成可能であること" do
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
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 自分が参加しているトピックにはページ作成可能
      expect(policy.can_create_page?(topic_record:)).to be(true)

      # 他のトピックにはページ作成不可
      expect(policy.can_create_page?(topic_record: other_topic_record)).to be(false)
    end

    it "非アクティブな場合はページ作成不可であること" do
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
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_create_page?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "自分が参加しているトピックのページを更新可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      other_topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)
      other_page_record = FactoryBot.create(:page_record, topic_record: other_topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 自分が参加しているトピックのページは更新可能
      expect(policy.can_update_page?(page_record:)).to be(true)

      # 他のトピックのページは更新不可
      expect(policy.can_update_page?(page_record: other_page_record)).to be(false)
    end

    it "非アクティブな場合はページ更新不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: false)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_update_page?(page_record:)).to be(false)
    end
  end

  describe "#can_update_draft_page?" do
    it "can_update_page?と同じ結果を返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_update_draft_page?(page_record:)).to eq(policy.can_update_page?(page_record:))
    end
  end

  describe "#can_show_page?" do
    it "公開トピックのページは誰でも閲覧可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Public.serialize)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      # 別のトピックのメンバーでも公開トピックのページは閲覧可能
      other_topic_record = FactoryBot.create(:topic_record, space_record:)
      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: other_topic_record,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_show_page?(page_record:)).to be(true)
    end

    it "非公開トピックのページは参加しているメンバーのみ閲覧可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Private.serialize)
      other_topic_record = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Private.serialize)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)
      other_page_record = FactoryBot.create(:page_record, topic_record: other_topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 自分が参加しているトピックのページは閲覧可能
      expect(policy.can_show_page?(page_record:)).to be(true)

      # 参加していないトピックのページは閲覧不可
      expect(policy.can_show_page?(page_record: other_page_record)).to be(false)
    end

    it "非アクティブな場合は非公開トピックのページ閲覧不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Private.serialize)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: false)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_show_page?(page_record:)).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "自分が参加しているトピックのページをゴミ箱に移動可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      other_topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)
      other_page_record = FactoryBot.create(:page_record, topic_record: other_topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      # 自分が参加しているトピックのページはゴミ箱移動可能
      expect(policy.can_trash_page?(page_record:)).to be(true)

      # 他のトピックのページはゴミ箱移動不可
      expect(policy.can_trash_page?(page_record: other_page_record)).to be(false)
    end

    it "非アクティブな場合はページをゴミ箱に移動不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: false)

      topic_member_record = FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      policy = TopicMemberPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )

      expect(policy.can_trash_page?(page_record:)).to be(false)
    end
  end
end
