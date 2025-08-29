# typed: false
# frozen_string_literal: true

require "rails_helper"

# BaseMemberPolicyは抽象クラスなので、テスト用の具象クラスを定義
class TestMemberPolicy < BaseMemberPolicy
end

RSpec.describe BaseMemberPolicy do

  it "user_recordとspace_member_recordを受け取って初期化できること" do
    user = FactoryBot.create(:user_record)
    space = FactoryBot.create(:space_record)
    space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space)

    policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

    expect(policy).to be_instance_of(TestMemberPolicy)
  end

  it "user_recordとspace_member_recordの関連性が不一致の場合はエラーになること" do
    user = FactoryBot.create(:user_record)
    other_user = FactoryBot.create(:user_record)
    space = FactoryBot.create(:space_record)
    space_member = FactoryBot.create(:space_member_record, user_record: other_user, space_record: space)

    expect do
      TestMemberPolicy.new(user_record: user, space_member_record: space_member)
    end.to raise_error(ArgumentError, /Mismatched relations/)
  end

  it "space_member_recordがnilの場合は初期化できること" do
    user = FactoryBot.create(:user_record)

    policy = TestMemberPolicy.new(user_record: user, space_member_record: nil)

    expect(policy).to be_instance_of(TestMemberPolicy)
  end

  describe "#joined_space?" do
    it "space_member_recordが存在する場合はtrueを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      expect(policy.joined_space?).to be(true)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: nil)

      expect(policy.joined_space?).to be(false)
    end
  end

  describe "#in_same_space?" do
    it "同じスペースに所属している場合はtrueを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      expect(policy.in_same_space?(space_record_id: space.id)).to be(true)
    end

    it "異なるスペースIDの場合はfalseを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      other_space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      expect(policy.in_same_space?(space_record_id: other_space.id)).to be(false)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: nil)

      expect(policy.in_same_space?(space_record_id: "any_space_id")).to be(false)
    end
  end

  describe "#active?" do
    it "activeなメンバーの場合はtrueを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space, active: true)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      expect(policy.active?).to be(true)
    end

    it "非activeなメンバーの場合はfalseを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space, active: false)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      expect(policy.active?).to be(false)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: nil)

      expect(policy.active?).to be(false)
    end
  end

  describe "#joined_topic?" do
    it "トピックに参加している場合はtrueを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space)
      topic = FactoryBot.create(:topic_record, space_record: space)
      FactoryBot.create(:topic_member_record, space_member_record: space_member, topic_record: topic)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      expect(policy.joined_topic?(topic_record_id: topic.id)).to be(true)
    end

    it "トピックに参加していない場合はfalseを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space)
      topic = FactoryBot.create(:topic_record, space_record: space)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      expect(policy.joined_topic?(topic_record_id: topic.id)).to be(false)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: nil)

      expect(policy.joined_topic?(topic_record_id: "any_topic_id")).to be(false)
    end
  end

  describe "#joined_topic_records" do
    it "参加しているトピックのレコードを返すこと" do
      user = FactoryBot.create(:user_record)
      space = FactoryBot.create(:space_record)
      space_member = FactoryBot.create(:space_member_record, user_record: user, space_record: space)
      topic1 = FactoryBot.create(:topic_record, space_record: space)
      topic2 = FactoryBot.create(:topic_record, space_record: space)
      FactoryBot.create(:topic_member_record, space_member_record: space_member, topic_record: topic1)
      FactoryBot.create(:topic_member_record, space_member_record: space_member, topic_record: topic2)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: space_member)

      topic_records = policy.joined_topic_records
      expect(topic_records).to include(topic1)
      expect(topic_records).to include(topic2)
      expect(topic_records.count).to eq(2)
    end

    it "space_member_recordがnilの場合はnilを返すこと" do
      user = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record: user, space_member_record: nil)

      expect(policy.joined_topic_records).to be_nil
    end
  end
end
