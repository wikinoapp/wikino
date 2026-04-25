# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe GuestPolicy do
  describe "#can_show_topic?" do
    it "公開トピックは閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Public.serialize)

      policy = GuestPolicy.new
      expect(policy.can_show_topic?(topic_record:)).to be(true)
    end

    it "非公開トピックは閲覧できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)

      policy = GuestPolicy.new
      expect(policy.can_show_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_topic?" do
    it "トピックを更新できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_update_topic?).to be(false)
    end
  end

  describe "#can_delete_topic?" do
    it "トピックを削除できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_delete_topic?).to be(false)
    end
  end

  describe "#can_create_page?" do
    it "ページを作成できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_create_page?).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "ページを更新できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_update_page?).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "ページをゴミ箱に移動できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_trash_page?).to be(false)
    end
  end

  describe "#can_show_draft_page?" do
    it "下書きページを閲覧できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_show_draft_page?(is_owner: true)).to be(false)
      expect(policy.can_show_draft_page?(is_owner: false)).to be(false)
    end
  end

  describe "#can_update_draft_page?" do
    it "下書きページを更新できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_update_draft_page?(is_owner: true)).to be(false)
      expect(policy.can_update_draft_page?(is_owner: false)).to be(false)
    end
  end

  describe "#can_update_space?" do
    it "スペースを更新できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_update_space?).to be(false)
    end
  end

  describe "#can_export_space?" do
    it "スペースをエクスポートできないこと" do
      policy = GuestPolicy.new
      expect(policy.can_export_space?).to be(false)
    end
  end

  describe "#can_create_topic?" do
    it "トピックを作成できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_create_topic?).to be(false)
    end
  end

  describe "#can_show_trash?" do
    it "ゴミ箱を閲覧できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_show_trash?).to be(false)
    end
  end

  describe "#can_create_bulk_restore_pages?" do
    it "一括復元できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_create_bulk_restore_pages?).to be(false)
    end
  end

  describe "#can_upload_attachment?" do
    it "ファイルアップロードできないこと" do
      policy = GuestPolicy.new
      expect(policy.can_upload_attachment?).to be(false)
    end
  end

  describe "#can_view_attachment?" do
    it "公開ページから参照されている添付ファイルは閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Public.serialize)
      page_record = FactoryBot.create(:page_record, space_record:, topic_record:)
      attachment_record = FactoryBot.create(:attachment_record, space_record:)
      FactoryBot.create(:page_attachment_reference_record, page_record:, attachment_record:)

      policy = GuestPolicy.new
      expect(policy.can_view_attachment?(attachment_record:)).to be(true)
    end

    it "公開ページから参照されていない添付ファイルは閲覧不可であること" do
      space_record = FactoryBot.create(:space_record)
      attachment_record = FactoryBot.create(:attachment_record, space_record:)

      policy = GuestPolicy.new
      expect(policy.can_view_attachment?(attachment_record:)).to be(false)
    end
  end

  describe "#can_delete_attachment?" do
    it "ファイル削除できないこと" do
      policy = GuestPolicy.new
      expect(policy.can_delete_attachment?).to be(false)
    end
  end
end
