# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe SpaceMemberPolicyFactory do
  describe ".build" do
    it "space_member_recordがnilの場合はGuestPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)

      policy = SpaceMemberPolicyFactory.build(user_record:, space_member_record: nil)

      expect(policy).to be_instance_of(GuestPolicy)
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

      policy = SpaceMemberPolicyFactory.build(user_record:, space_member_record:)

      expect(policy).to be_instance_of(OwnerPolicy)
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

      policy = SpaceMemberPolicyFactory.build(user_record:, space_member_record:)

      expect(policy).to be_instance_of(MemberPolicy)
    end

    it "user_recordがnilでもGuestPolicyを返すこと" do
      policy = SpaceMemberPolicyFactory.build(user_record: nil, space_member_record: nil)

      expect(policy).to be_instance_of(GuestPolicy)
    end
  end
end

