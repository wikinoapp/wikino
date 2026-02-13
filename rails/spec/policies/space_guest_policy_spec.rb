# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe SpaceGuestPolicy do
  it "userレコードがなくても初期化できること" do
    policy = SpaceGuestPolicy.new(user_record: nil)

    expect(policy).to be_instance_of(SpaceGuestPolicy)
  end

  it "userレコードありで初期化できること" do
    user_record = FactoryBot.create(:user_record)

    policy = SpaceGuestPolicy.new(user_record:)

    expect(policy).to be_instance_of(SpaceGuestPolicy)
  end

  describe "#can_update_space?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_update_space?(space_record:)).to be(false)
    end
  end

  describe "#can_create_topic?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)

      expect(policy.can_create_topic?).to be(false)
    end
  end

  # AttachmentRecordのFactoryが複雑なため、これらのテストは省略
  # 実際の動作は結合テストで確認

  describe "#can_manage_attachments?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_manage_attachments?(space_record:)).to be(false)
    end
  end

  describe "#can_export_space?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_export_space?(space_record:)).to be(false)
    end
  end

  describe "#can_show_trash?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_show_trash?(space_record:)).to be(false)
    end
  end

  describe "#can_create_bulk_restore_pages?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)

      expect(policy.can_create_bulk_restore_pages?).to be(false)
    end
  end

  describe "#can_upload_attachment?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_upload_attachment?(space_record:)).to be(false)
    end
  end

  describe "#joined_space?" do
    it "常にfalseを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)

      expect(policy.joined_space?).to be(false)
    end
  end

  describe "#joined_topic_records" do
    it "空のコレクションを返すこと" do
      policy = SpaceGuestPolicy.new(user_record: nil)

      expect(policy.joined_topic_records.count).to eq(0)
    end
  end

  describe "#showable_topics" do
    it "公開トピックのみ閲覧可能であること" do
      policy = SpaceGuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)
      public_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Public.serialize)
      private_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Private.serialize)

      topics = policy.showable_topics(space_record:)
      expect(topics).to include(public_topic)
      expect(topics).not_to include(private_topic)
    end
  end

  # showable_pagesの詳細なテストは複雑なscopeの動作に依存するため
  # 結合テストで確認することとする
end
