# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe PermissionResolver do
  describe "#resolve" do
    it "Space Ownerの場合、OwnerPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(OwnerPolicy)
    end

    it "Space Memberの場合、MemberPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(MemberPolicy)
    end

    it "非メンバーの場合、GuestPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(GuestPolicy)
    end

    it "ユーザーがnilの場合、GuestPolicyを返すこと" do
      space_record = FactoryBot.create(:space_record)

      resolver = PermissionResolver.new(
        user_record: nil,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(GuestPolicy)
    end

    it "Topic Adminの場合、OwnerPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve

      expect(policy).to be_a(OwnerPolicy)
    end

    it "Topic Memberの場合、MemberPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve

      expect(policy).to be_a(MemberPolicy)
    end

    it "Space OwnerはTopic権限よりも優先されること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)
      # Topic MemberであってもSpace Ownerが優先される
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve

      # Space Ownerの権限が優先される
      expect(policy).to be_a(OwnerPolicy)
    end
  end

  describe "#resolve_for_topic" do
    it "topic_recordがnilの場合、通常のresolveと同じ結果を返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve_for_topic

      expect(policy).to be_a(MemberPolicy)
    end

    it "topic_recordが指定されている場合、Topic権限を考慮すること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve_for_topic

      expect(policy).to be_a(OwnerPolicy)
    end
  end
end
