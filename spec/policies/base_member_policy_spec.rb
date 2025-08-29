# typed: false
# frozen_string_literal: true

require "rails_helper"

# BaseMemberPolicyは抽象クラスなので、テスト用の具象クラスを定義
class TestMemberPolicy < BaseMemberPolicy
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    true
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    true
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    true
  end

  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    true
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    true
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    true
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    true
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    true
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    true
  end

  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    true
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    true
  end

  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    true
  end

  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    true
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    true
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    true
  end

  sig { override.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    T.cast(TopicRecord.none, TopicRecord::PrivateAssociationRelation)
  end

  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    T.cast(PageRecord.none, PageRecord::PrivateAssociationRelation)
  end

  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    false
  end

  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    false
  end
end

RSpec.describe BaseMemberPolicy do
  it "user_recordとspace_member_recordを受け取って初期化できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)

    policy = TestMemberPolicy.new(user_record:, space_member_record:)

    expect(policy).to be_instance_of(TestMemberPolicy)
  end

  it "user_recordとspace_member_recordの関連性が不一致の場合はエラーになること" do
    user_record = FactoryBot.create(:user_record)
    other_user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record: other_user_record, space_record:)

    expect do
      TestMemberPolicy.new(user_record:, space_member_record:)
    end.to raise_error(ArgumentError, /Mismatched relations/)
  end

  it "space_member_recordがnilの場合は初期化できること" do
    user_record = FactoryBot.create(:user_record)

    policy = TestMemberPolicy.new(user_record:, space_member_record: nil)

    expect(policy).to be_instance_of(TestMemberPolicy)
  end

  describe "#joined_space?" do
    it "space_member_recordが存在する場合はtrueを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.joined_space?).to be(true)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record:, space_member_record: nil)

      expect(policy.joined_space?).to be(false)
    end
  end

  describe "#in_same_space?" do
    it "同じスペースに所属している場合はtrueを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.in_same_space?(space_record_id: space_record.id)).to be(true)
    end

    it "異なるスペースIDの場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.in_same_space?(space_record_id: other_space_record.id)).to be(false)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record:, space_member_record: nil)

      expect(policy.in_same_space?(space_record_id: "any_space_id")).to be(false)
    end
  end

  describe "#active?" do
    it "activeなメンバーの場合はtrueを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.active?).to be(true)
    end

    it "非activeなメンバーの場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: false)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.active?).to be(false)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record:, space_member_record: nil)

      expect(policy.active?).to be(false)
    end
  end

  describe "#joined_topic?" do
    it "トピックに参加している場合はtrueを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record:)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.joined_topic?(topic_record_id: topic_record.id)).to be(true)
    end

    it "トピックに参加していない場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.joined_topic?(topic_record_id: topic_record.id)).to be(false)
    end

    it "space_member_recordがnilの場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record:, space_member_record: nil)

      expect(policy.joined_topic?(topic_record_id: "any_topic_id")).to be(false)
    end
  end

  describe "#joined_topic_records" do
    it "参加しているトピックのレコードを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
      topic_record1 = FactoryBot.create(:topic_record, space_record:)
      topic_record2 = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record: topic_record1)
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record: topic_record2)

      policy = TestMemberPolicy.new(user_record:, space_member_record:)

      topic_records = policy.joined_topic_records
      expect(topic_records).to include(topic_record1)
      expect(topic_records).to include(topic_record2)
      expect(topic_records.count).to eq(2)
    end

    it "space_member_recordがnilの場合は空のコレクションを返すこと" do
      user_record = FactoryBot.create(:user_record)

      policy = TestMemberPolicy.new(user_record:, space_member_record: nil)

      expect(policy.joined_topic_records.to_a).to eq([])
    end
  end
end
