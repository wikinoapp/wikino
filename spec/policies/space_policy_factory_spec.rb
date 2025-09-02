# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe SpacePolicyFactory do
  describe ".build" do
    it "space_member_recordがnilの場合はGuestPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)

      policy = SpacePolicyFactory.build(user_record:, space_member_record: nil)

      expect(policy).to be_instance_of(SpaceGuestPolicy)
    end

    it "Ownerロールの場合はOwnerPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )

      policy = SpacePolicyFactory.build(user_record:, space_member_record:)

      expect(policy).to be_instance_of(SpaceOwnerPolicy)
    end

    it "Memberロールの場合はMemberPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )

      policy = SpacePolicyFactory.build(user_record:, space_member_record:)

      expect(policy).to be_instance_of(SpaceMemberPolicy)
    end

    it "user_recordがnilでもGuestPolicyを返すこと" do
      policy = SpacePolicyFactory.build(user_record: nil, space_member_record: nil)

      expect(policy).to be_instance_of(SpaceGuestPolicy)
    end

    it "未知のロールの場合はエラーを発生させること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )

      # モックを使ってroleメソッドをオーバーライド
      allow(space_member_record).to receive(:role).and_return("unknown_role")

      expect do
        SpacePolicyFactory.build(user_record:, space_member_record:)
      end.to raise_error(ArgumentError, /Unknown role: unknown_role/)
    end

    it "Factoryから生成されたPolicyが正しく動作すること" do
      # Ownerロールのケース
      owner_user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      owner_space_member_record = FactoryBot.create(
        :space_member_record,
        user_record: owner_user_record,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )

      owner_policy = SpacePolicyFactory.build(
        user_record: owner_user_record,
        space_member_record: owner_space_member_record
      )

      # Ownerはスペース更新可能
      expect(owner_policy.can_update_space?(space_record:)).to be(true)

      # Memberロールのケース
      member_user_record = FactoryBot.create(:user_record)
      member_space_member_record = FactoryBot.create(
        :space_member_record,
        user_record: member_user_record,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )

      member_policy = SpacePolicyFactory.build(
        user_record: member_user_record,
        space_member_record: member_space_member_record
      )

      # Memberはスペース更新不可
      expect(member_policy.can_update_space?(space_record:)).to be(false)

      # Guestのケース
      guest_policy = SpacePolicyFactory.build(
        user_record: nil,
        space_member_record: nil
      )

      # Guestはスペース更新不可
      expect(guest_policy.can_update_space?(space_record:)).to be(false)
    end
  end
end
