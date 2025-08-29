# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe GuestPolicy do
  it "userレコードがなくても初期化できること" do
    policy = GuestPolicy.new(user_record: nil)

    expect(policy).to be_instance_of(GuestPolicy)
  end

  it "userレコードありで初期化できること" do
    user_record = FactoryBot.create(:user_record)

    policy = GuestPolicy.new(user_record:)

    expect(policy).to be_instance_of(GuestPolicy)
  end

  describe "#can_update_space?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_update_space?(space_record:)).to be(false)
    end
  end

  describe "#can_update_topic?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      topic_record = FactoryBot.create(:topic_record)

      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_delete_topic?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      topic_record = FactoryBot.create(:topic_record)

      expect(policy.can_delete_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_manage_topic_members?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      topic_record = FactoryBot.create(:topic_record)

      expect(policy.can_manage_topic_members?(topic_record:)).to be(false)
    end
  end

  describe "#can_create_topic?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)

      expect(policy.can_create_topic?).to be(false)
    end
  end

  describe "#can_show_page?" do
    it "公開トピックのページは閲覧可能であること" do
      policy = GuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)
      public_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Public.serialize)
      page_record = FactoryBot.create(:page_record, space_record:, topic_record: public_topic)

      expect(policy.can_show_page?(page_record:)).to be(true)
    end

    it "非公開トピックのページは閲覧不可であること" do
      policy = GuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)
      private_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Private.serialize)
      page_record = FactoryBot.create(:page_record, space_record:, topic_record: private_topic)

      expect(policy.can_show_page?(page_record:)).to be(false)
    end
  end

  # AttachmentRecordのFactoryが複雑なため、これらのテストは省略
  # 実際の動作は結合テストで確認

  describe "#can_manage_attachments?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_manage_attachments?(space_record:)).to be(false)
    end
  end

  describe "#can_export_space?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_export_space?(space_record:)).to be(false)
    end
  end

  describe "#can_create_page?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      topic_record = FactoryBot.create(:topic_record)

      expect(policy.can_create_page?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      page_record = FactoryBot.create(:page_record)

      expect(policy.can_update_page?(page_record:)).to be(false)
    end
  end

  describe "#can_update_draft_page?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      page_record = FactoryBot.create(:page_record)

      expect(policy.can_update_draft_page?(page_record:)).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      page_record = FactoryBot.create(:page_record)

      expect(policy.can_trash_page?(page_record:)).to be(false)
    end
  end

  describe "#can_show_trash?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_show_trash?(space_record:)).to be(false)
    end
  end

  describe "#can_create_bulk_restore_pages?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)

      expect(policy.can_create_bulk_restore_pages?).to be(false)
    end
  end

  describe "#can_upload_attachment?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)
      space_record = FactoryBot.create(:space_record)

      expect(policy.can_upload_attachment?(space_record:)).to be(false)
    end
  end

  describe "#joined_space?" do
    it "常にfalseを返すこと" do
      policy = GuestPolicy.new(user_record: nil)

      expect(policy.joined_space?).to be(false)
    end
  end

  describe "#joined_topic_records" do
    it "空のコレクションを返すこと" do
      policy = GuestPolicy.new(user_record: nil)

      expect(policy.joined_topic_records.count).to eq(0)
    end
  end

  describe "#showable_topics" do
    it "公開トピックのみ閲覧可能であること" do
      policy = GuestPolicy.new(user_record: nil)
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
